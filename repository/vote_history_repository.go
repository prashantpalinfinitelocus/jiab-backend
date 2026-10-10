package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type VoteHistoryRepository interface {
	GenericRepository[entities.VoteHistory]
	GetVoteHistory(ctx context.Context, tx *gorm.DB, userId string, language string) (*entities.VoteHistory, error)
}

type voteHistoryRepository struct {
	*GormRepository[entities.VoteHistory]
}

func NewVoteHistoryRepo() VoteHistoryRepository {
	return &voteHistoryRepository{
		GormRepository: NewGormRepository[entities.VoteHistory](),
	}
}

func (r *voteHistoryRepository) GetVoteHistory(ctx context.Context, tx *gorm.DB, userId string, language string) (*entities.VoteHistory, error) {
	var voteHistory entities.VoteHistory

	err := tx.Where("user_id = ? and language = ?", userId, language).
    Order("voted_at DESC").
    First(&voteHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("Failed to get vote history", err)
	}

	return &voteHistory, nil
}
