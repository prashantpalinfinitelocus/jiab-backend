package repository

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type MemeReactionRepository interface {
	GenericRepository[entities.MemeReaction]
	CountUserDailyReactions(ctx context.Context, db *gorm.DB, userID string, day time.Time) (int, error)
	HasReacted(ctx context.Context, db *gorm.DB, userID string, memeID string) (bool, error)
	GetReactionCountsByMemeIDs(ctx context.Context, db *gorm.DB, memeIDs []string) (map[string]*dtos.ReactionCount, error)
	FindByUserAndMeme(ctx context.Context, db *gorm.DB, userID string, memeID string) (*entities.MemeReaction, error)
}

type memeReactionRepository struct {
	*GormRepository[entities.MemeReaction]
}

func NewMemeReactionRepository() MemeReactionRepository {
	return &memeReactionRepository{GormRepository: NewGormRepository[entities.MemeReaction]()}
}

func (r *memeReactionRepository) CountUserDailyReactions(ctx context.Context, db *gorm.DB, userID string, day time.Time) (int, error) {
	var count int64
	start := day.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	err := db.WithContext(ctx).Model(&entities.MemeReaction{}).Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, start, end).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *memeReactionRepository) HasReacted(ctx context.Context, db *gorm.DB, userID string, memeID string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&entities.MemeReaction{}).Where("user_id = ? AND meme_id = ?", userID, memeID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *memeReactionRepository) GetReactionCountsByMemeIDs(ctx context.Context, db *gorm.DB, memeIDs []string) (map[string]*dtos.ReactionCount, error) {
	if len(memeIDs) == 0 {
		return make(map[string]*dtos.ReactionCount), nil
	}

	var results []dtos.MemeReactionResult
	err := db.WithContext(ctx).Model(&entities.MemeReaction{}).
		Select("meme_id, reaction, COUNT(*) as count").
		Where("meme_id IN ?", memeIDs).
		Group("meme_id, reaction").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	reactionCounts := make(map[string]*dtos.ReactionCount)
	for _, memeID := range memeIDs {
		reactionCounts[memeID] = &dtos.ReactionCount{
			MemeID:  memeID,
			Happy:   0,
			Sad:     0,
			Neutral: 0,
		}
	}

	for _, r := range results {
		if rc, exists := reactionCounts[r.MemeID]; exists {
			switch r.Reaction {
			case "happy":
				rc.Happy = r.Count
			case "sad":
				rc.Sad = r.Count
			case "neutral":
				rc.Neutral = r.Count
			}
		}
	}

	return reactionCounts, nil
}

func (r *memeReactionRepository) FindByUserAndMeme(ctx context.Context, db *gorm.DB, userID string, memeID string) (*entities.MemeReaction, error) {
	var reaction entities.MemeReaction
	err := db.WithContext(ctx).Where("user_id = ? AND meme_id = ?", userID, memeID).First(&reaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &reaction, nil
}
