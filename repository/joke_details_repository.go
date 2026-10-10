package repository

import "github.com/Infinite-Locus-Product/coca_cola_backend/entities"

type JokeDetailsRepository interface {
	GenericRepository[entities.JokeDetailsRecieved]
}

type jokeDetailsRepository struct {
	*GormRepository[entities.JokeDetailsRecieved]
}

func NewJokeDetailsRepository() JokeDetailsRepository {
	return &jokeDetailsRepository{
		GormRepository: NewGormRepository[entities.JokeDetailsRecieved](),
	}
}
