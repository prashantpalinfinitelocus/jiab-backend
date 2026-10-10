package repository

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type JokeConsumptionRepository interface {
	GenericRepository[entities.JokeConsumptionByUser]
	GetSupriseMeJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string, platform int) (int, error)
	GetScrollAndLolJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	FetchScrollAndLolJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
	GetUserTotalJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error)
	ActivateJokeConsumption(ctx context.Context, tx *gorm.DB, userId string, jokeId string) error
	GetRefreshRingJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	FetchRefreshRingJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
	GetUserTotalRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error)
	GetSpicyTrayJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	FetchSpicyTrayJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
}

type jokeConsumptionRepository struct {
	*GormRepository[entities.JokeConsumptionByUser]
}

func NewJokeConsumptionRepository() JokeConsumptionRepository {
	return &jokeConsumptionRepository{
		GormRepository: NewGormRepository[entities.JokeConsumptionByUser](),
	}
}

func (r *jokeConsumptionRepository) GetSupriseMeJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string, platform int) (int, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	var count int64
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("user_id = ? AND joke_generated_at BETWEEN ? AND ? AND platform = ? AND joke_type = ?", userId, startTime, endTime, platform, entities.SupriseMeJoke).Count(&count).Error; err != nil {
		return 0, errs.NewInternalServerError("failed to get joke consumption by user id", err)
	}
	return int(count), nil
}

func (r *jokeConsumptionRepository) GetScrollAndLolJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	var count int64
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("user_id = ? AND joke_generated_at BETWEEN ? AND ? AND joke_type = ? AND is_active = ?", userId, startTime, endTime, entities.ScrollAndLolJoke, true).Count(&count).Error; err != nil {
		return 0, errs.NewInternalServerError("failed to get joke consumption by user id", err)
	}
	return int(count), nil
}

func (r *jokeConsumptionRepository) FetchScrollAndLolJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	var jokes []entities.JokeConsumptionByUser
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("joke_type = ? AND selected_language = ? AND user_id = ? AND joke_generated_at BETWEEN ? AND ?", entities.ScrollAndLolJoke, languageID, userId, startTime, endTime).Find(&jokes).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get joke consumption by user id", err)
	}
	return jokes, nil
}

func (r *jokeConsumptionRepository) GetUserTotalJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error) {
	var jokes []entities.JokeConsumptionByUser
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("joke_type = ? AND user_id = ? AND is_deleted = ? AND is_active = ? AND joke_generated_at BETWEEN ? AND ?", entities.ScrollAndLolJoke, userId, false, true, startTime, endTime).Find(&jokes).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get user total jokes", err)
	}
	return jokes, nil
}

func (r *jokeConsumptionRepository) ActivateJokeConsumption(ctx context.Context, tx *gorm.DB, userId string, jokeId string) error {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	result := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).
		Where("joke_type = ? AND user_id = ? AND joke_id = ? AND is_deleted = ? AND joke_generated_at BETWEEN ? AND ?", entities.ScrollAndLolJoke, userId, jokeId, false, startTime, endTime).
		Update("is_active", true)

	if result.Error != nil {
		return errs.NewInternalServerError("failed to activate joke consumption", result.Error)
	}

	if result.RowsAffected == 0 {
		return errs.NewNotFoundError("joke consumption record not found", nil)
	}

	return nil
}

func (r *jokeConsumptionRepository) GetRefreshRingJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	var count int64
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("user_id = ? AND joke_generated_at BETWEEN ? AND ? AND joke_type = ? AND is_active = ?", userId, startTime, endTime, entities.RefreshRingJoke, true).Count(&count).Error; err != nil {
		return 0, errs.NewInternalServerError("failed to get refresh ring joke consumption by user id", err)
	}
	return int(count), nil
}

func (r *jokeConsumptionRepository) FetchRefreshRingJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	var jokes []entities.JokeConsumptionByUser
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("joke_type = ? AND selected_language = ? AND user_id = ? AND joke_generated_at BETWEEN ? AND ?", entities.RefreshRingJoke, languageID, userId, startTime, endTime).Find(&jokes).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get refresh ring joke consumption by user id", err)
	}
	return jokes, nil
}

func (r *jokeConsumptionRepository) GetUserTotalRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error) {
	var jokes []entities.JokeConsumptionByUser
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("joke_type = ? AND user_id = ? AND is_deleted = ? AND is_active = ? AND joke_generated_at BETWEEN ? AND ?", entities.RefreshRingJoke, userId, false, false, startTime, endTime).Find(&jokes).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get user total refresh ring jokes", err)
	}
	return jokes, nil
}

func (r *jokeConsumptionRepository) GetSpicyTrayJokeConsumptionByUserID(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	var count int64
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("user_id = ? AND joke_generated_at BETWEEN ? AND ? AND joke_type = ? AND is_active = ?", userId, startTime, endTime, entities.SpicyTrayJoke, true).Count(&count).Error; err != nil {
		return 0, errs.NewInternalServerError("failed to get spicy tray joke consumption by user id", err)
	}
	return int(count), nil
}

func (r *jokeConsumptionRepository) FetchSpicyTrayJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	var jokes []entities.JokeConsumptionByUser
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	if err := tx.WithContext(ctx).Model(&entities.JokeConsumptionByUser{}).Where("joke_type = ? AND selected_language = ? AND user_id = ? AND joke_generated_at BETWEEN ? AND ?", entities.SpicyTrayJoke, languageID, userId, startTime, endTime).Find(&jokes).Error; err != nil {
		return nil, errs.NewInternalServerError("failed to get spicy tray joke consumption by user id", err)
	}
	return jokes, nil
}
