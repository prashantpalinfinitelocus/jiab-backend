package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type MemeRepository interface {
	GenericRepository[entities.Meme]
	FindApprovedFeed(ctx context.Context, db *gorm.DB, userId string, languageID *int, q *string, limit int, cursor *string, trending *bool) ([]entities.Meme, *string, error)
	IncrementViewCount(ctx context.Context, db *gorm.DB, memeID string) error
	FindPending(ctx context.Context, db *gorm.DB, limit int, cursor *string) ([]entities.Meme, *string, error)
	FindByUserID(ctx context.Context, db *gorm.DB, userID string, status *int, limit int, cursor *string) ([]entities.Meme, *string, error)
	// DeleteWithDependents removes a meme together with its captions, reactions, votes and view history.
	// Call it inside a transaction: either everything goes or nothing does.
	DeleteWithDependents(ctx context.Context, db *gorm.DB, memeID string) error
}

type memeRepository struct {
	*GormRepository[entities.Meme]
}

// Pages follow the sort order, so the cursor (a meme id) is compared on the same keys the list is sorted by.
// id breaks ties between memes created at the same instant.
const (
	cursorBeforeClause = "(created_at, id) < (SELECT created_at, id FROM memes WHERE id = ?)"
	orderByNewest      = "created_at DESC, id DESC"
	orderByMostViewed  = "view_count DESC, created_at DESC, id DESC"
)

func NewMemeRepository() MemeRepository {
	return &memeRepository{GormRepository: NewGormRepository[entities.Meme]()}
}

func (r *memeRepository) FindApprovedFeed(ctx context.Context, db *gorm.DB, userId string, languageID *int, q *string, limit int, cursor *string, trending *bool) ([]entities.Meme, *string, error) {
	query := db.WithContext(ctx).Model(&entities.Meme{}).Where(constants.STATUS_WHERE_CLAUSE, 2)
	if languageID != nil {
		query = query.Where("language_id = ?", *languageID)
	}
	if q != nil && *q != "" {
		query = query.Where("template_id ILIKE ?", "%"+*q+"%")
	}
	if cursor != nil && *cursor != "" {
		if trending != nil && *trending {
			query = query.Where("(view_count, created_at, id) < (SELECT view_count, created_at, id FROM memes WHERE id = ?)", *cursor)
		} else {
			query = query.Where(cursorBeforeClause, *cursor)
		}
	}

	orderBy := orderByNewest
	if trending != nil && *trending {
		orderBy = orderByMostViewed
	}

	var memes []entities.Meme
	err := query.Order(orderBy).Limit(limit).Find(&memes).Error
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(memes) > 0 {
		last := memes[len(memes)-1].ID
		next = &last
	}
	return memes, next, nil
}

func (r *memeRepository) DeleteWithDependents(ctx context.Context, db *gorm.DB, memeID string) error {
	tx := db.WithContext(ctx)
	// Rows that point at the meme go first, so no foreign key can block the delete.
	for _, model := range []interface{}{&entities.MemeText{}, &entities.MemeReaction{}, &entities.MemeViewHistory{}} {
		if err := tx.Where("meme_id = ?", memeID).Delete(model).Error; err != nil {
			return err
		}
	}
	// Votes refer to the meme by the same id, stored in asset_id.
	if err := tx.Where("asset_id = ?", memeID).Delete(&entities.MemeVoteHistory{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", memeID).Delete(&entities.Meme{}).Error
}

func (r *memeRepository) IncrementViewCount(ctx context.Context, db *gorm.DB, memeID string) error {
	return db.WithContext(ctx).Model(&entities.Meme{}).Where("id = ?", memeID).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (r *memeRepository) FindPending(ctx context.Context, db *gorm.DB, limit int, cursor *string) ([]entities.Meme, *string, error) {
	query := db.WithContext(ctx).Model(&entities.Meme{}).Where(constants.STATUS_WHERE_CLAUSE, 1)
	if cursor != nil && *cursor != "" {
		query = query.Where(cursorBeforeClause, *cursor)
	}
	var memes []entities.Meme
	if err := query.Order(orderByNewest).Limit(limit).Find(&memes).Error; err != nil {
		return nil, nil, err
	}
	var next *string
	if len(memes) > 0 {
		last := memes[len(memes)-1].ID
		next = &last
	}
	return memes, next, nil
}

func (r *memeRepository) FindByUserID(ctx context.Context, db *gorm.DB, userID string, status *int, limit int, cursor *string) ([]entities.Meme, *string, error) {
	query := db.WithContext(ctx).Model(&entities.Meme{}).Where("user_id = ?", userID)
	if status != nil {
		query = query.Where(constants.STATUS_WHERE_CLAUSE, *status)
	}
	if cursor != nil && *cursor != "" {
		query = query.Where(cursorBeforeClause, *cursor)
	}
	var memes []entities.Meme
	if err := query.Order(orderByNewest).Limit(limit).Find(&memes).Error; err != nil {
		return nil, nil, err
	}
	var next *string
	if len(memes) > 0 {
		last := memes[len(memes)-1].ID
		next = &last
	}
	return memes, next, nil
}
