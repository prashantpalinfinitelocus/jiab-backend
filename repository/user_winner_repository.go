package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type UserWinnerRepository interface {
	GenericRepository[entities.UserWinner]
	GetTopUsers(ctx context.Context, tx *gorm.DB, limit int, date *time.Time) ([]entities.UserWinner, error)
	GetUserRank(ctx context.Context, tx *gorm.DB, userID string, date *time.Time) (*entities.UserWinner, error)
}

type userWinnerRepository struct {
	*GormRepository[entities.UserWinner]
}

func NewUserWinnerRepository() UserWinnerRepository {
	return &userWinnerRepository{
		GormRepository: NewGormRepository[entities.UserWinner](),
	}
}

func (r *userWinnerRepository) GetTopUsers(ctx context.Context, tx *gorm.DB, limit int, date *time.Time) ([]entities.UserWinner, error) {
	var topUsers []entities.UserWinner
	query := tx.Table("user_winners").
		Joins("JOIN users ON users.id = user_winners.user_id").
		Joins("JOIN winner_rewards ON winner_rewards.id = user_winners.winner_reward_id").
		Where("user_winners.is_active = ? AND user_winners.is_deleted = ?", true, false)

	if date != nil {
		startOfDay := date.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24*time.Hour - time.Second)
		query = query.Where("user_winners.winner_time BETWEEN ? AND ?", startOfDay, endOfDay)
	}

	err := query.Order("user_winners.user_rank asc").
		Limit(limit).
		Select("user_winners.*, users.phone_number as mobile, users.avatar_id, winner_rewards.image_url as winner_image_url, winner_rewards.description as winner_description").
		Find(&topUsers).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return topUsers, nil
}

func (r *userWinnerRepository) GetUserRank(ctx context.Context, tx *gorm.DB, userID string, date *time.Time) (*entities.UserWinner, error) {
	var userWinner entities.UserWinner
	query := tx.WithContext(ctx).
		Joins("JOIN users ON users.id = user_winners.user_id").
		Joins("JOIN winner_rewards ON winner_rewards.id = user_winners.winner_reward_id").
		Where("user_winners.user_id = ? AND user_winners.is_active = ? AND user_winners.is_deleted = ?", userID, true, false)

	if date != nil {
		startOfDay := date.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24*time.Hour - time.Second)
		query = query.Where("user_winners.winner_time BETWEEN ? AND ?", startOfDay, endOfDay)
	}

	err := query.Select("user_winners.*, users.phone_number as mobile, users.avatar_id, winner_rewards.image_url as winner_image_url, winner_rewards.description as winner_description").
		First(&userWinner).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &userWinner, nil
}
