package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type CustomerDataService interface {
	GetUserKOCID(ctx context.Context, userID string) (*dtos.UserKOCIDResponse, error)
}

type customerDataService struct {
	userRepo   repository.UserRepository
	kocidRepo  repository.UserKOCIDRepository
	cdsClient  *vendors.CustomerDataServiceClient
	txnManager *utils.TransactionManager
}

func NewCustomerDataService(
	userRepo repository.UserRepository,
	kocidRepo repository.UserKOCIDRepository,
	cdsClient *vendors.CustomerDataServiceClient,
	txnManager *utils.TransactionManager,
) CustomerDataService {
	return &customerDataService{
		userRepo:   userRepo,
		kocidRepo:  kocidRepo,
		cdsClient:  cdsClient,
		txnManager: txnManager,
	}
}

func (s *customerDataService) GetUserKOCID(ctx context.Context, userID string) (*dtos.UserKOCIDResponse, error) {
	// Single DB call - Check if KOCID already exists
	kocidRecord, err := s.kocidRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NewInternalServerError("Failed to check existing KOCID", err)
	}

	if kocidRecord != nil {
		// Return cached KOCID - no additional DB calls
		syncedAt := kocidRecord.SyncedAt.Format(time.RFC3339)
		return &dtos.UserKOCIDResponse{
			KOCID:    &kocidRecord.HashedKOCID,
			SyncedAt: &syncedAt,
			Message:  "KOCID retrieved from cache",
		}, nil
	}

	// KOCID doesn't exist, sync from Customer Data Service
	kocid, err := s.syncKOCIDFromCDS(ctx, userID)
	if err != nil {
		// Return graceful error - don't break user flow
		logrus.WithError(err).Warn("Failed to sync KOCID from CDS")
		return &dtos.UserKOCIDResponse{
			KOCID:   nil,
			Message: "KOCID sync temporarily unavailable",
		}, nil
	}

	syncedAt := time.Now().Format(time.RFC3339)
	return &dtos.UserKOCIDResponse{
		KOCID:    &kocid,
		SyncedAt: &syncedAt,
		Message:  "KOCID synced successfully",
	}, nil
}

func (s *customerDataService) syncKOCIDFromCDS(ctx context.Context, userID string) (string, error) {
	// Single DB call - Get user details
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return "", err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		return "", err
	}

	s.txnManager.CommitTxn(tx)

	// Step 1: Send user data to datamart (optional but recommended)
	if err := s.sendUserToDatamart(ctx, user); err != nil {
		logrus.WithError(err).Warn("Failed to send user to datamart, continuing with KOCID sync")
	}

	// Step 2: Get hashed KOCID
	kocidReq := s.buildKOCIDRequest(user)
	kocid, err := s.cdsClient.GetHashedKOCID(ctx, kocidReq)
	if err != nil {
		return "", err
	}

	// Step 3: Single DB write - Store KOCID
	tx, err = s.txnManager.StartTxn()
	if err != nil {
		return "", err
	}
	defer s.txnManager.AbortTxn(tx)

	kocidRecord := &entities.UserKOCID{
		UserID:      userID,
		HashedKOCID: kocid,
		SyncedAt:    time.Now(),
	}

	if err := s.kocidRepo.Create(ctx, tx, kocidRecord); err != nil {
		return "", errors.NewInternalServerError("Failed to store KOCID", err)
	}

	s.txnManager.CommitTxn(tx)

	return kocid, nil
}

// Simplified datamart request - no address, minimal data
func (s *customerDataService) buildDatamartRequest(user *entities.User) *dtos.DatamartCollectRequest {
	req := &dtos.DatamartCollectRequest{
		ClientID: s.cdsClient.Config.ClientID,
	}

	// Map only user table data - no additional DB calls
	if user.Email != nil && *user.Email != "" {
		req.Email = *user.Email
	}

	if user.PhoneNumber != "" {
		req.PhoneWithCountryCode = s.formatPhoneWithCountryCode(user.PhoneNumber)
	}

	if user.Name != nil && *user.Name != "" {
		names := strings.Fields(*user.Name)
		if len(names) > 0 {
			req.FirstName = names[0]
		}
		if len(names) > 1 {
			req.LastName = strings.Join(names[1:], " ")
		}
	}

	if user.DOB != nil {
		req.DOB = user.DOB.Format("02/01/2006") // DD/MM/YYYY format
	}

	if user.Gender != nil {
		req.Gender = s.mapGender(*user.Gender)
	}

	// Add user identifiers
	req.UserIdentifiers = []dtos.UserIdentifier{
		{
			UserIdentifier:        user.ID,
			UserIdentifierType:    "IDENTIFIER_ID",
			UserIdentifierSubType: "USER_ID",
		},
	}

	return req
}

// Simplified KOCID request - no address, minimal data
func (s *customerDataService) buildKOCIDRequest(user *entities.User) *dtos.GetHashedKOCIDRequest {
	req := &dtos.GetHashedKOCIDRequest{
		ClientID: s.cdsClient.Config.ClientID,
	}

	// Add phone with country code
	if user.PhoneNumber != "" {
		req.PhoneWithCountryCode = s.formatPhoneWithCountryCode(user.PhoneNumber)
	}

	return req
}

func (s *customerDataService) formatPhoneWithCountryCode(phone string) string {
	// Assuming Indian numbers, add +91 prefix
	if !strings.HasPrefix(phone, "+") {
		return "+91-" + phone
	}
	return phone
}

func (s *customerDataService) mapGender(gender int) string {
	switch gender {
	case 1:
		return "Male"
	case 2:
		return "Female"
	default:
		return "Other"
	}
}

func (s *customerDataService) sendUserToDatamart(ctx context.Context, user *entities.User) error {
	req := s.buildDatamartRequest(user)
	return s.cdsClient.CollectUserData(ctx, req)
}
