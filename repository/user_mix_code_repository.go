package repository

import (
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type UserMixCodeRepository struct {
	db *gorm.DB
}

func NewUserMixCodeRepository(db *gorm.DB) *UserMixCodeRepository {
	return &UserMixCodeRepository{db: db}
}

func (r *UserMixCodeRepository) Create(mixCode *entities.UserMixCodeDetails) error {
	return r.db.Create(mixCode).Error
}

func (r *UserMixCodeRepository) GetByMixCode(mixCode string) (*entities.UserMixCodeDetails, error) {
	var result entities.UserMixCodeDetails
	err := r.db.Where("mix_code = ? AND is_deleted = false", mixCode).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *UserMixCodeRepository) GetDailyMixCodeCount(userID string) (int64, int64, error) {
	var invalidCount int64
	var validCount int64
	startOfDay := time.Now().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.Model(&entities.UserMixCodeDetails{}).
		Where("user_id = ? AND created_on >= ? AND created_on < ? AND is_deleted = ? AND (is_valid_mix_code = ? OR is_already_redeemed_mix_code = ? OR is_expired_mix_code = ? OR is_active_mix_code = ?)",
			userID, startOfDay, endOfDay, false, false, true, true, false).
		Count(&invalidCount).Error
	if err != nil {
		return 0, 0, errs.NewInternalServerError("Failed to get daily mix code count", err)
	}

	err = r.db.Model(&entities.UserMixCodeDetails{}).
		Where("user_id = ? AND created_on >= ? AND created_on < ? AND is_deleted = ? AND (is_valid_mix_code = ? AND is_redeemed_successfully = ?)",
			userID, startOfDay, endOfDay, false, true, true).
		Count(&validCount).Error
	if err != nil {
		return 0, 0, errs.NewInternalServerError("Failed to get daily mix code count", err)
	}

	return invalidCount, validCount, err
}

func (r *UserMixCodeRepository) Update(mixCode *entities.UserMixCodeDetails) error {
	return r.db.Save(mixCode).Error
}
