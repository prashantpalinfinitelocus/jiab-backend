package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type ReferralService interface {
	GenerateReferral(ctx context.Context, userID string, mobileNumber string, req dtos.GenerateReferralRequest) (*dtos.ReferralResponse, error)
	GetUserReferrals(ctx context.Context, userID string, filter dtos.ReferralFilter) (*dtos.ReferralListResponse, error)
	SendReferralAgain(ctx context.Context, userID string, req dtos.SendReferralAgainRequest) (*string, error)
	VerifyReferral(ctx context.Context, user *entities.User, req dtos.VerifyReferralRequest) (string, error)
}

type referralService struct {
	txnManager          *utils.TransactionManager
	referralRepo        repository.UserReferralRepository
	userRepo            repository.UserRepository
	infobipService      InfobipService
	comicCoinService    ComicCoinService
	notificationService NotificationService
}

func NewReferralService(txnManager *utils.TransactionManager, referralRepo repository.UserReferralRepository, userRepo repository.UserRepository, infobipService InfobipService, comicCoinService ComicCoinService, notificationService NotificationService) ReferralService {
	return &referralService{
		txnManager:          txnManager,
		referralRepo:        referralRepo,
		userRepo:            userRepo,
		infobipService:      infobipService,
		comicCoinService:    comicCoinService,
		notificationService: notificationService,
	}
}

func generateReferralCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToUpper(base64.URLEncoding.EncodeToString(bytes)[:8]), nil
}

func (s *referralService) GenerateReferral(ctx context.Context, userID string, mobileNumber string, req dtos.GenerateReferralRequest) (*dtos.ReferralResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	if mobileNumber == req.ReferTo {
		s.txnManager.AbortTxn(tx)
		return nil, errs.ErrCannotReferYourself
	}

	// Check if there's already an active referral for this user pair
	hasActiveReferral, err := s.referralRepo.CheckActiveReferral(ctx, tx, userID, req.ReferTo)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if hasActiveReferral {
		s.txnManager.AbortTxn(tx)
		return nil, errs.ErrReferralAlreadyExists
	}

	userExists, err := s.userRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{"phone_number": req.ReferTo, "is_active": true})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if userExists != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("Cannot refer an existing user", nil)
	}

	code, err := generateReferralCode()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate referral code", err)
	}

	referral := &entities.UserReferral{
		UserID:       userID,
		ReferTo:      req.ReferTo,
		ReferOn:      time.Now(),
		IsAccept:     false,
		IsResend:     false,
		ReSendCount:  0,
		ReferralCode: &code,
		IsActive:     true,
		IsDeleted:    false,
		CreatedBy:    userID,
		CreatedOn:    time.Now(),
	}

	if err := s.referralRepo.Create(ctx, tx, referral); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if err := s.infobipService.SendReferral(referral.ReferTo, *referral.ReferralCode); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to send referral message", err)
	}

	s.txnManager.CommitTxn(tx)

	return &dtos.ReferralResponse{
		ReferralCode: code,
	}, nil
}

func (s *referralService) GetUserReferrals(ctx context.Context, userID string, filter dtos.ReferralFilter) (*dtos.ReferralListResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.AbortTxn(tx)

	referrals, total, err := s.referralRepo.GetUserReferrals(ctx, tx, userID, filter)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	response := &dtos.ReferralListResponse{
		Referrals: make([]dtos.ReferralStatusResponse, 0),
		Total:     total,
	}

	if filter.Page != nil && filter.PageSize != nil {
		response.Page = filter.Page
		response.PageSize = filter.PageSize
	}

	for _, ref := range referrals {
		var status string
		if ref.IsAccept {
			status = "Accepted"
		} else {
			status = "Pending"
		}

		referralResponse := dtos.ReferralStatusResponse{
			ID:           ref.ID,
			MobileNumber: ref.ReferTo,
			Status:       status,
		}
		response.Referrals = append(response.Referrals, referralResponse)
	}

	s.txnManager.CommitTxn(tx)
	return response, nil
}

func (s *referralService) SendReferralAgain(ctx context.Context, userID string, req dtos.SendReferralAgainRequest) (*string, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	referral, err := s.referralRepo.FindById(ctx, tx, req.ReferralID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if referral.ReSendOn != nil && referral.ReSendOn.Add(time.Hour*24*7).After(time.Now()) {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("Cannot resend referral within 7 days of last resend", nil)
	}

	if referral == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError("Referral not found", nil)
	}

	if referral.UserID != userID {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewForbiddenError("Not authorized to resend this referral", nil)
	}

	if err := s.referralRepo.UpdateResendStatus(ctx, tx, req.ReferralID); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if err := s.infobipService.SendReferral(referral.ReferTo, *referral.ReferralCode); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to send referral message", err)
	}

	s.txnManager.CommitTxn(tx)
	return referral.ReferralCode, nil
}

func (s *referralService) VerifyReferral(ctx context.Context, user *entities.User, req dtos.VerifyReferralRequest) (string, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return "", err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	referral, err := s.referralRepo.GetReferralByCode(ctx, tx, req.ReferralCode)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	if referral == nil {
		s.txnManager.AbortTxn(tx)
		return "", errs.ErrWrongReferralCode
	}

	if referral.IsAccept && !referral.IsActive {
		s.txnManager.AbortTxn(tx)
		return "", errs.NewBadRequestError("Referral code is already used", nil)
	}

	if referral.UserID == user.ID {
		s.txnManager.AbortTxn(tx)
		return "", errs.NewBadRequestError("You cannot verify your own referral code", nil)
	}

	if referral.ReferTo != user.PhoneNumber {
		s.txnManager.AbortTxn(tx)
		return "", errs.ErrWrongReferralCode
	}

	if !referral.IsActive {
		s.txnManager.AbortTxn(tx)
		return "", errs.NewBadRequestError("Referral code is no longer active", nil)
	}

	// Fetch the mobile number of the user who created the referral
	referrerUser, err := s.userRepo.FindByField(ctx, tx, "id", referral.CreatedBy)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	if err := s.referralRepo.UpdateReferralStatus(ctx, tx, referral.ID, true); err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	_, err = s.comicCoinService.UpdateBalance(ctx, tx, referral.UserID, constants.REWARD_REFERRER_ON_VERIFY)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	rewardReferrerOnVerify := constants.REWARD_REFERRER_ON_VERIFY
	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, referral.UserID, &rewardReferrerOnVerify)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	// The referrer's friend just verified their code — tell the referrer.
	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_REFERRAL_SUCCESSFUL, referral.UserID, nil)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	_, err = s.comicCoinService.UpdateBalance(ctx, tx, user.ID, constants.REWARD_REFEREE_ON_VERIFY)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	rewardRefereeOnVerify := constants.REWARD_REFEREE_ON_VERIFY
	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, user.ID, &rewardRefereeOnVerify)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	if err := s.userRepo.Update(ctx, tx, user.ID, &entities.User{
		ReferralCode: &req.ReferralCode,
		IsReferral:   true,
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return "", err
	}

	s.txnManager.CommitTxn(tx)
	return referrerUser.PhoneNumber, nil
}
