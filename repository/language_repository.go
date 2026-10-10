package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type LanguageRepository interface {
	GenericRepository[entities.Language]
	GetLanguages(ctx context.Context, db *gorm.DB, filter dtos.LanguageFilter) ([]entities.Language, int64, error)
	GetLanguageByID(ctx context.Context, db *gorm.DB, id int) (*entities.Language, error)
	FindAll(ctx context.Context, tx *gorm.DB) ([]entities.Language, error)
	FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.Language, error)
	FindAllActive(ctx context.Context, tx *gorm.DB) ([]entities.Language, error)
	FindLanguagesByGenreID(ctx context.Context, tx *gorm.DB, genreID int) ([]entities.Language, error)
	FindLanguageInfoByGenreID(ctx context.Context, tx *gorm.DB, genreID int) ([]dtos.JokeGenreLanguageListItem, error)
	FindLanguagesByIDs(ctx context.Context, db *gorm.DB, ids []int) (map[int]*entities.Language, error)
}

type languageRepository struct {
	*GormRepository[entities.Language]
}

func NewLanguageRepository() LanguageRepository {
	return &languageRepository{
		GormRepository: NewGormRepository[entities.Language](),
	}
}

func (r *languageRepository) GetLanguages(ctx context.Context, db *gorm.DB, filter dtos.LanguageFilter) ([]entities.Language, int64, error) {
	var languages []entities.Language
	var total int64

	query := db.WithContext(ctx).Model(&entities.Language{}).Where("is_deleted = ?", false)

	if filter.Search != nil && *filter.Search != "" {
		search := strings.ToLower(*filter.Search)
		query = query.Where("LOWER(name) LIKE ? OR LOWER(language_key) LIKE ?",
			fmt.Sprintf("%%%s%%", search),
			fmt.Sprintf("%%%s%%", search))
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	err := query.Count(&total).Error
	if err != nil {
		logrus.Errorf("error counting languages: %v", err)
		return nil, 0, errs.NewInternalServerError("error counting languages", err)
	}

	if filter.Page != nil && filter.PageSize != nil && *filter.Page > 0 && *filter.PageSize > 0 {
		offset := (*filter.Page - 1) * *filter.PageSize
		query = query.Offset(offset).Limit(*filter.PageSize)
	}

	err = query.Find(&languages).Error
	if err != nil {
		logrus.Errorf("error finding languages: %v", err)
		return nil, 0, errs.NewInternalServerError("error finding languages", err)
	}

	return languages, total, nil
}

func (r *languageRepository) GetLanguageByID(ctx context.Context, db *gorm.DB, id int) (*entities.Language, error) {
	var language entities.Language

	err := db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, false).First(&language).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logrus.Errorf("error finding language by ID: %v", err)
		return nil, errs.NewInternalServerError("error finding language by ID", err)
	}

	return &language, nil
}

func (r *languageRepository) FindAll(ctx context.Context, tx *gorm.DB) ([]entities.Language, error) {
	var languages []entities.Language
	err := tx.Find(&languages).Error
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func (r *languageRepository) FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.Language, error) {
	var language entities.Language
	err := tx.Where("id = ?", id).First(&language).Error
	if err != nil {
		return nil, err
	}
	return &language, nil
}

func (r *languageRepository) FindAllActive(ctx context.Context, tx *gorm.DB) ([]entities.Language, error) {
	var languages []entities.Language
	err := tx.Where("is_active = ?", true).Find(&languages).Error
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func (r *languageRepository) FindLanguagesByGenreID(ctx context.Context, tx *gorm.DB, genreID int) ([]entities.Language, error) {
	var languages []entities.Language
	err := tx.Table("language as l").
		Select("l.id as language_id, l.name as language_name, jgl.script as script").
		Joins("JOIN joke_genre_language as jgl ON l.id = jgl.language_id").
		Where("jgl.genre_id = ?", genreID).
		Scan(&languages).Error
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func (r *languageRepository) FindLanguageInfoByGenreID(ctx context.Context, tx *gorm.DB, genreID int) ([]dtos.JokeGenreLanguageListItem, error) {
	var result []dtos.JokeGenreLanguageListItem
	err := tx.Table("languages as l").
		Select("l.id as language_id, l.name as language_name, jgl.language_script as script, jgl.genre_emoticon as genre_emoticon").
		Joins("JOIN joke_genre_language as jgl ON l.id = jgl.language_id").
		Where("jgl.joke_genre_id = ?", genreID).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *languageRepository) FindLanguagesByIDs(ctx context.Context, db *gorm.DB, ids []int) (map[int]*entities.Language, error) {
	if len(ids) == 0 {
		return make(map[int]*entities.Language), nil
	}

	var languages []entities.Language
	err := db.WithContext(ctx).Where("id IN ? AND is_deleted = ? AND is_active = ?", ids, false, true).Find(&languages).Error
	if err != nil {
		return nil, err
	}

	languageMap := make(map[int]*entities.Language)
	for i := range languages {
		languageMap[languages[i].ID] = &languages[i]
	}

	return languageMap, nil
}
