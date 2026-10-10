package repository

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type NotificationTypeRepository interface {
	GenericRepository[entities.NotificationType]
}

type notificationTypeRepository struct {
	*GormRepository[entities.NotificationType]
}

func NewNotificationTypeRepository() NotificationTypeRepository {
	return &notificationTypeRepository{
		GormRepository: NewGormRepository[entities.NotificationType](),
	}
}
