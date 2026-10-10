package services

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type JokeFormatService interface {
	GetJokeFormats(ctx context.Context, filter dtos.JokeFormatFilter) (*dtos.JokeFormatPaginatedResponse, error)
}

type jokeFormatService struct {
	txnManager *utils.TransactionManager
	formatRepo repository.JokeFormatRepository
}

func NewJokeFormatService(txnManager *utils.TransactionManager, formatRepo repository.JokeFormatRepository) JokeFormatService {
	return &jokeFormatService{
		txnManager: txnManager,
		formatRepo: formatRepo,
	}
}

func (s *jokeFormatService) GetJokeFormats(ctx context.Context, filter dtos.JokeFormatFilter) (*dtos.JokeFormatPaginatedResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	formats, total, err := s.formatRepo.GetJokeFormats(ctx, tx, filter)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Convert to response DTOs
	formatResponses := make([]dtos.JokeFormatResponse, len(formats))
	for i, format := range formats {
		formatResponses[i] = dtos.JokeFormatResponse{
			ID:       format.ID,
			Value:    format.Value,
			Title:    format.Title,
			IconURL:  format.IconURL,
			IsActive: format.IsActive,
		}
	}

	s.txnManager.CommitTxn(tx)

	jokeResponse := &dtos.JokeFormatPaginatedResponse{
		Formats: formatResponses,
		Total:   total,
	}
	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		jokeResponse.Page = *filter.Page
		jokeResponse.PageSize = *filter.PageSize
	}

	return jokeResponse, nil
}
