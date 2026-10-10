package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/handlers"
	"github.com/Infinite-Locus-Product/coca_cola_backend/middlewares"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

// Phone-number fixtures in this file must be globally distinct across every
// test function, not just within one. matchUser resolves a number to "the
// first matching users row" with no way to disambiguate, so two tests
// seeding the same number can match each other's user. Today they only get
// away with it because `go test` happens to run them serially in source
// order and each one's t.Cleanup fires before the next could collide -
// which breaks the moment someone runs a subset with -run or adds
// t.Parallel(). Convention: 9998880001..9998880009, one per seeded user, no
// digit sequence reused anywhere in this file.

// seedLocation creates a minimal State -> City -> PinCode chain for the given
// pincode value, since a DSR "update" request's address field now supplies
// only the pincode (not the internal pin_code_id/city_id/state_id row IDs an
// external privacy portal has no way of knowing) and the service resolves
// those IDs by looking up a real PinCode row. Registers cleanup.
func seedLocation(t *testing.T, db *gorm.DB, pincode int) {
	t.Helper()
	state := &entities.State{
		Name: fmt.Sprintf("Test State %d", pincode), ShortName: "TS", CountryID: 1,
		IsActive: true, CreatedBy: "test", CreatedOn: time.Now(),
	}
	if err := db.Create(state).Error; err != nil {
		t.Fatalf("failed to seed state for pincode %d: %v", pincode, err)
	}
	city := &entities.City{
		Name: fmt.Sprintf("Test City %d", pincode), StateID: state.ID,
		IsActive: true, CreatedBy: "test", CreatedOn: time.Now(),
	}
	if err := db.Create(city).Error; err != nil {
		t.Fatalf("failed to seed city for pincode %d: %v", pincode, err)
	}
	pinCode := &entities.PinCode{
		Pincode: pincode, CityID: city.ID, IsDeliverable: true, DeliveryCharges: 0,
		IsActive: true, CreatedBy: "test", CreatedOn: time.Now(),
	}
	if err := db.Table("pin_code").Create(pinCode).Error; err != nil {
		t.Fatalf("failed to seed pin_code %d: %v", pincode, err)
	}
	t.Cleanup(func() {
		db.Table("pin_code").Delete(&entities.PinCode{}, pinCode.ID)
		db.Delete(&entities.City{}, city.ID)
		db.Delete(&entities.State{}, state.ID)
	})
}

func TestDSRRequestRepository_CreateAndFind(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()

	ctx := context.Background()
	tx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn failed: %v", err)
	}
	defer txnManager.RollbackOnPanic(tx)

	record := &entities.DSRRequest{
		RequestType:          "access",
		RequesterFirstName:   "Smoke",
		RequesterLastName:    "Test",
		RequesterPhoneNumber: "9998887772",
		Status:               "not_found",
	}
	if err := dsrRequestRepo.Create(ctx, tx, record); err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("Create failed: %v", err)
	}
	if record.ID == "" {
		txnManager.AbortTxn(tx)
		t.Fatalf("expected a generated ID after Create")
	}

	found, err := dsrRequestRepo.FindById(ctx, tx, record.ID)
	if err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("FindById failed: %v", err)
	}
	if found.RequesterPhoneNumber != "9998887772" {
		txnManager.AbortTxn(tx)
		t.Fatalf("expected requester_phone_number to round-trip, got %q", found.RequesterPhoneNumber)
	}

	txnManager.CommitTxn(tx)
	t.Logf("dsr_requests round-trip OK, id=%s", record.ID)
}

func TestDSRService_HandleRequest_NotFoundAndValidation(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService("", "") // unused by this test's code paths
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)

	ctx := context.Background()

	// Phone number guaranteed not to exist.
	resp, err := dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "access",
		FirstName:   "Nobody",
		LastName:    "Here",
		PhoneNumber: "0000000000",
	})
	if err != nil {
		t.Fatalf("expected no error for not_found outcome, got: %v", err)
	}
	if resp.Status != "not_found" {
		t.Fatalf("expected status=not_found, got %q", resp.Status)
	}
	if resp.RequestID == "" {
		t.Fatalf("expected a request_id even for not_found")
	}

	// Update with an unwhitelisted field must be rejected before touching the DB.
	_, err = dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update",
		FirstName:   "Someone",
		LastName:    "Else",
		PhoneNumber: "0000000000",
		Fields:      map[string]interface{}{"wallet_balance": "999999"},
	})
	if err == nil {
		t.Fatalf("expected an error for an unwhitelisted update field")
	}
	t.Logf("unwhitelisted field correctly rejected: %v", err)
}

// TestDSRService_HandleRequest_FailedAuditSurvivesRollback covers the bug
// fixed in review: a "failed" dsr_requests audit row must durably persist
// even though the business transaction that produced the failure gets
// rolled back. It seeds a real user so matchUser succeeds, drives a request
// that fails inside its handler, and verifies - via a fresh query against
// the DB, not the service's own transaction - that a "failed" row with the
// right matched_user_id actually landed in dsr_requests.
//
// The failure is triggered deterministically and entirely in-process: an
// "update" request whose "dob" value cannot be parsed by time.Parse. "dob"
// is on the updatable whitelist, so the request clears
// validateUpdateFields and matchUser, the business transaction is opened,
// and only then does handleUpdate's dob case reject it - exactly the class
// of mid-transaction handler failure this test exists to exercise.
//
// It deliberately does NOT drive an "access" request any more. handleAccess
// used to be a stub that always errored, but it is a real working
// implementation now, so an access request would only "fail" when the
// environment's GCS credentials happen to be missing - making the test pass
// by accident of its surroundings rather than by the condition it asserts,
// and silently stop guarding this bug as soon as GCS access works.
func TestDSRService_HandleRequest_FailedAuditSurvivesRollback(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	userRepo := repository.NewUserRepository()
	gcsService, err := utils.NewGCSService("", "") // unused by this test's code paths
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)

	ctx := context.Background()

	phone := "9998880001"
	user := &entities.User{
		PhoneNumber:           phone,
		IsActive:              true,
		IsProfileComplete:     true,
		ProfilePercentage:     50,
		RegisteredOn:          time.Now(),
		LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, db, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", user.ID)
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	// An unparseable dob fails inside handleUpdate, after the business
	// transaction is already open. No external service is involved, so this
	// failure is deterministic in every environment.
	_, err = dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update",
		FirstName:   "Someone",
		LastName:    "Real",
		PhoneNumber: phone,
		Fields:      map[string]interface{}{"dob": "not-a-real-date"},
	})
	if err == nil {
		t.Fatalf("expected an error for an unparseable dob value")
	}
	t.Logf("handleUpdate correctly failed mid-transaction: %v", err)

	// Reload from the DB directly (not via the aborted service transaction)
	// to confirm the failure audit row actually persisted.
	var persisted entities.DSRRequest
	if err := db.Where("matched_user_id = ?", user.ID).Order("requested_on desc").First(&persisted).Error; err != nil {
		t.Fatalf("expected a failed dsr_requests row to be persisted, but query failed: %v", err)
	}
	if persisted.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", persisted.Status)
	}
	if persisted.MatchedUserID == nil || *persisted.MatchedUserID != user.ID {
		t.Fatalf("expected matched_user_id=%s, got %v", user.ID, persisted.MatchedUserID)
	}
	if persisted.ErrorMessage == nil || *persisted.ErrorMessage == "" {
		t.Fatalf("expected a non-empty error_message on the failed audit row")
	}
	t.Logf("failed audit row correctly persisted: id=%s error_message=%q", persisted.ID, *persisted.ErrorMessage)
}

func TestDSRService_HandleRequest_Delete_AnonymizesUser(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService("", "")
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)
	userRepo := repository.NewUserRepository()

	ctx := context.Background()
	tx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn failed: %v", err)
	}

	phone := "9998880002"
	name := "Delete Me"
	email := "deleteme@example.com"
	user := &entities.User{
		Name:                  &name,
		Email:                 &email,
		PhoneNumber:           phone,
		IsActive:              true,
		RegisteredOn:          time.Now(),
		ProfilePercentage:     0,
		LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, tx, user); err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed user: %v", err)
	}

	if err := tx.Create(&entities.Address{
		UserID: user.ID, Address1: "123 Real St", Pincode: 400001,
		PinCodeID: 1, CityID: 1, StateID: 1, CreatedBy: "test", CreatedOn: time.Now(),
	}).Error; err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed address: %v", err)
	}

	if err := tx.Create(&entities.OTPLogs{
		MobileNo: phone, OTP: "123456", CreatedBy: "test", CreatedOn: time.Now(), LastModifiedBy: "test",
	}).Error; err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed otp log: %v", err)
	}

	if err := tx.Create(&entities.UserRefreshToken{
		UserID: user.ID, CreatedOn: time.Now(), CreatedBy: "test",
	}).Error; err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed refresh token: %v", err)
	}

	txnManager.CommitTxn(tx)

	// This test permanently mutates its seeded rows (the whole point: the
	// users row is redacted and the user_address row becomes "[REDACTED]"),
	// so without cleanup every run leaks a redacted user, a redacted address
	// and a dsr_requests row into the shared local Postgres. Deleted in
	// FK-safe order: children first, users last.
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_refresh_tokens WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM user_address WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM otp_logs WHERE mobile_no = ?", phone)
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", user.ID)
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	resp, err := dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "delete",
		FirstName:   "Delete",
		LastName:    "Me",
		PhoneNumber: phone,
	})
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", resp.Status)
	}
	if len(resp.AnonymizedTables) == 0 {
		t.Fatalf("expected anonymized_tables to be non-empty")
	}
	t.Logf("anonymized tables: %v", resp.AnonymizedTables)

	verifyTx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn (verify) failed: %v", err)
	}
	defer txnManager.CommitTxn(verifyTx)

	var reloaded entities.User
	if err := verifyTx.First(&reloaded, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.Name != nil {
		t.Errorf("expected name to be redacted, got %v", *reloaded.Name)
	}
	if reloaded.Email != nil {
		t.Errorf("expected email to be redacted, got %v", *reloaded.Email)
	}
	if reloaded.IsActive {
		t.Errorf("expected is_active=false after delete")
	}
	if reloaded.PhoneNumber == phone {
		t.Errorf("expected phone_number to be replaced with a redacted placeholder")
	}

	var otpCount int64
	verifyTx.Model(&entities.OTPLogs{}).Where("mobile_no = ?", phone).Count(&otpCount)
	if otpCount != 0 {
		t.Errorf("expected otp_logs for %s to be hard-deleted, found %d", phone, otpCount)
	}

	var refreshToken entities.UserRefreshToken
	if err := verifyTx.Where("user_id = ?", user.ID).First(&refreshToken).Error; err != nil {
		t.Fatalf("failed to reload refresh token: %v", err)
	}
	if !refreshToken.IsDeleted {
		t.Errorf("expected user_refresh_tokens.is_deleted=true after delete")
	}
}

func TestDSRService_HandleRequest_Update(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService("", "")
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)
	userRepo := repository.NewUserRepository()

	ctx := context.Background()
	tx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn failed: %v", err)
	}
	phone := "9998880003"
	user := &entities.User{
		PhoneNumber: phone, IsActive: true, RegisteredOn: time.Now(),
		ProfilePercentage: 0, LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, tx, user); err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed user: %v", err)
	}
	txnManager.CommitTxn(tx)
	// This test's phone numbers are hardcoded and matchUser's lookup is a
	// bare `ORDER BY id LIMIT 1` with no way to disambiguate duplicates, so a
	// stale row left behind by a previous run of this same test (this local
	// Postgres instance persists across test invocations) would otherwise be
	// matched instead of the row seeded by *this* run, non-deterministically
	// failing the assertions below. Clean up so reruns are reliable.
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_address WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", user.ID)
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	seedLocation(t, db, 400002)
	resp, err := dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update",
		PhoneNumber: phone,
		Fields: map[string]interface{}{
			"name": "New Name",
			"address": map[string]interface{}{
				"address1": "456 New Address", "pincode": 400002,
			},
		},
	})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if resp.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", resp.Status)
	}

	verifyTx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn (verify) failed: %v", err)
	}
	defer txnManager.CommitTxn(verifyTx)

	var reloaded entities.User
	if err := verifyTx.First(&reloaded, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.Name == nil || *reloaded.Name != "New Name" {
		t.Errorf("expected name to be updated to 'New Name', got %v", reloaded.Name)
	}

	var address entities.Address
	if err := verifyTx.Where("user_id = ?", user.ID).First(&address).Error; err != nil {
		t.Fatalf("expected an address row to exist after update: %v", err)
	}
	if address.Address1 != "456 New Address" {
		t.Errorf("expected address1='456 New Address', got %q", address.Address1)
	}

	// Second update with the same phone number should conflict, not overwrite.
	tx2, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn (second user) failed: %v", err)
	}
	otherUser := &entities.User{
		PhoneNumber: "9998880004", IsActive: true, RegisteredOn: time.Now(),
		ProfilePercentage: 0, LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, tx2, otherUser); err != nil {
		txnManager.AbortTxn(tx2)
		t.Fatalf("failed to seed second user: %v", err)
	}
	txnManager.CommitTxn(tx2)
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE id = ?", otherUser.ID)
	})

	_, err = dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update", FirstName: "Update", LastName: "Me", PhoneNumber: phone,
		Fields: map[string]interface{}{"phone_number": "9998880004"},
	})
	if err == nil {
		t.Fatalf("expected a conflict error when updating to an already-used phone_number")
	}
	t.Logf("phone_number conflict correctly rejected: %v", err)

	// A malformed phone_number (not a bare 10-digit string) must be rejected
	// with a clean validation error, not surface a raw DB error from the
	// varchar(10) column truncating/rejecting it.
	_, err = dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update", FirstName: "Update", LastName: "Me", PhoneNumber: phone,
		Fields: map[string]interface{}{"phone_number": "919998880009"},
	})
	if err == nil {
		t.Fatalf("expected a validation error for a malformed (12-digit) phone_number")
	}
	t.Logf("malformed phone_number correctly rejected: %v", err)

	// upsertAddress's update branch: seed a user that already HAS a default
	// user_address row, then update via the DSR API and confirm the SAME row
	// was updated in place (id unchanged, fields overwritten) rather than a
	// second row being inserted alongside it.
	tx3, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn (address-update user) failed: %v", err)
	}
	addressUpdatePhone := "9998880005"
	addressUpdateUser := &entities.User{
		PhoneNumber: addressUpdatePhone, IsActive: true, RegisteredOn: time.Now(),
		ProfilePercentage: 0, LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, tx3, addressUpdateUser); err != nil {
		txnManager.AbortTxn(tx3)
		t.Fatalf("failed to seed address-update user: %v", err)
	}
	existingAddress := &entities.Address{
		UserID: addressUpdateUser.ID, Address1: "Original Address", Pincode: 400003,
		PinCodeID: 3, CityID: 3, StateID: 3, IsDefault: true, IsActive: true,
		CreatedBy: "test", CreatedOn: time.Now(),
	}
	if err := tx3.Create(existingAddress).Error; err != nil {
		txnManager.AbortTxn(tx3)
		t.Fatalf("failed to seed existing default address: %v", err)
	}
	txnManager.CommitTxn(tx3)
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_address WHERE user_id = ?", addressUpdateUser.ID)
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", addressUpdateUser.ID)
		db.Exec("DELETE FROM users WHERE id = ?", addressUpdateUser.ID)
	})

	seedLocation(t, db, 400004)
	resp2, err := dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "update",
		PhoneNumber: addressUpdatePhone,
		Fields: map[string]interface{}{
			"address": map[string]interface{}{
				"address1": "Updated Address", "pincode": 400004,
			},
		},
	})
	if err != nil {
		t.Fatalf("address-update request failed: %v", err)
	}
	if resp2.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", resp2.Status)
	}

	verifyTx2, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn (verify address update) failed: %v", err)
	}
	defer txnManager.CommitTxn(verifyTx2)

	var addressRows []entities.Address
	if err := verifyTx2.Where("user_id = ?", addressUpdateUser.ID).Find(&addressRows).Error; err != nil {
		t.Fatalf("failed to reload addresses for address-update user: %v", err)
	}
	if len(addressRows) != 1 {
		t.Fatalf("expected exactly 1 user_address row after update (in-place update, not insert), got %d", len(addressRows))
	}
	updatedAddress := addressRows[0]
	if updatedAddress.ID != existingAddress.ID {
		t.Errorf("expected the existing address row (id=%d) to be updated in place, got a different row (id=%d)", existingAddress.ID, updatedAddress.ID)
	}
	if updatedAddress.Address1 != "Updated Address" {
		t.Errorf("expected address1='Updated Address', got %q", updatedAddress.Address1)
	}
	if updatedAddress.Pincode != 400004 {
		t.Errorf("expected pincode=400004, got %d", updatedAddress.Pincode)
	}
	if updatedAddress.LastModifiedBy == nil || *updatedAddress.LastModifiedBy != "dsr-api" {
		t.Errorf("expected last_modified_by='dsr-api', got %v", updatedAddress.LastModifiedBy)
	}
	if updatedAddress.LastModifiedOn == nil {
		t.Errorf("expected last_modified_on to be set after an in-place address update")
	}
}

func TestDSRService_HandleRequest_Access(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService(os.Getenv("GCP_BUCKET_NAME"), os.Getenv("GCP_PROJECT_ID"))
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)
	userRepo := repository.NewUserRepository()

	ctx := context.Background()
	tx, err := txnManager.StartTxn()
	if err != nil {
		t.Fatalf("StartTxn failed: %v", err)
	}
	phone := "9998880006"
	user := &entities.User{
		PhoneNumber: phone, IsActive: true, RegisteredOn: time.Now(),
		ProfilePercentage: 0, LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, tx, user); err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := tx.Create(&entities.Address{
		UserID: user.ID, Address1: "789 Export St", Pincode: 400003,
		PinCodeID: 3, CityID: 3, StateID: 3, CreatedBy: "test", CreatedOn: time.Now(),
	}).Error; err != nil {
		txnManager.AbortTxn(tx)
		t.Fatalf("failed to seed address: %v", err)
	}
	txnManager.CommitTxn(tx)
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_address WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", user.ID)
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	resp, err := dsrService.HandleRequest(ctx, dtos.DSRRequestDTO{
		RequestType: "access",
		FirstName:   "Access", LastName: "Me", PhoneNumber: phone,
	})
	if err != nil {
		t.Fatalf("access request failed: %v", err)
	}
	if resp.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", resp.Status)
	}
	if resp.ExportData == nil {
		t.Fatalf("expected non-nil export_data")
	}

	// Round-trip through JSON, the same shape an HTTP caller actually
	// receives, rather than relying on the in-process concrete type.
	raw, err := json.Marshal(resp.ExportData)
	if err != nil {
		t.Fatalf("failed to marshal export_data: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("export_data is not valid JSON: %v", err)
	}
	if doc["user"] == nil {
		t.Fatalf("expected \"user\" key in export document")
	}
	addresses, ok := doc["addresses"].([]interface{})
	if !ok || len(addresses) != 1 {
		t.Fatalf("expected exactly 1 address in export document, got %v", doc["addresses"])
	}
}

func TestDSRHandler_EndToEnd(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService(os.Getenv("GCP_BUCKET_NAME"), os.Getenv("GCP_PROJECT_ID"))
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)
	dsrHandler := handlers.NewDSRHandler(dsrService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares.SuccessResponseMiddleware())
	router.Use(middlewares.ErrorHandler())
	router.POST("/dsr/requests", dsrHandler.HandleRequest)

	body, _ := json.Marshal(dtos.DSRRequestDTO{
		RequestType: "access",
		FirstName:   "HTTP", LastName: "Test", PhoneNumber: "0000000001",
	})
	req := httptest.NewRequest(http.MethodPost, "/dsr/requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			RequestID string `json:"request_id"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse response body %q: %v", rec.Body.String(), err)
	}
	if !parsed.Success {
		t.Fatalf("expected success=true, body=%s", rec.Body.String())
	}
	if parsed.Data.Status != "not_found" {
		t.Fatalf("expected data.status=not_found, got %+v, body=%s", parsed.Data, rec.Body.String())
	}

	// Malformed request_type should be a 4xx, not a 500.
	badBody, _ := json.Marshal(map[string]string{"request_type": "delete_everything", "first_name": "x", "last_name": "y", "phone_number": "0000000002"})
	badReq := httptest.NewRequest(http.MethodPost, "/dsr/requests", bytes.NewReader(badBody))
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code < 400 || badRec.Code >= 500 {
		t.Fatalf("expected a 4xx for an invalid request_type, got %d: %s", badRec.Code, badRec.Body.String())
	}
	var badParsed struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(badRec.Body.Bytes(), &badParsed); err != nil {
		t.Fatalf("failed to parse error response body %q: %v", badRec.Body.String(), err)
	}
	if badParsed.Success {
		t.Fatalf("expected success=false for the invalid-request_type case, body=%s", badRec.Body.String())
	}
}

// TestDSRHandler_Delete_Completed_EndToEnd drives a REAL, matched delete
// request through the HTTP handler with the production
// SuccessResponseMiddleware/ErrorHandler pair mounted.
//
// TestDSRHandler_EndToEnd only covers not_found and a validation reject, so
// the completed-response shape had no handler-level coverage - and that is
// exactly the seam where a double-wrapped body ({"data":{"data":{...}}}) was
// caught in review. This test asserts the un-wrapped shape explicitly: the
// DSR payload's own keys must sit directly under "data", with no nested
// "data" of their own. Delete (rather than update) is used because it needs
// no fields payload and its response carries anonymized_tables, which is
// easy to assert on.
func TestDSRHandler_Delete_Completed_EndToEnd(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	dsrRequestRepo := repository.NewDSRRequestRepository()
	pinCodeRepo := repository.NewPinCodeRepository()
	cityRepo := repository.NewCityRepository()
	gcsService, err := utils.NewGCSService("", "") // unused by the delete path
	if err != nil {
		t.Fatalf("NewGCSService failed: %v", err)
	}
	dsrService := services.NewDSRService(txnManager, dsrRequestRepo, gcsService, pinCodeRepo, cityRepo)
	dsrHandler := handlers.NewDSRHandler(dsrService)
	userRepo := repository.NewUserRepository()

	ctx := context.Background()
	phone := "9998880007"
	name := "Handler Delete"
	email := "handler-delete@example.com"
	user := &entities.User{
		Name: &name, Email: &email, PhoneNumber: phone, IsActive: true,
		RegisteredOn: time.Now(), ProfilePercentage: 0, LastLanguageSelection: 1,
	}
	if err := userRepo.Create(ctx, db, user); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_refresh_tokens WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM user_address WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM otp_logs WHERE mobile_no = ?", phone)
		db.Exec("DELETE FROM dsr_requests WHERE matched_user_id = ?", user.ID)
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares.SuccessResponseMiddleware())
	router.Use(middlewares.ErrorHandler())
	router.POST("/dsr/requests", dsrHandler.HandleRequest)

	body, _ := json.Marshal(dtos.DSRRequestDTO{
		RequestType: "delete",
		FirstName:   "Handler", LastName: "Delete", PhoneNumber: phone,
	})
	req := httptest.NewRequest(http.MethodPost, "/dsr/requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse response body %q: %v", rec.Body.String(), err)
	}
	if success, ok := raw["success"].(bool); !ok || !success {
		t.Fatalf("expected success=true, body=%s", rec.Body.String())
	}
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a \"data\" object in the response, body=%s", rec.Body.String())
	}
	if _, nested := data["data"]; nested {
		t.Fatalf("response is double-wrapped (data.data present), body=%s", rec.Body.String())
	}
	if data["status"] != "completed" {
		t.Fatalf("expected data.status=completed, got %v, body=%s", data["status"], rec.Body.String())
	}
	if requestID, ok := data["request_id"].(string); !ok || requestID == "" {
		t.Fatalf("expected a non-empty data.request_id, body=%s", rec.Body.String())
	}
	anonymized, ok := data["anonymized_tables"].([]interface{})
	if !ok || len(anonymized) == 0 {
		t.Fatalf("expected a non-empty data.anonymized_tables, body=%s", rec.Body.String())
	}

	// The response claimed "completed", so the anonymization must actually be
	// committed and visible to a fresh read.
	var reloaded entities.User
	if err := db.First(&reloaded, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.Name != nil || reloaded.Email != nil {
		t.Errorf("expected name/email to be redacted, got name=%v email=%v", reloaded.Name, reloaded.Email)
	}
	if reloaded.IsActive {
		t.Errorf("expected is_active=false after a completed delete")
	}
	if reloaded.PhoneNumber == phone {
		t.Errorf("expected phone_number to be replaced with a redacted placeholder")
	}
}
