package services

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type LanguageService interface {
	GetLanguages(ctx context.Context, filter dtos.LanguageFilter) (*dtos.LanguagePaginatedResponse, error)
	GetLanguageByID(ctx context.Context, id int) (*dtos.LanguageResponse, error)
}

type languageService struct {
	txnManager   *utils.TransactionManager
	languageRepo repository.LanguageRepository
}

func NewLanguageService(txnManager *utils.TransactionManager, languageRepo repository.LanguageRepository) LanguageService {
	return &languageService{
		txnManager:   txnManager,
		languageRepo: languageRepo,
	}
}

func (s *languageService) GetLanguages(ctx context.Context, filter dtos.LanguageFilter) (*dtos.LanguagePaginatedResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	languages, total, err := s.languageRepo.GetLanguages(ctx, tx, filter)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	var languageResponses []dtos.LanguageResponse
	for _, language := range languages {
		languageResponses = append(languageResponses, dtos.LanguageResponse{
			ID:          language.ID,
			Name:        language.Name,
			LanguageKey: language.LanguageKey,
			IsActive:    language.IsActive,
		})
	}

	s.txnManager.CommitTxn(tx)
	languageResponse := &dtos.LanguagePaginatedResponse{
		Languages: languageResponses,
		Total:     total,
	}
	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		languageResponse.Page = *filter.Page
		languageResponse.PageSize = *filter.PageSize
	}

	return languageResponse, nil
}

func (s *languageService) GetLanguageByID(ctx context.Context, id int) (*dtos.LanguageResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	language, err := s.languageRepo.GetLanguageByID(ctx, tx, id)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if language == nil {
		s.txnManager.CommitTxn(tx)
		return nil, nil
	}

	s.txnManager.CommitTxn(tx)

	return &dtos.LanguageResponse{
		ID:          language.ID,
		Name:        language.Name,
		LanguageKey: language.LanguageKey,
		IsActive:    language.IsActive,
	}, nil
}
