package repository

import (
	"context"
	"strings"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type JokeFormatRepository interface {
	GetJokeFormats(ctx context.Context, db *gorm.DB, filter dtos.JokeFormatFilter) ([]entities.JokeFormat, int64, error)
	GetJokeFormatByTitle(ctx context.Context, db *gorm.DB, title string) (entities.JokeFormat, error)
}

type jokeFormatRepository struct {
	*GormRepository[entities.JokeFormat]
}

func NewJokeFormatRepository() JokeFormatRepository {
	return &jokeFormatRepository{
		GormRepository: NewGormRepository[entities.JokeFormat](),
	}
}

func (r *jokeFormatRepository) GetJokeFormats(ctx context.Context, db *gorm.DB, filter dtos.JokeFormatFilter) ([]entities.JokeFormat, int64, error) {
	var formats []entities.JokeFormat
	var total int64

	query := db.WithContext(ctx).Model(&entities.JokeFormat{}).
		Where("is_deleted = ?", false)

	if filter.Search != nil && *filter.Search != "" {
		search := strings.ToLower(*filter.Search)
		query = query.Where("LOWER(value) LIKE ? OR LOWER(title) LIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	err := query.Count(&total).Error
	if err != nil {
		logrus.Errorf("error counting joke formats: %v", err)
		return nil, 0, errs.NewInternalServerError("error counting joke formats", err)
	}

	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		offset := (*filter.Page - 1) * *filter.PageSize
		query = query.Offset(offset).Limit(*filter.PageSize)
	}

	err = query.Order("id").Find(&formats).Error
	if err != nil {
		logrus.Errorf("error fetching joke formats: %v", err)
		return nil, 0, errs.NewInternalServerError("error fetching joke formats", err)
	}

	return formats, total, nil
}

func (r *jokeFormatRepository) GetJokeFormatByTitle(ctx context.Context, db *gorm.DB, title string) (entities.JokeFormat, error) {
	var format entities.JokeFormat
	err := db.WithContext(ctx).Model(&entities.JokeFormat{}).
		Where("title = ?", title).
		First(&format).Error
	return format, err
}
