package services

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

// MemeTemplateAdminService manages the internally-hosted meme template
// catalog that replaced the Giphy integration.
type MemeTemplateAdminService interface {
	BulkRegister(ctx context.Context, req dtos.BulkRegisterMemeTemplatesRequest) (dtos.BulkRegisterMemeTemplatesResponse, error)
}

type memeTemplateAdminService struct {
	txnManager *utils.TransactionManager
	repo       repository.MemeTemplateRepository
}

func NewMemeTemplateAdminService(txnManager *utils.TransactionManager, repo repository.MemeTemplateRepository) MemeTemplateAdminService {
	return &memeTemplateAdminService{txnManager: txnManager, repo: repo}
}

func (s *memeTemplateAdminService) BulkRegister(ctx context.Context, req dtos.BulkRegisterMemeTemplatesRequest) (dtos.BulkRegisterMemeTemplatesResponse, error) {
	rows := make([]entities.MemeTemplate, 0, len(req.Templates))
	for _, t := range req.Templates {
		rows = append(rows, entities.MemeTemplate{
			Episode:          t.Episode,
			Genre:            t.Genre,
			GCSObjectPath:    t.GCSObjectPath,
			GCSURL:           t.GCSURL,
			OriginalFilename: t.OriginalFilename,
			ContentType:      t.ContentType,
			IsActive:         true,
			IsDeleted:        false,
		})
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return dtos.BulkRegisterMemeTemplatesResponse{}, err
	}

	if err := s.repo.CreateMany(ctx, tx, rows); err != nil {
		s.txnManager.AbortTxn(tx)
		return dtos.BulkRegisterMemeTemplatesResponse{}, err
	}

	s.txnManager.CommitTxn(tx)
	return dtos.BulkRegisterMemeTemplatesResponse{Registered: len(rows)}, nil
}
