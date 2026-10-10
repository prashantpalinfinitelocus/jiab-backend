package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type ReferenceDataRepository interface {
}

type referenceDataRepository struct {
	*GormRepository[entities.Language]
}
