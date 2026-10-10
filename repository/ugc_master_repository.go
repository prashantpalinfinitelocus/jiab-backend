package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
)

type UGCMasterRepository interface {
	GenericRepository[entities.UGCMaster]
	GetUGCContentByIDs(ctx context.Context, tx *gorm.DB, ids []string, fetchOnlyInactive bool) ([]entities.UGCMaster, error)
	GetUGCContentByUserID(ctx context.Context, tx *gorm.DB, userId string) ([]entities.UGCMaster, error)
	GetUGCContentWithArtist(ctx context.Context, tx *gorm.DB, ids []string) ([]entities.UGCMaster, error)
	GetCountOfUGCContent(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	UpdateUGCStatus(ctx context.Context, tx *gorm.DB, assetID string, statusID int, isActive bool) error
	RemoveUGC(ctx context.Context, tx *gorm.DB, assetID string) error
	GetUGCContent(ctx context.Context, tx *gorm.DB, userId string, platformID int, formatID int) ([]entities.UGCMaster, error)
	GetUGCAssessmentContent(ctx context.Context, tx *gorm.DB, filter dtos.UGCAssessmentFilter) ([]entities.UGCMaster, int64, error)
}

type ugcMasterRepository struct {
	*GormRepository[entities.UGCMaster]
}

func NewUGCMasterRepository() UGCMasterRepository {
	return &ugcMasterRepository{
		GormRepository: NewGormRepository[entities.UGCMaster](),
	}
}

func (r *ugcMasterRepository) GetUGCContentByIDs(ctx context.Context, tx *gorm.DB, ids []string, fetchOnlyInactive bool) ([]entities.UGCMaster, error) {
	var ugcContent []entities.UGCMaster
	if err := tx.WithContext(ctx).Model(&entities.UGCMaster{}).Where("asset_id IN (?) and is_deleted = ? and is_active = ?", ids, false, !fetchOnlyInactive).Preload("Format").Find(&ugcContent).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get ugc content by ids", err)
	}

	return ugcContent, nil
}

func (r *ugcMasterRepository) GetUGCContentByUserID(ctx context.Context, tx *gorm.DB, userId string) ([]entities.UGCMaster, error) {
	ugcContent := []entities.UGCMaster{}
	if err := tx.WithContext(ctx).
		Model(&entities.UGCMaster{}).
		Select("ugc_master.*").
		Joins("join joke_formats on ugc_master.format = joke_formats.id").
		Where("user_id = ? and ugc_master.is_deleted = ?", userId, false).
		Preload("Format").
		Preload("Status").
		Find(&ugcContent).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get ugc content by user id", err)
	}
	return ugcContent, nil
}

func (r *ugcMasterRepository) GetUGCContentWithArtist(ctx context.Context, tx *gorm.DB, ids []string) ([]entities.UGCMaster, error) {
	ugcContent := []entities.UGCMaster{}
	if err := tx.WithContext(ctx).
		Model(&entities.UGCMaster{}).
		Select("ugc_master.*, users.name as artist_name").
		Joins("join users on ugc_master.user_id = users.id").
		Where("asset_id IN (?) and ugc_master.is_deleted = ? and ugc_master.is_active = ?", ids, false, true).
		Preload("User").
		Find(&ugcContent).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get ugc content by artist name", err)
	}
	return ugcContent, nil
}

func (r *ugcMasterRepository) GetCountOfUGCContent(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	var count int64
	err := tx.WithContext(ctx).Model(&entities.UGCMaster{}).Where("user_id = ? and is_deleted = ? and created_on BETWEEN ? AND ?", userId, false, startTime, endTime).Count(&count).Error
	if err != nil {
		return 0, errs.NewInternalServerError("failed to get count of ugc content", err)
	}
	return int(count), nil
}

func (r *ugcMasterRepository) UpdateUGCStatus(ctx context.Context, tx *gorm.DB, assetID string, statusID int, isActive bool) error {
	now := time.Now()
	ugcMaster := &entities.UGCMaster{
		StatusID:       statusID,
		LastModifiedOn: &now,
		IsActive:       isActive,
	}
	if err := tx.WithContext(ctx).Model(&entities.UGCMaster{}).Where("asset_id = ?", assetID).Updates(ugcMaster).Error; err != nil {
		return errs.NewInternalServerError("failed to update ugc status", err)
	}
	return nil
}

func (r *ugcMasterRepository) RemoveUGC(ctx context.Context, tx *gorm.DB, assetID string) error {
	now := time.Now()
	ugcMaster := &entities.UGCMaster{
		IsActive:       false,
		IsDeleted:      true,
		LastModifiedOn: &now,
	}
	if err := tx.WithContext(ctx).Model(&entities.UGCMaster{}).Where("asset_id = ?", assetID).Updates(ugcMaster).Error; err != nil {
		return errs.NewInternalServerError("failed to remove ugc", err)
	}
	return nil
}

func (r *ugcMasterRepository) GetUGCContent(ctx context.Context, tx *gorm.DB, userId string, platformID int, formatID int) ([]entities.UGCMaster, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)

	var ugcJokes []entities.UGCMaster
	err := tx.WithContext(ctx).Model(&entities.UGCMaster{}).
		Where("user_id = ? AND platform_id = ? AND format = ? AND is_deleted = ? AND is_active = ? AND created_on BETWEEN ? AND ?",
			userId, platformID, formatID, false, true, startTime, endTime).
		Find(&ugcJokes).Error

	if err != nil {
		return nil, errs.NewInternalServerError("failed to get today's ugc jokes by user platform format", err)
	}

	return ugcJokes, nil
}

func (r *ugcMasterRepository) GetUGCAssessmentContent(ctx context.Context, tx *gorm.DB, filter dtos.UGCAssessmentFilter) ([]entities.UGCMaster, int64, error) {
	var ugcContent []entities.UGCMaster
	var totalCount int64

	query := tx.WithContext(ctx).Model(&entities.UGCMaster{}).
		Where("is_deleted = ? AND created_on BETWEEN ? AND ? AND is_active = ?", false, filter.StartDate, filter.EndDate, false).
		Debug()

	if filter.LanguageID != nil {
		query = query.Where("language_id = ?", *filter.LanguageID)
	}
	if filter.GenreID != nil {
		query = query.Where("joke_genre_id = ?", *filter.GenreID)
	}
	if filter.Source != nil {
		query = query.Where("platform_id = ?", *filter.Source)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, errs.NewInternalServerError("failed to get UGC assessment content count", err)
	}

	if err := query.
		Preload("Language").
		Preload("Genre").
		Preload("Format").
		Preload("Status").
		Preload("User").
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Order("created_on DESC").
		Find(&ugcContent).Error; err != nil {
		return nil, 0, errs.NewInternalServerError("failed to get UGC assessment content", err)
	}

	return ugcContent, totalCount, nil
}
