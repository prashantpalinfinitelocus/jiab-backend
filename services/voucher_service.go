package services

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type VoucherService interface {
	GetVoucherInfo(ctx context.Context, userId string, languageID int) ([]dtos.VoucherInfoDTO, error)
}

type voucherService struct {
	txnManager      *utils.TransactionManager
	voucherInfoRepo repository.VoucherInfoRepository
	gcsService      utils.GCSService
}

func NewVoucherService(txnManager *utils.TransactionManager, voucherInfoRepo repository.VoucherInfoRepository, gcsService utils.GCSService) VoucherService {
	return &voucherService{
		txnManager:      txnManager,
		voucherInfoRepo: voucherInfoRepo,
		gcsService:      gcsService,
	}
}

func (s *voucherService) GetVoucherInfo(ctx context.Context, userId string, languageID int) ([]dtos.VoucherInfoDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	voucherInfos, err := s.voucherInfoRepo.GetVoucherInfo(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	voucherInfoDTOs := make([]dtos.VoucherInfoDTO, len(voucherInfos))
	for i, voucherInfo := range voucherInfos {
		winnerRewardsLanguage, err := s.voucherInfoRepo.FindTranslation(ctx, tx, languageID, voucherInfo.WinnerReward.ID)
		if err != nil {
			return nil, err
		}

		if winnerRewardsLanguage == nil {
			winnerRewardsLanguage, err = s.voucherInfoRepo.FindTranslation(ctx, tx, constants.DEFAULT_LANGUAGE_ID, voucherInfo.WinnerReward.ID)
			if err != nil {
				return nil, err
			}
			if winnerRewardsLanguage == nil {
				return nil, errs.NewInternalServerError("Translation not found", err)
			}
		}

		imagePath := constants.CONTEST_GCP_PREFIX + voucherInfo.WinnerReward.ImageURL
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, imagePath, 60*time.Minute)
		if err != nil {
			signedURL = ""
		}
		voucherInfoDTOs[i] = dtos.VoucherInfoDTO{
			ID:                  voucherInfo.ID,
			VoucherCode:         voucherInfo.VoucherCode,
			SecureCode:          voucherInfo.SecureCode,
			AssignedOn:          voucherInfo.AssignedOn.Format(time.RFC3339),
			SecureCodeAvailable: voucherInfo.SecureCodeAvailable,
			LastDateToRedeem:    voucherInfo.LastDateToRedeem.Format(time.RFC3339),
			VoucherName:         winnerRewardsLanguage.Name,
			VoucherImage:        signedURL,
			VoucherDescription:  winnerRewardsLanguage.Description,
			VoucherURL:          voucherInfo.WinnerReward.IconURL,
			ButtonValue:         winnerRewardsLanguage.ButtonValue,
		}
	}

	s.txnManager.CommitTxn(tx)

	return voucherInfoDTOs, nil
}
