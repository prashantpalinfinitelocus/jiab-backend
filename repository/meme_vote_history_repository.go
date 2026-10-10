package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type MemeVoteHistoryRepository interface {
	GenericRepository[entities.MemeVoteHistory]
	GetMemeVoteHistory(ctx context.Context, tx *gorm.DB, userId string, language string) (*entities.MemeVoteHistory, error)
	GetMemeVoteHistoryByAsset(ctx context.Context, tx *gorm.DB, userId string, assetID string) (*entities.MemeVoteHistory, error)
	CheckUserVotedMemes(ctx context.Context, tx *gorm.DB, userId string, assetIDs []string) (map[string]bool, error)
}

type memeVoteHistoryRepository struct {
	*GormRepository[entities.MemeVoteHistory]
}

func NewMemeVoteHistoryRepo() MemeVoteHistoryRepository {
	return &memeVoteHistoryRepository{
		GormRepository: NewGormRepository[entities.MemeVoteHistory](),
	}
}

func (r *memeVoteHistoryRepository) GetMemeVoteHistory(ctx context.Context, tx *gorm.DB, userId string, language string) (*entities.MemeVoteHistory, error) {
	var voteHistory entities.MemeVoteHistory

	err := tx.Where("user_id = ? and language = ?", userId, language).
		Order("voted_at DESC").
		First(&voteHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("Failed to get meme vote history", err)
	}

	return &voteHistory, nil
}

func (r *memeVoteHistoryRepository) GetMemeVoteHistoryByAsset(ctx context.Context, tx *gorm.DB, userId string, assetID string) (*entities.MemeVoteHistory, error) {
	var voteHistory entities.MemeVoteHistory

	err := tx.Where("user_id = ? and asset_id = ?", userId, assetID).First(&voteHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("Failed to get meme vote history by asset", err)
	}

	return &voteHistory, nil
}

func (r *memeVoteHistoryRepository) CheckUserVotedMemes(ctx context.Context, tx *gorm.DB, userId string, assetIDs []string) (map[string]bool, error) {
	if len(assetIDs) == 0 {
		return map[string]bool{}, nil
	}

	var voteHistories []entities.MemeVoteHistory
	err := tx.Where("user_id = ? and asset_id IN ?", userId, assetIDs).Find(&voteHistories).Error
	if err != nil {
		return nil, errs.NewInternalServerError("Failed to check voted memes", err)
	}

	votedMap := make(map[string]bool)
	for _, vh := range voteHistories {
		votedMap[vh.AssetID] = true
	}

	return votedMap, nil
}
