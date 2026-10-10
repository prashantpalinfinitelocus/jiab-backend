package repository

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type UserNotificationRepository interface {
	GenericRepository[entities.UserNotification]
	GetUserNotifications(ctx context.Context, tx *gorm.DB, userId string, filter dtos.NotificationFilter) ([]entities.UserNotification, int, error)
	// FindUserIDsNotifiedWithinWindow bulk-checks a whole batch of user IDs in
	// one query, returning the subset that already received a notification of
	// the given type within `window`. Used to enforce per-user send-frequency
	// caps (e.g. once/day, once/week) without one query per user.
	FindUserIDsNotifiedWithinWindow(ctx context.Context, tx *gorm.DB, userIDs []string, notificationTypeID int, window time.Duration) (map[string]bool, error)
	MarkAsRead(ctx context.Context, tx *gorm.DB, userId string) error
}

type userNotificationRepository struct {
	*GormRepository[entities.UserNotification]
}

func NewUserNotificationRepository() UserNotificationRepository {
	return &userNotificationRepository{
		GormRepository: NewGormRepository[entities.UserNotification](),
	}
}

func (r *userNotificationRepository) GetUserNotifications(ctx context.Context, tx *gorm.DB, userId string, filter dtos.NotificationFilter) ([]entities.UserNotification, int, error) {
	var totalCount int64
	err := tx.WithContext(ctx).Model(&entities.UserNotification{}).Where("user_id = ?", userId).Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	query := tx.WithContext(ctx).Model(&entities.UserNotification{}).Where("user_id = ?", userId).Order("created_on desc").Preload("NotificationType")

	// Apply pagination only if both page number and page size are set
	if filter.PageNumber > 0 && filter.PageSize > 0 {
		offset := (filter.PageNumber - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var notifications []entities.UserNotification
	err = query.Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}

	return notifications, int(totalCount), nil
}

// FindUserIDsNotifiedWithinWindow returns the subset of userIDs that already
// received notificationTypeID within the last `window` duration, in a single
// query — avoids one query per user when checking a whole batch.
func (r *userNotificationRepository) FindUserIDsNotifiedWithinWindow(ctx context.Context, tx *gorm.DB, userIDs []string, notificationTypeID int, window time.Duration) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(userIDs) == 0 {
		return result, nil
	}

	since := time.Now().Add(-window)

	var notifiedIDs []string
	err := tx.WithContext(ctx).Model(&entities.UserNotification{}).
		Distinct("user_id").
		Where("user_id IN ?", userIDs).
		Where("notification_type_id = ?", notificationTypeID).
		Where("created_on >= ?", since).
		Where("is_deleted = ? and is_active = ?", false, true).
		Pluck("user_id", &notifiedIDs).Error
	if err != nil {
		return nil, err
	}

	for _, id := range notifiedIDs {
		result[id] = true
	}
	return result, nil
}

func (r *userNotificationRepository) MarkAsRead(ctx context.Context, tx *gorm.DB, userId string) error {
	now := time.Now()
	err := tx.WithContext(ctx).Model(&entities.UserNotification{}).Where("user_id = ?", userId).Where("is_read = ?", false).Update("is_read", true).Update("is_new", false).Update("last_modified_on", &now).Error
	if err != nil {
		return err
	}
	return nil
}
