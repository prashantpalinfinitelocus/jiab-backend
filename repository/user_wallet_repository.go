package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type UserWalletRepository interface {
	GenericRepository[entities.UserWallet]
}

type userWalletRepository struct {
	*GormRepository[entities.UserWallet]
}

func NewUserWalletRepository() UserWalletRepository {
	return &userWalletRepository{
		GormRepository: NewGormRepository[entities.UserWallet](),
	}
}
