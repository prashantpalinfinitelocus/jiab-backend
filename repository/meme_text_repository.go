package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type MemeTextRepository interface {
	GenericRepository[entities.MemeText]
	FindByMemeIDs(ctx context.Context, db *gorm.DB, memeIDs []string) ([]entities.MemeText, error)
}

type memeTextRepository struct {
	*GormRepository[entities.MemeText]
}

func NewMemeTextRepository() MemeTextRepository {
	return &memeTextRepository{GormRepository: NewGormRepository[entities.MemeText]()}
}

func (r *memeTextRepository) FindByMemeIDs(ctx context.Context, db *gorm.DB, memeIDs []string) ([]entities.MemeText, error) {
	var texts []entities.MemeText
	if len(memeIDs) == 0 {
		return texts, nil
	}
	err := db.WithContext(ctx).Where("meme_id IN ?", memeIDs).Find(&texts).Error
	if err != nil {
		return nil, err
	}
	return texts, nil
}
