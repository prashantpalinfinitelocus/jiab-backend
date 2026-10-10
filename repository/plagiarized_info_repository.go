package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type PlagiarizedInfoRepository interface {
	GenericRepository[entities.PlagiarizedInfo]
	GetByAssetID(ctx context.Context, tx *gorm.DB, assetID string) (*entities.PlagiarizedInfo, error)
	UpdateStatus(ctx context.Context, tx *gorm.DB, id int, status bool) error
}

type plagiarizedInfoRepository struct {
	*GormRepository[entities.PlagiarizedInfo]
}

func NewPlagiarizedInfoRepository() PlagiarizedInfoRepository {
	return &plagiarizedInfoRepository{
		GormRepository: NewGormRepository[entities.PlagiarizedInfo](),
	}
}

func (r *plagiarizedInfoRepository) GetByAssetID(ctx context.Context, tx *gorm.DB, assetID string) (*entities.PlagiarizedInfo, error) {
	var plagiarizedInfo entities.PlagiarizedInfo
	if err := tx.WithContext(ctx).Where("asset_id = ? AND is_deleted = ? AND is_active = ?", assetID, false, true).First(&plagiarizedInfo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("failed to get plagiarized info by asset ID", err)
	}
	return &plagiarizedInfo, nil
}

func (r *plagiarizedInfoRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, id int, status bool) error {
	return tx.WithContext(ctx).Model(&entities.PlagiarizedInfo{}).Where("id = ?", id).Update("is_active", status).Error
}
