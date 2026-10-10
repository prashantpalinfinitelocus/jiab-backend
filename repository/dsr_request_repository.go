package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type DSRRequestRepository interface {
	GenericRepository[entities.DSRRequest]
}

type dsrRequestRepository struct {
	*GormRepository[entities.DSRRequest]
}

func NewDSRRequestRepository() DSRRequestRepository {
	return &dsrRequestRepository{GormRepository: NewGormRepository[entities.DSRRequest]()}
}
