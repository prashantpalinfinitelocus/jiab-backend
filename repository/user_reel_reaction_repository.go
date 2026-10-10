package repository

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type UserReelReactionRepository interface {
	GenericRepository[entities.UserCreatorCollabReaction]
	GetUserReactionCount(ctx context.Context, db *gorm.DB, userID string) (int, error)
}

type userReelReactionRepository struct {
	*GormRepository[entities.UserCreatorCollabReaction]
}

func NewUserReelReactionRepository() UserReelReactionRepository {
	return &userReelReactionRepository{
		GormRepository: NewGormRepository[entities.UserCreatorCollabReaction](),
	}
}

func (r *userReelReactionRepository) GetUserReactionCount(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	var count int64
	startTime := time.Now().Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)
	err := db.WithContext(ctx).Model(&entities.UserCreatorCollabReaction{}).Where("user_id = ? AND reaction_dttm BETWEEN ? AND ?", userID, startTime, endTime).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
