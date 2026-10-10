package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type UserVideoRepository interface {
	GenericRepository[entities.UserVideo]
}

type userVideoRepository struct {
	*GormRepository[entities.UserVideo]
}

func NewUserVideoRepository() UserVideoRepository {
	return &userVideoRepository{
		GormRepository: NewGormRepository[entities.UserVideo](),
	}
}
