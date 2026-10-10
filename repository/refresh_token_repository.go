package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type RefreshTokenRepository interface {
	GenericRepository[entities.UserRefreshToken]
}

type refreshTokenRepository struct {
	*GormRepository[entities.UserRefreshToken]
}

func NewRefreshTokenRepository() RefreshTokenRepository {
	return &refreshTokenRepository{
		GormRepository: NewGormRepository[entities.UserRefreshToken](),
	}
}
