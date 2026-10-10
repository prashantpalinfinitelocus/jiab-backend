package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type MemeTemplateRepository interface {
	GenericRepository[entities.MemeTemplate]
	// SearchTemplates returns active, non-deleted templates whose genre
	// case-insensitively equals query, or whose original filename
	// case-insensitively contains it, most recently added first.
	SearchTemplates(ctx context.Context, db *gorm.DB, query string, limit, offset int) ([]entities.MemeTemplate, error)
	// RandomTemplates returns up to limit random active, non-deleted
	// templates, optionally restricted to a genre.
	RandomTemplates(ctx context.Context, db *gorm.DB, genre string, limit int) ([]entities.MemeTemplate, error)
	// DistinctGenres returns the distinct genre values present among active,
	// non-deleted templates, used to power autocomplete suggestions.
	DistinctGenres(ctx context.Context, db *gorm.DB) ([]string, error)
}

type memeTemplateRepository struct {
	*GormRepository[entities.MemeTemplate]
}

func NewMemeTemplateRepository() MemeTemplateRepository {
	return &memeTemplateRepository{
		GormRepository: NewGormRepository[entities.MemeTemplate](),
	}
}

func (r *memeTemplateRepository) SearchTemplates(ctx context.Context, db *gorm.DB, query string, limit, offset int) ([]entities.MemeTemplate, error) {
	var templates []entities.MemeTemplate
	q := db.WithContext(ctx).
		Where("is_active = ? AND is_deleted = ?", true, false)

	if query != "" {
		q = q.Where("genre ILIKE ? OR original_filename ILIKE ?", query, "%"+query+"%")
	}

	err := q.Order("created_on DESC").Limit(limit).Offset(offset).Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *memeTemplateRepository) RandomTemplates(ctx context.Context, db *gorm.DB, genre string, limit int) ([]entities.MemeTemplate, error) {
	var templates []entities.MemeTemplate
	q := db.WithContext(ctx).
		Where("is_active = ? AND is_deleted = ?", true, false)

	if genre != "" {
		q = q.Where("genre = ?", genre)
	}

	err := q.Order("RANDOM()").Limit(limit).Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *memeTemplateRepository) DistinctGenres(ctx context.Context, db *gorm.DB) ([]string, error) {
	var genres []string
	err := db.WithContext(ctx).Model(&entities.MemeTemplate{}).
		Where("is_active = ? AND is_deleted = ?", true, false).
		Distinct("genre").
		Order("genre ASC").
		Pluck("genre", &genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}
