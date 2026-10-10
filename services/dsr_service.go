package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type DSRService interface {
	HandleRequest(ctx context.Context, req dtos.DSRRequestDTO) (*dtos.DSRResponseDTO, error)
}

type dsrService struct {
	txnManager     *utils.TransactionManager
	dsrRequestRepo repository.DSRRequestRepository
	gcsService     utils.GCSService
	pinCodeRepo    repository.PinCodeRepository
	cityRepo       repository.CityRepository
}

func NewDSRService(txnManager *utils.TransactionManager, dsrRequestRepo repository.DSRRequestRepository, gcsService utils.GCSService, pinCodeRepo repository.PinCodeRepository, cityRepo repository.CityRepository) DSRService {
	return &dsrService{txnManager: txnManager, dsrRequestRepo: dsrRequestRepo, gcsService: gcsService, pinCodeRepo: pinCodeRepo, cityRepo: cityRepo}
}

func (s *dsrService) HandleRequest(ctx context.Context, req dtos.DSRRequestDTO) (*dtos.DSRResponseDTO, error) {
	if req.RequestType == "update" {
		if err := validateUpdateFields(req.Fields); err != nil {
			return nil, err
		}
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.matchUser(ctx, tx, req.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to match user: %w", err)
	}

	if user == nil {
		audit, auditErr := s.recordAudit(ctx, tx, req, nil, "not_found", nil, nil)
		if auditErr != nil {
			s.txnManager.AbortTxn(tx)
			return nil, auditErr
		}
		// tx.Commit() directly instead of txnManager.CommitTxn, which
		// discards the commit error: returning status "not_found" with a
		// request_id asserts that a durable audit row exists, and a failed
		// commit (constraint violation, connection drop, serialization
		// failure) means it does not. For a compliance audit trail,
		// reporting success over an unpersisted row is the worst failure
		// mode, so the error is surfaced instead.
		if commitErr := tx.Commit().Error; commitErr != nil {
			return nil, errs.NewInternalServerError("failed to commit dsr not_found audit", commitErr)
		}
		return &dtos.DSRResponseDTO{RequestID: audit.ID, Status: "not_found"}, nil
	}

	if req.Email != "" && user.Email != nil && *user.Email != "" && !strings.EqualFold(*user.Email, req.Email) {
		logrus.Warnf("dsr request: matched user_id=%s by phone, but provided email %q does not match stored email %q", user.ID, req.Email, *user.Email)
	}

	// "access" is dispatched on its own path rather than through the switch
	// below because its DB work is entirely read-only and it has to make
	// network calls (GCS upload + URL signing). Running it inline would hold
	// tx open across those calls and, worse, would leave the uploaded PII
	// object orphaned in GCS - on a working signed URL, with no committed
	// record of its path - if a later step on tx failed and rolled back.
	// handleAccessRequest (dsr_export.go) sequences read -> release tx ->
	// upload -> audit instead.
	if req.RequestType == "access" {
		return s.handleAccessRequest(ctx, tx, req, user)
	}

	var (
		resp    *dtos.DSRResponseDTO
		details interface{}
	)

	switch req.RequestType {
	case "delete":
		resp, details, err = s.handleDelete(ctx, tx, user)
	case "update":
		resp, details, err = s.handleUpdate(ctx, tx, user, req.Fields)
	default:
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError(fmt.Sprintf("unsupported request_type %q", req.RequestType), nil)
	}

	if err != nil {
		// tx must be rolled back to discard any partial handler side effects
		// (e.g. a handler that anonymized some tables before failing on
		// another). The failure audit row must survive that rollback, so it
		// is written on an independent transaction opened after tx is
		// aborted, not on tx itself.
		s.txnManager.AbortTxn(tx)
		errMsg := err.Error()
		if auditErr := s.recordFailureAudit(ctx, req, &user.ID, errMsg, nil); auditErr != nil {
			logrus.Errorf("failed to record dsr failure audit: %v", auditErr)
		}
		return nil, err
	}

	audit, err := s.recordAudit(ctx, tx, req, &user.ID, "completed", details, nil)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// As on the not_found path: tx.Commit() directly, so a failed commit is
	// never reported to the caller as status "completed". This commit carries
	// BOTH the handler's writes (anonymization / field updates) and the
	// "completed" audit row, so if it fails nothing at all was persisted.
	if commitErr := tx.Commit().Error; commitErr != nil {
		wrapped := errs.NewInternalServerError("failed to commit dsr transaction", commitErr)
		// Best-effort failure audit on a fresh independent transaction. If
		// the commit failed because the connection itself is gone this will
		// fail too, which is why it only logs - the caller still gets the
		// error rather than a false "completed".
		if auditErr := s.recordFailureAudit(ctx, req, &user.ID, wrapped.Error(), nil); auditErr != nil {
			logrus.Errorf("failed to record dsr failure audit after commit failure: %v", auditErr)
		}
		return nil, wrapped
	}

	resp.RequestID = audit.ID
	resp.Status = "completed"
	return resp, nil
}

func (s *dsrService) matchUser(ctx context.Context, tx *gorm.DB, phoneNumber string) (*entities.User, error) {
	bare10, withCountryCode := normalizePhoneVariants(phoneNumber)
	var user entities.User
	err := tx.WithContext(ctx).Where("phone_number = ? OR phone_number = ?", bare10, withCountryCode).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *dsrService) recordAudit(ctx context.Context, tx *gorm.DB, req dtos.DSRRequestDTO, matchedUserID *string, status string, details interface{}, errMsg *string) (*entities.DSRRequest, error) {
	var detailsJSON *string
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal dsr audit details: %w", err)
		}
		s := string(b)
		detailsJSON = &s
	}

	var email *string
	if req.Email != "" {
		email = &req.Email
	}

	now := time.Now()
	record := &entities.DSRRequest{
		RequestType:          req.RequestType,
		RequesterFirstName:   req.FirstName,
		RequesterLastName:    req.LastName,
		RequesterEmail:       email,
		RequesterPhoneNumber: req.PhoneNumber,
		MatchedUserID:        matchedUserID,
		Status:               status,
		Details:              detailsJSON,
		ErrorMessage:         errMsg,
		RequestedOn:          now,
		CompletedOn:          &now,
	}
	if err := s.dsrRequestRepo.Create(ctx, tx, record); err != nil {
		return nil, err
	}
	return record, nil
}

// recordFailureAudit writes a "failed" dsr_requests row on its own
// independent transaction, so it durably persists even though the caller
// has already rolled back (or is about to roll back) the business
// transaction that produced the failure.
//
// details is usually nil, but the access/export flow passes the GCS
// object_path when an export object was already uploaded before the failure
// occurred, so the orphaned PII object is traceable and can be cleaned up.
func (s *dsrService) recordFailureAudit(ctx context.Context, req dtos.DSRRequestDTO, matchedUserID *string, errMsg string, details interface{}) error {
	auditTx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(auditTx)

	if _, err := s.recordAudit(ctx, auditTx, req, matchedUserID, "failed", details, &errMsg); err != nil {
		s.txnManager.AbortTxn(auditTx)
		return err
	}

	s.txnManager.CommitTxn(auditTx)
	return nil
}

func (s *dsrService) handleDelete(ctx context.Context, tx *gorm.DB, user *entities.User) (*dtos.DSRResponseDTO, interface{}, error) {
	originalPhone := user.PhoneNumber
	bare10, withCountryCode := normalizePhoneVariants(originalPhone)
	// users.phone_number is a fixed varchar(10) column in the live schema, so
	// the placeholder must fit within 10 characters. "deleted-"+user.ID (a
	// uuid) would be far too long and fail with a "value too long" DB error,
	// so derive a compact, still-unambiguous marker from the user's id
	// instead: "del" (3 chars) + first 7 hex chars of the id with hyphens
	// stripped (7 chars) = 10 chars exactly.
	redactedPhone := "del" + strings.ReplaceAll(user.ID, "-", "")[:7]

	if err := tx.WithContext(ctx).Model(&entities.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"name": nil, "email": nil, "dob": nil, "gender": nil,
		"alternate_mobile": nil, "profile_picture": nil, "ip_address": nil,
		"phone_number": redactedPhone, "is_active": false, "refresh_token": nil,
	}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to redact users row: %w", err)
	}
	touched := []string{"users"}

	if err := tx.WithContext(ctx).Model(&entities.UserRefreshToken{}).Where("user_id = ?", user.ID).Update("is_deleted", true).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to invalidate user_refresh_tokens: %w", err)
	}
	touched = append(touched, "user_refresh_tokens")

	if err := tx.WithContext(ctx).Model(&entities.Address{}).Where("user_id = ?", user.ID).Updates(map[string]interface{}{
		"address1": "[REDACTED]", "address2": nil, "nearest_landmark": nil, "shipping_mobile": nil,
	}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to redact user_address: %w", err)
	}
	touched = append(touched, "user_address")

	// NOTE: entities.UserWinner.Mobile is NOT a real column on the
	// user_winners table - it's populated only by a read-time SQL alias
	// (`users.phone_number as mobile`, see repository/user_winner_repository.go)
	// joined in from the users table. There is no persisted mobile value on
	// user_winners to redact; an UPDATE against a "mobile" column fails with
	// "column \"mobile\" of relation \"user_winners\" does not exist" on the
	// live schema. Since that value is always re-derived from users.phone_number
	// at read time, redacting the users row above already fully redacts it -
	// no separate write against user_winners is needed or possible.

	if err := tx.WithContext(ctx).Where("mobile_no = ? OR mobile_no = ?", bare10, withCountryCode).Delete(&entities.OTPLogs{}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete otp_logs: %w", err)
	}
	touched = append(touched, "otp_logs")

	if err := tx.WithContext(ctx).Where("phone_number = ? OR phone_number = ?", bare10, withCountryCode).Delete(&entities.SSOToken{}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete sso_tokens: %w", err)
	}
	touched = append(touched, "sso_tokens")

	resp := &dtos.DSRResponseDTO{AnonymizedTables: touched}
	details := map[string]interface{}{"anonymized_tables": touched}
	return resp, details, nil
}

// isBare10DigitPhone reports whether value is exactly 10 ASCII digits, the
// only shape users.phone_number (a hard varchar(10) column) can hold. A
// caller-supplied value in another shape (91-prefixed 12-digit, "+91...",
// non-numeric) must be rejected before it ever reaches a query or an
// Updates() call: the conflict-check SELECT would trivially find no match
// against it (every existing row is exactly 10 chars), letting it through to
// the final UPDATE, which would then fail with a raw, unwrapped Postgres
// "value too long for type character varying(10)" error instead of a clean
// validation error.
func isBare10DigitPhone(value string) bool {
	if len(value) != 10 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *dsrService) handleUpdate(ctx context.Context, tx *gorm.DB, user *entities.User, fields map[string]interface{}) (*dtos.DSRResponseDTO, interface{}, error) {
	userUpdates := map[string]interface{}{}
	updatedFields := []string{}

	for key, rawValue := range fields {
		if key == "address" {
			if err := s.upsertAddress(ctx, tx, user.ID, rawValue); err != nil {
				return nil, nil, err
			}
			updatedFields = append(updatedFields, "address")
			continue
		}

		value, ok := rawValue.(string)
		if !ok {
			return nil, nil, errs.NewBadRequestError(fmt.Sprintf("field %q must be a string", key), nil)
		}

		switch key {
		case "name":
			userUpdates["name"] = value
		case "email":
			userUpdates["email"] = value
		case "phone_number":
			if !isBare10DigitPhone(value) {
				return nil, nil, errs.NewBadRequestError("phone_number must be a 10-digit number", nil)
			}
			// The conflict check must consider BOTH shapes the number can be
			// stored as (bare 10-digit and 91-prefixed 12-digit), exactly like
			// matchUser and handleDelete do. Checking only the bare form would
			// miss another account holding the same logical number in the
			// 91-prefixed form, leaving two accounts sharing a phone number and
			// making matchUser's "first match wins" resolution non-deterministic
			// for every later DSR request against that number.
			bare10, withCountryCode := normalizePhoneVariants(value)
			var existing entities.User
			err := tx.WithContext(ctx).Where("(phone_number = ? OR phone_number = ?) AND id <> ?", bare10, withCountryCode, user.ID).First(&existing).Error
			if err == nil {
				return nil, nil, errs.NewConflictError("phone_number is already in use by another account", nil)
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, fmt.Errorf("failed to check phone_number conflict: %w", err)
			}
			userUpdates["phone_number"] = value
		case "dob":
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				return nil, nil, errs.NewBadRequestError("dob must be in YYYY-MM-DD format", err)
			}
			userUpdates["dob"] = parsed
		case "gender":
			g, err := strconv.Atoi(value)
			if err != nil {
				return nil, nil, errs.NewBadRequestError("gender must be an integer", err)
			}
			userUpdates["gender"] = g
		case "alternate_mobile":
			userUpdates["alternate_mobile"] = value
		case "profile_picture":
			userUpdates["profile_picture"] = value
		default:
			// validateUpdateFields (dsr_helpers.go) already rejected anything
			// outside the whitelist, so reaching here means the whitelist has
			// gained a key without a matching case above. Failing loudly is
			// essential: updatedFields is appended to unconditionally below, so
			// silently falling through would report the field as "updated" in
			// both the API response and the compliance audit row while nothing
			// was actually written.
			return nil, nil, errs.NewInternalServerError(fmt.Sprintf("whitelisted field %q has no update handler", key), nil)
		}
		updatedFields = append(updatedFields, key)
	}

	if len(userUpdates) > 0 {
		if err := tx.WithContext(ctx).Model(&entities.User{}).Where("id = ?", user.ID).Updates(userUpdates).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to update user fields: %w", err)
		}
	}

	resp := &dtos.DSRResponseDTO{UpdatedFields: updatedFields}
	details := map[string]interface{}{"updated_fields": updatedFields}
	return resp, details, nil
}

func (s *dsrService) upsertAddress(ctx context.Context, tx *gorm.DB, userID string, rawValue interface{}) error {
	b, err := json.Marshal(rawValue)
	if err != nil {
		return errs.NewBadRequestError("field \"address\" is malformed", err)
	}
	var addr dtos.DSRAddressFieldDTO
	if err := json.Unmarshal(b, &addr); err != nil {
		return errs.NewBadRequestError("field \"address\" is malformed", err)
	}
	if addr.Address1 == "" || addr.Pincode == 0 {
		return errs.NewBadRequestError("field \"address\" requires address1 and pincode", nil)
	}

	pinCode, err := s.pinCodeRepo.FindByPincode(ctx, tx, addr.Pincode)
	if err != nil {
		return fmt.Errorf("failed to look up pincode: %w", err)
	}
	if pinCode == nil {
		return errs.NewBadRequestError(fmt.Sprintf("pincode %d not found", addr.Pincode), nil)
	}
	city, err := s.cityRepo.FindByID(ctx, tx, pinCode.CityID)
	if err != nil || city == nil {
		return fmt.Errorf("failed to look up city for pincode %d: %w", addr.Pincode, err)
	}

	var existing entities.Address
	err = tx.WithContext(ctx).Where("user_id = ? AND is_default = ?", userID, true).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		newAddr := entities.Address{
			UserID: userID, Address1: addr.Address1, Address2: addr.Address2,
			Pincode: addr.Pincode, PinCodeID: pinCode.ID, CityID: pinCode.CityID, StateID: city.StateID,
			NearestLandmark: addr.NearestLandmark, ShippingMobile: addr.ShippingMobile,
			IsDefault: true, IsActive: true, CreatedBy: "dsr-api", CreatedOn: time.Now(),
		}
		if err := tx.WithContext(ctx).Create(&newAddr).Error; err != nil {
			return fmt.Errorf("failed to insert user_address: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("failed to look up existing user_address: %w", err)
	default:
		now := time.Now()
		modifiedBy := "dsr-api"
		if err := tx.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"address1": addr.Address1, "address2": addr.Address2,
			"pincode": addr.Pincode, "pin_code_id": pinCode.ID, "city_id": pinCode.CityID, "state_id": city.StateID,
			"nearest_landmark": addr.NearestLandmark, "shipping_mobile": addr.ShippingMobile,
			"last_modified_by": modifiedBy, "last_modified_on": now,
		}).Error; err != nil {
			return fmt.Errorf("failed to update user_address: %w", err)
		}
		return nil
	}
}
