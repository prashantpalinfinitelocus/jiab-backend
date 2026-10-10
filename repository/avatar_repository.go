package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type AvatarRepository struct {
	db *gorm.DB
}

func NewAvatarRepository(db *gorm.DB) *AvatarRepository {
	return &AvatarRepository{
		db: db,
	}
}

func (r *AvatarRepository) FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.Avatar, error) {
	var avatar entities.Avatar
	if err := tx.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&avatar).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &avatar, nil
}

func (r *AvatarRepository) FindByIDs(ctx context.Context, tx *gorm.DB, ids []int) ([]entities.Avatar, error) {
	var avatars []entities.Avatar
	if err := tx.WithContext(ctx).Where("id IN ? AND is_deleted = false", ids).Find(&avatars).Error; err != nil {
		return nil, err
	}
	return avatars, nil
}

func (r *AvatarRepository) FindAllActive(ctx context.Context, tx *gorm.DB) ([]entities.Avatar, error) {
	var avatars []entities.Avatar
	if err := tx.WithContext(ctx).Where("is_active = true AND is_deleted = false").Find(&avatars).Error; err != nil {
		return nil, err
	}
	return avatars, nil
}
