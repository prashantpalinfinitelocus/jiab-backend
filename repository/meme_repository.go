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
}

type memeRepository struct {
	*GormRepository[entities.Meme]
}

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
			query = query.Where("view_count < (SELECT view_count FROM memes WHERE id = ?)", *cursor)
		} else {
			query = query.Where(constants.ID_LESS_THAN_CLAUSE, *cursor)
		}
	}

	orderBy := constants.ORDER_BY_CREATED_AT
	if trending != nil && *trending {
		orderBy = constants.ORDER_BY_VIEW_COUNT
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

func (r *memeRepository) IncrementViewCount(ctx context.Context, db *gorm.DB, memeID string) error {
	return db.WithContext(ctx).Model(&entities.Meme{}).Where("id = ?", memeID).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (r *memeRepository) FindPending(ctx context.Context, db *gorm.DB, limit int, cursor *string) ([]entities.Meme, *string, error) {
	query := db.WithContext(ctx).Model(&entities.Meme{}).Where(constants.STATUS_WHERE_CLAUSE, 1)
	if cursor != nil && *cursor != "" {
		query = query.Where(constants.ID_LESS_THAN_CLAUSE, *cursor)
	}
	var memes []entities.Meme
	if err := query.Order(constants.ORDER_BY_CREATED_AT).Limit(limit).Find(&memes).Error; err != nil {
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
		query = query.Where(constants.ID_LESS_THAN_CLAUSE, *cursor)
	}
	var memes []entities.Meme
	if err := query.Order(constants.ORDER_BY_CREATED_AT).Limit(limit).Find(&memes).Error; err != nil {
		return nil, nil, err
	}
	var next *string
	if len(memes) > 0 {
		last := memes[len(memes)-1].ID
		next = &last
	}
	return memes, next, nil
}
