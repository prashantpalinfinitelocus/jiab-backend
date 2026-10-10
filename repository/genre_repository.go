package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/sirupsen/logrus"
)

type GenreRepository interface {
	GenericRepository[entities.JokeGenre]
	GetGenres(ctx context.Context, db *gorm.DB, filter dtos.GenreFilter) ([]entities.JokeGenre, int64, error)
	GetGenreByID(ctx context.Context, db *gorm.DB, id int) (*entities.JokeGenre, error)
}

type genreRepository struct {
	*GormRepository[entities.JokeGenre]
}

func NewGenreRepository() GenreRepository {
	return &genreRepository{
		GormRepository: NewGormRepository[entities.JokeGenre](),
	}
}

func (r *genreRepository) GetGenres(ctx context.Context, db *gorm.DB, filter dtos.GenreFilter) ([]entities.JokeGenre, int64, error) {
	var genres []entities.JokeGenre
	var total int64

	query := db.WithContext(ctx).Model(&entities.JokeGenre{}).Where("joke_genres.is_deleted = ?", false)

	if filter.Search != nil && *filter.Search != "" {
		search := strings.ToLower(*filter.Search)
		query = query.Where("LOWER(joke_genres.genre) LIKE ?", fmt.Sprintf("%%%s%%", search))
	}

	// Apply IsActive filter - default to true if not specified
	isActive := true
	if filter.IsActive != nil {
		isActive = *filter.IsActive
	}
	query = query.Where("joke_genres.is_active = ?", isActive)

	if filter.LanguageID != nil {
		query = query.Joins("JOIN joke_genre_language jgl ON joke_genres.id = jgl.joke_genre_id").
			Where("jgl.language_master_id = ?", *filter.LanguageID)
	}

	err := query.Count(&total).Error
	if err != nil {
		logrus.Errorf("error counting genres: %v", err)
		return nil, 0, errs.NewInternalServerError("error counting genres", err)
	}

	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		offset := (*filter.Page - 1) * *filter.PageSize
		query = query.Offset(offset).Limit(*filter.PageSize)
	}

	err = query.Find(&genres).Error
	if err != nil {
		logrus.Errorf("error finding genres: %v", err)
		return nil, 0, errs.NewInternalServerError("error finding genres", err)
	}

	return genres, total, nil
}

func (r *genreRepository) GetGenreByID(ctx context.Context, db *gorm.DB, id int) (*entities.JokeGenre, error) {
	var genre entities.JokeGenre

	err := db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, false).First(&genre).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logrus.Errorf("error finding genre by ID: %v", err)
		return nil, errs.NewInternalServerError("error finding genre by ID", err)
	}

	return &genre, nil
}
