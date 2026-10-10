package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type MixCodeService struct {
	txnManager          *utils.TransactionManager
	mixCodeRepo         *repository.UserMixCodeRepository
	userWalletRepo      repository.UserWalletRepository
	comicCoinService    ComicCoinService
	notificationService NotificationService
	mixcodeClient       *vendors.MixcodeClient
}

func NewMixCodeService(
	txnManager *utils.TransactionManager,
	mixCodeRepo *repository.UserMixCodeRepository,
	userWalletRepo repository.UserWalletRepository,
	comicCoinService ComicCoinService,
	notificationService NotificationService,
	mixcodeClient *vendors.MixcodeClient,
) *MixCodeService {
	return &MixCodeService{
		txnManager:          txnManager,
		mixCodeRepo:         mixCodeRepo,
		userWalletRepo:      userWalletRepo,
		comicCoinService:    comicCoinService,
		notificationService: notificationService,
		mixcodeClient:       mixcodeClient,
	}
}

func (s *MixCodeService) RedeemMixCode(ctx context.Context, userID string, mixCode string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	invalidCount, validCount, err := s.mixCodeRepo.GetDailyMixCodeCount(userID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	if invalidCount >= int64(constants.MIXCODE_INVALID_LIMIT) || validCount >= int64(constants.MIXCODE_REDEEM_LIMIT) {
		s.txnManager.AbortTxn(tx)
		return errs.ErrDailyLimitExceeded
	}

	// To validate the mix code
	endpoint := fmt.Sprintf("%s%s.json", constants.MIXCODE_VALIDATE_CODE_ENDPOINT, mixCode)
	validateResp, err := s.mixcodeClient.ApiClient.Get(endpoint, map[string]string{"programId": s.mixcodeClient.Cfg.ProgramID})
	if err != nil {
		return err
	}

	if validateResp.StatusCode != 200 {
		mixCode := &entities.UserMixCodeDetails{
			UserID:         userID,
			MixCode:        mixCode,
			IsValidMixCode: false,
			IsActive:       true,
			CreatedBy:      userID,
			CreatedOn:      time.Now(),
		}
		if err := s.mixCodeRepo.Create(mixCode); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		return errs.ErrInvalidUniqueCode
	}

	var validateMixCode map[string]interface{}
	if err := validateResp.ParseJSON(&validateMixCode); err != nil {
		return errs.NewInternalServerError("Failed to parse mix code validation response", err)
	}

	newMixCode := &entities.UserMixCodeDetails{
		UserID:         userID,
		MixCode:        mixCode,
		IsValidMixCode: true,
		IsActive:       true,
		CreatedBy:      userID,
		CreatedOn:      time.Now(),
	}

	if validateMixCode["redeemed"] == true {
		newMixCode.IsAlreadyRedeemedMixCode = true
		if err := s.mixCodeRepo.Create(newMixCode); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		s.txnManager.CommitTxn(tx)
		return errs.ErrUniqueCodeAlreadyRedeemed
	}

	if validateMixCode["lot"].(map[string]interface{})["expired"] == true {
		newMixCode.IsExpiredMixCode = true
		if err := s.mixCodeRepo.Create(newMixCode); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		s.txnManager.CommitTxn(tx)
		return errs.ErrUniqueCodeExpired
	}

	if validateMixCode["lot"].(map[string]interface{})["active"] == false {
		newMixCode.IsActiveMixCode = false
		if err := s.mixCodeRepo.Create(newMixCode); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		s.txnManager.CommitTxn(tx)
		return errs.ErrUniqueCodeNotActive
	}

	transactionID, ok := ctx.Value("requestID").(string)
	if !ok {
		uuid, _ := uuid.NewUUID()
		transactionID = uuid.String()
	}
	redeemResp, err := s.mixcodeClient.ApiClient.Put(
		endpoint,
		map[string]string{"programId": s.mixcodeClient.Cfg.ProgramID},
		map[string]string{
			"programId":     s.mixcodeClient.Cfg.ProgramID,
			"transactionId": transactionID,
			"consumerId":    userID,
		},
	)
	if err != nil {
		return err
	}

	if redeemResp.StatusCode != 200 {
		if err := s.mixCodeRepo.Create(newMixCode); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		s.txnManager.CommitTxn(tx)
		return errs.ErrFailedToRedeemUniqueCode
	}

	newMixCode.IsRedeemedSuccessfully = true
	newMixCode.LastModifiedBy = &userID
	newMixCode.IsActiveMixCode = true
	newMixCode.IsAlreadyRedeemedMixCode = false
	newMixCode.IsExpiredMixCode = false
	now := time.Now()
	newMixCode.LastModifiedOn = &now

	if err := s.mixCodeRepo.Create(newMixCode); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	// Add 20 coins to user wallet
	if _, err := s.comicCoinService.UpdateBalance(ctx, tx, userID, 20); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	// Send notification for winning coins
	coinAmount := constants.MIXCODE_REDEEM_COIN_COUNT
	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &coinAmount)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}
