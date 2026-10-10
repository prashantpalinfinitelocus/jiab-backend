package services

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type PlagiarizedInfoService interface {
	CreatePlagiarizedInfo(ctx context.Context, request dtos.PlagiarizedInfoRequest, userID string) (*dtos.PlagiarizedInfoResponse, error)
	RemovePlagiarizedUGC(ctx context.Context, request dtos.ChangePlagiarizedInfoStatusRequest) error
}

type plagiarizedInfoService struct {
	txnManager          *utils.TransactionManager
	plagiarizedInfoRepo repository.PlagiarizedInfoRepository
	ugcMasterRepo       repository.UGCMasterRepository
	gluedinService      GluedinService
	notificationService NotificationService
}

func NewPlagiarizedInfoService(txnManager *utils.TransactionManager, plagiarizedInfoRepo repository.PlagiarizedInfoRepository, ugcMasterRepo repository.UGCMasterRepository, notificationService NotificationService, gluedinService GluedinService) PlagiarizedInfoService {
	return &plagiarizedInfoService{
		txnManager:          txnManager,
		plagiarizedInfoRepo: plagiarizedInfoRepo,
		ugcMasterRepo:       ugcMasterRepo,
		notificationService: notificationService,
		gluedinService:      gluedinService,
	}
}

func (s *plagiarizedInfoService) CreatePlagiarizedInfo(ctx context.Context, request dtos.PlagiarizedInfoRequest, userID string) (*dtos.PlagiarizedInfoResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Error("error starting transaction", err)
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Create new record directly without checking for existing records
	plagiarizedInfo := &entities.PlagiarizedInfo{
		AssetID:     request.AssetID,
		UserID:      userID,
		Information: request.Information,
		IsActive:    true,
		IsDeleted:   false,
		CreatedBy:   userID,
		CreatedOn:   time.Now(),
	}

	err = s.plagiarizedInfoRepo.Create(ctx, tx, plagiarizedInfo)
	if err != nil {
		logrus.Error("error creating plagiarized info", err)
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_REPORTED, plagiarizedInfo.UserID, nil); err != nil {
		logrus.Error("error sending UGC reported notification", err)
	}

	s.txnManager.CommitTxn(tx)
	return &dtos.PlagiarizedInfoResponse{
		ID:          plagiarizedInfo.ID,
		AssetID:     plagiarizedInfo.AssetID,
		Information: plagiarizedInfo.Information,
		Message:     "Plagiarized information created successfully",
	}, nil
}

func (s *plagiarizedInfoService) RemovePlagiarizedUGC(ctx context.Context, req dtos.ChangePlagiarizedInfoStatusRequest) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Error("error starting transaction", err)
		return err
	}

	plagiarizedInfo, err := s.plagiarizedInfoRepo.FindById(ctx, tx, req.PlagiarizedInfoID)
	if err != nil {
		logrus.Error("error finding plagiarized info", err)
		s.txnManager.AbortTxn(tx)
		return err
	}
	if plagiarizedInfo == nil {
		logrus.Error("plagiarized info not found")
		s.txnManager.AbortTxn(tx)
		return errs.NewNotFoundError("Plagiarized info not found", nil)
	}

	err = s.plagiarizedInfoRepo.UpdateStatus(ctx, tx, req.PlagiarizedInfoID, false)
	if err != nil {
		logrus.Error("error updating plagiarized info status", err)
		s.txnManager.AbortTxn(tx)
		return err
	}
	ugcMaster, err := s.ugcMasterRepo.GetUGCContentByIDs(ctx, tx, []string{plagiarizedInfo.AssetID}, false)
	if err != nil {
		logrus.Error("error finding ugc master", err)
		s.txnManager.AbortTxn(tx)
		return err
	}

	if len(ugcMaster) == 0 {
		logrus.Error("ugc master not found")
		s.txnManager.AbortTxn(tx)
		return errs.NewNotFoundError("UGC master not found", nil)
	}

	err = s.gluedinService.ChangeUGCStatus(ugcMaster[0].AssetID, constants.UGC_STATUS_INACTIVE)
	if err != nil {
		logrus.Error("error changing ugc status", err)
		s.txnManager.AbortTxn(tx)
		return err
	}

	err = s.ugcMasterRepo.RemoveUGC(ctx, tx, ugcMaster[0].AssetID)
	if err != nil {
		logrus.Error("error removing ugc", err)
		s.txnManager.AbortTxn(tx)
		return err
	}

	reportedBy := plagiarizedInfo.UserID

	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_REMOVED, reportedBy, nil); err != nil {
		logrus.Error("error sending UGC removed notification", err)
	}

	s.txnManager.CommitTxn(tx)
	return nil
}
