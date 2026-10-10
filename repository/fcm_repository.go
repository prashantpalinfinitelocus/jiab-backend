package repository

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type FCMDeviceTokenRepository interface {
	GenericRepository[entities.FCMDeviceToken]
	FindByDate(ctx context.Context, db *gorm.DB, date time.Time) ([]entities.FCMDeviceToken, error)
	InvalidateByDeviceToken(ctx context.Context, db *gorm.DB, deviceToken string) error
	BatchInvalidateByDeviceTokens(ctx context.Context, db *gorm.DB, deviceTokens []string) (int64, error)
}

type fcmDeviceTokenRepository struct {
	*GormRepository[entities.FCMDeviceToken]
}

func NewFCMDeviceTokenRepository() FCMDeviceTokenRepository {
	return &fcmDeviceTokenRepository{
		GormRepository: NewGormRepository[entities.FCMDeviceToken](),
	}
}

// InvalidateByDeviceToken marks a single device token inactive, e.g. after FCM
// reports it as permanently unregistered/invalid.
func (r *fcmDeviceTokenRepository) InvalidateByDeviceToken(ctx context.Context, db *gorm.DB, deviceToken string) error {
	now := time.Now()
	return db.WithContext(ctx).
		Model(&entities.FCMDeviceToken{}).
		Where("device_token = ?", deviceToken).
		Updates(map[string]interface{}{
			"is_active":        false,
			"last_modified_on": now,
		}).Error
}

// BatchInvalidateByDeviceTokens marks multiple device tokens inactive in one query.
func (r *fcmDeviceTokenRepository) BatchInvalidateByDeviceTokens(ctx context.Context, db *gorm.DB, deviceTokens []string) (int64, error) {
	if len(deviceTokens) == 0 {
		return 0, nil
	}
	now := time.Now()
	result := db.WithContext(ctx).
		Model(&entities.FCMDeviceToken{}).
		Where("device_token IN ?", deviceTokens).
		Updates(map[string]interface{}{
			"is_active":        false,
			"last_modified_on": now,
		})
	return result.RowsAffected, result.Error
}

func (r *fcmDeviceTokenRepository) FindByDate(ctx context.Context, db *gorm.DB, date time.Time) ([]entities.FCMDeviceToken, error) {
	var tokens []entities.FCMDeviceToken

	// Extract date components for comparison
	year, month, day := date.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, date.Location())
	endOfDay := time.Date(year, month, day, 23, 59, 59, 999999999, date.Location())

	result := db.WithContext(ctx).
		Where("last_modified_on >= ? AND last_modified_on <= ?", startOfDay, endOfDay).
		Find(&tokens)

	if result.Error != nil {
		return nil, result.Error
	}

	return tokens, nil
}
