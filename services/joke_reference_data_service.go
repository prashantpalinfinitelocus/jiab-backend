package services

import (
	"context"
	"fmt"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"gorm.io/gorm"
)

type JokeReferenceDataService interface {
	GetJokeReferenceData(ctx context.Context, tx *gorm.DB, genre *int, language *int) (string, string, error)
	GetJokeFormat(ctx context.Context, tx *gorm.DB, format string) (int, error)
	GetJokeGenres(ctx context.Context, tx *gorm.DB, userID string) ([]dtos.JokeGenreLanguageResponseDTO, error)
	GetJokeGenre(ctx context.Context, tx *gorm.DB, genreID *int) (string, error)
	GetJokeLanguage(ctx context.Context, tx *gorm.DB, languageID int) (string, error)
	GetAllGenres(ctx context.Context, tx *gorm.DB, filter dtos.GenreFilter) ([]entities.JokeGenre, int64, error)
	GetJokeLanguageKey(ctx context.Context, tx *gorm.DB, languageID int) (string, error)
}

type jokeReferenceDataService struct {
	languageRepo repository.LanguageRepository
	genreRepo    repository.GenreRepository
	formatRepo   repository.JokeFormatRepository
}

func NewJokeReferenceDataService(languageRepo repository.LanguageRepository, genreRepo repository.GenreRepository, formatRepo repository.JokeFormatRepository) JokeReferenceDataService {
	return &jokeReferenceDataService{
		languageRepo: languageRepo,
		genreRepo:    genreRepo,
		formatRepo:   formatRepo,
	}
}

func (s *jokeReferenceDataService) GetJokeReferenceData(ctx context.Context, tx *gorm.DB, genre *int, language *int) (string, string, error) {
	var genreResult string
	if genre != nil && *genre != 0 {
		genreEntity, err := s.genreRepo.GetGenreByID(ctx, tx, *genre)
		if err != nil {
			return "", "", err
		}
		if genreEntity == nil {
			genreResult = ""
		} else {
			genreResult = genreEntity.Genre
		}
	} else {
		genreResult = ""
	}

	languageEntity, err := s.languageRepo.GetLanguageByID(ctx, tx, *language)
	if err != nil {
		return "", "", err
	}

	return genreResult, languageEntity.Name, nil
}

func (s *jokeReferenceDataService) GetJokeFormat(ctx context.Context, tx *gorm.DB, format string) (int, error) {
	formatEntity, err := s.formatRepo.GetJokeFormatByTitle(ctx, tx, format)
	if err != nil {
		return 0, err
	}
	return formatEntity.ID, nil
}

func (s *jokeReferenceDataService) GetJokeGenre(ctx context.Context, tx *gorm.DB, genreID *int) (string, error) {
	if genreID == nil {
		return "", nil
	}
	genreEntity, err := s.genreRepo.GetGenreByID(ctx, tx, *genreID)
	if err != nil {
		return "", err
	}

	if genreEntity == nil {
		return "", nil
	}
	return genreEntity.Genre, nil
}

func (s *jokeReferenceDataService) GetJokeLanguage(ctx context.Context, tx *gorm.DB, languageID int) (string, error) {
	languageEntity, err := s.languageRepo.GetLanguageByID(ctx, tx, languageID)
	if err != nil {
		return "", err
	}

	if languageEntity == nil {
		return "", nil
	}
	return languageEntity.Name, nil
}

func (s *jokeReferenceDataService) GetJokeLanguageKey(ctx context.Context, tx *gorm.DB, languageID int) (string, error) {
	languageEntity, err := s.languageRepo.GetLanguageByID(ctx, tx, languageID)
	if err != nil {
		return "", err
	}
	return languageEntity.LanguageKey, nil
}


func (s *jokeReferenceDataService) GetJokeGenres(ctx context.Context, tx *gorm.DB, userID string) ([]dtos.JokeGenreLanguageResponseDTO, error) {
	isActive := true
	genres, _, err := s.genreRepo.GetGenres(ctx, tx, dtos.GenreFilter{IsActive: &isActive})
	if err != nil {
		return nil, err
	}

	var response []dtos.JokeGenreLanguageResponseDTO
	for _, genre := range genres {
		langList, err := s.languageRepo.FindLanguageInfoByGenreID(ctx, tx, int(genre.ID))
		if err != nil {
			return nil, err
		}

		// Modify language script by adding genre emoticon
		for i := range langList {
			langList[i].LanguageScript = fmt.Sprintf("%s@%s", genre.GenreEmoticon, langList[i].LanguageScript)
		}

		genre_name := fmt.Sprintf("%s@%s", genre.GenreEmoticon, genre.Genre)

		response = append(response, dtos.JokeGenreLanguageResponseDTO{
			ID:                    int(genre.ID),
			Genre:                 genre_name,
			ImageURL:              &genre.ImageURL,
			JokeGenreLanguageList: langList,
		})
	}

	return response, nil
}

func (s *jokeReferenceDataService) GetAllGenres(ctx context.Context, tx *gorm.DB, filter dtos.GenreFilter) ([]entities.JokeGenre, int64, error) {
	return s.genreRepo.GetGenres(ctx, tx, filter)
}
