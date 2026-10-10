package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type SSOTokenRepository interface {
	GenericRepository[entities.SSOToken]
	FindByToken(ctx context.Context, tx *gorm.DB, token string) (*entities.SSOToken, error)
	MarkAsUsed(ctx context.Context, tx *gorm.DB, token string) error
}

type ssoTokenRepository struct {
	*GormRepository[entities.SSOToken]
}

func NewSSOTokenRepository() SSOTokenRepository {
	return &ssoTokenRepository{
		GormRepository: NewGormRepository[entities.SSOToken](),
	}
}

func (r *ssoTokenRepository) FindByToken(ctx context.Context, tx *gorm.DB, token string) (*entities.SSOToken, error) {
	var ssoToken entities.SSOToken
	if err := tx.WithContext(ctx).Where("token = ?", token).First(&ssoToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ssoToken, nil
}

func (r *ssoTokenRepository) MarkAsUsed(ctx context.Context, tx *gorm.DB, token string) error {
	now := time.Now()
	return tx.WithContext(ctx).
		Model(&entities.SSOToken{}).
		Where("token = ?", token).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": now,
		}).Error
}
