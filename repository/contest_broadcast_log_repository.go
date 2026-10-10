package repository

import (
	"context"
	"errors"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type ContestBroadcastLogRepository interface {
	GenericRepository[entities.ContestBroadcastLog]
	FindByContestAndType(ctx context.Context, db *gorm.DB, contestID, announcementType string) (*entities.ContestBroadcastLog, error)
}

type contestBroadcastLogRepository struct {
	*GormRepository[entities.ContestBroadcastLog]
}

func NewContestBroadcastLogRepository() ContestBroadcastLogRepository {
	return &contestBroadcastLogRepository{
		GormRepository: NewGormRepository[entities.ContestBroadcastLog](),
	}
}

func (r *contestBroadcastLogRepository) FindByContestAndType(ctx context.Context, db *gorm.DB, contestID, announcementType string) (*entities.ContestBroadcastLog, error) {
	var log entities.ContestBroadcastLog
	if err := db.WithContext(ctx).
		Where("contest_id = ? AND announcement_type = ?", contestID, announcementType).
		First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}
