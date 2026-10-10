package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type MemeViewHistoryRepository interface {
	GenericRepository[entities.MemeViewHistory]
	HasViewed(ctx context.Context, db *gorm.DB, userID string, memeID string) (bool, error)
	CheckUserViewedMemes(ctx context.Context, db *gorm.DB, userID string, memeIDs []string) (map[string]bool, error)
}

type memeViewHistoryRepository struct {
	*GormRepository[entities.MemeViewHistory]
}

func NewMemeViewHistoryRepository() MemeViewHistoryRepository {
	return &memeViewHistoryRepository{
		GormRepository: NewGormRepository[entities.MemeViewHistory](),
	}
}

func (r *memeViewHistoryRepository) HasViewed(ctx context.Context, db *gorm.DB, userID string, memeID string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&entities.MemeViewHistory{}).Where("user_id = ? AND meme_id = ?", userID, memeID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *memeViewHistoryRepository) CheckUserViewedMemes(ctx context.Context, db *gorm.DB, userID string, memeIDs []string) (map[string]bool, error) {
	if len(memeIDs) == 0 {
		return make(map[string]bool), nil
	}

	var viewedMemes []string
	err := db.WithContext(ctx).Model(&entities.MemeViewHistory{}).
		Select("meme_id").
		Where("user_id = ? AND meme_id IN ?", userID, memeIDs).
		Pluck("meme_id", &viewedMemes).Error

	if err != nil {
		return nil, err
	}

	viewedMap := make(map[string]bool)
	for _, memeID := range memeIDs {
		viewedMap[memeID] = false
	}
	for _, memeID := range viewedMemes {
		viewedMap[memeID] = true
	}

	return viewedMap, nil
}
