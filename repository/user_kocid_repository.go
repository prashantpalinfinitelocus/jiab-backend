package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type UserKOCIDRepository interface {
	GetByUserID(ctx context.Context, userID string) (*entities.UserKOCID, error)
	Create(ctx context.Context, tx *gorm.DB, kocid *entities.UserKOCID) error
	Update(ctx context.Context, tx *gorm.DB, kocid *entities.UserKOCID) error
	GetByHashedKOCID(ctx context.Context, hashedKOCID string) (*entities.UserKOCID, error)
}

type userKOCIDRepository struct {
	db *gorm.DB
}

func NewUserKOCIDRepository(db *gorm.DB) UserKOCIDRepository {
	return &userKOCIDRepository{db: db}
}

func (r *userKOCIDRepository) GetByUserID(ctx context.Context, userID string) (*entities.UserKOCID, error) {
	var kocid entities.UserKOCID

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&kocid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &kocid, nil
}

func (r *userKOCIDRepository) Create(ctx context.Context, tx *gorm.DB, kocid *entities.UserKOCID) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	return db.WithContext(ctx).Create(kocid).Error
}

func (r *userKOCIDRepository) Update(ctx context.Context, tx *gorm.DB, kocid *entities.UserKOCID) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	kocid.UpdatedAt = time.Now()
	return db.WithContext(ctx).Save(kocid).Error
}

func (r *userKOCIDRepository) GetByHashedKOCID(ctx context.Context, hashedKOCID string) (*entities.UserKOCID, error) {
	var kocid entities.UserKOCID

	err := r.db.WithContext(ctx).
		Where("hashed_kocid = ?", hashedKOCID).
		First(&kocid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &kocid, nil
}
