package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type WAAssetActivityRepository struct {
	db *gorm.DB
}

func NewWAAssetActivityRepository(db *gorm.DB) WAAssetActivityRepository {
	return WAAssetActivityRepository{
		db: db,
	}
}

func (r *WAAssetActivityRepository) FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.WAAssetActivity, error) {
	var waAssetActivity entities.WAAssetActivity
	if err := tx.WithContext(ctx).Where("id = ?", id).First(&waAssetActivity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &waAssetActivity, nil
}

func (r *WAAssetActivityRepository) Create(ctx context.Context, tx *gorm.DB, waAssetActivity *entities.WAAssetActivity) error {
	return tx.Create(waAssetActivity).Error
}
