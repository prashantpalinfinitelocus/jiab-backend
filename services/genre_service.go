package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type GenreService interface {
	GetGenres(ctx context.Context, filter dtos.GenreFilter) (*dtos.GenrePaginatedResponse, error)
	GetGenreByID(ctx context.Context, id int) (*dtos.JokeGenreResponse, error)
}

type genreService struct {
	txnManager *utils.TransactionManager
	genreRepo  repository.GenreRepository
	gcsService utils.GCSService
}

func NewGenreService(txnManager *utils.TransactionManager, genreRepo repository.GenreRepository, gcsService utils.GCSService) GenreService {
	return &genreService{
		txnManager: txnManager,
		genreRepo:  genreRepo,
		gcsService: gcsService,
	}
}

func (s *genreService) GetGenres(ctx context.Context, filter dtos.GenreFilter) (*dtos.GenrePaginatedResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Get genres from repository
	genres, total, err := s.genreRepo.GetGenres(ctx, tx, filter)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("error getting genres: %v", err)
	}

	// Convert to response DTOs
	genreResponses := make([]dtos.JokeGenreResponse, len(genres))
	for i, genre := range genres {
		imageURL := ""
		if genre.ImageURL != "" {
			signedURL, err := s.gcsService.GetFileSignedURL(ctx, genre.ImageURL, 24*time.Hour)
			if err != nil {
				imageURL = genre.ImageURL // fallback to raw path if signing fails
			} else {
				imageURL = signedURL
			}
		}
		genreResponses[i] = dtos.JokeGenreResponse{
			ID:            genre.ID,
			Genre:         genre.Genre,
			GenreEmoticon: genre.GenreEmoticon,
			ImageURL:      imageURL,
			EmoIsActive:   genre.EmoIsActive,
			EmoIndex:      genre.EmoIndex,
			IsActive:      genre.IsActive,
		}
	}

	s.txnManager.CommitTxn(tx)
	jokeResponse := dtos.GenrePaginatedResponse{
		Genres: genreResponses,
		Total:  total,
	}
	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		jokeResponse.Page = *filter.Page
		jokeResponse.PageSize = *filter.PageSize
	}

	return &jokeResponse, nil
}

func (s *genreService) GetGenreByID(ctx context.Context, id int) (*dtos.JokeGenreResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	genre, err := s.genreRepo.GetGenreByID(ctx, tx, id)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if genre == nil {
		return nil, nil
	}

	s.txnManager.CommitTxn(tx)

	imageURL := ""
	if genre.ImageURL != "" {
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, genre.ImageURL, 24*time.Hour)
		if err != nil {
			imageURL = genre.ImageURL // fallback to raw path if signing fails
		} else {
			imageURL = signedURL
		}
	}

	return &dtos.JokeGenreResponse{
		ID:            genre.ID,
		Genre:         genre.Genre,
		GenreEmoticon: genre.GenreEmoticon,
		ImageURL:      imageURL,
		EmoIsActive:   genre.EmoIsActive,
		EmoIndex:      genre.EmoIndex,
		IsActive:      genre.IsActive,
	}, nil
}
