package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type ProfilingQuestionAnswerRepository interface {
	GenericRepository[entities.ProfilingQuestionAnswer]
}

type profilingQuestionAnswerRepository struct {
	GenericRepository[entities.ProfilingQuestionAnswer]
}

func NewProfilingQuestionAnswerRepository() ProfilingQuestionAnswerRepository {
	return &profilingQuestionAnswerRepository{
		GenericRepository: NewGormRepository[entities.ProfilingQuestionAnswer](),
	}
}

