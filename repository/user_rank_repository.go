package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type UserRankRepository interface {
	GenericRepository[entities.UserRank]
	FindByUserID(ctx context.Context, tx *gorm.DB, userID string) (*entities.UserRank, error)
}

type userRankRepository struct {
	*GormRepository[entities.UserRank]
}

func NewUserRankRepository() UserRankRepository {
	return &userRankRepository{
		GormRepository: NewGormRepository[entities.UserRank](),
	}
}

func (r *userRankRepository) FindByUserID(ctx context.Context, tx *gorm.DB, userID string) (*entities.UserRank, error) {
	var userRank entities.UserRank
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&userRank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &userRank, nil
}
