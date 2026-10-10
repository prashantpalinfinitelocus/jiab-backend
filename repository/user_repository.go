package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type UserRepository interface {
	GenericRepository[entities.User]
	FindByEmail(ctx context.Context, tx *gorm.DB, email string) (*entities.User, error)
	// FindPageByConditions returns up to `limit` users matching `conditions`
	// (simple equality filters) with id > afterID, ordered by id. Used for
	// keyset-paginated batch processing over large user populations — cheap
	// at any offset, unlike OFFSET/LIMIT which gets slower as the offset grows.
	FindPageByConditions(ctx context.Context, tx *gorm.DB, conditions map[string]interface{}, afterID string, limit int) ([]entities.User, error)
	// FindByIDs bulk-fetches users by id, for re-verifying a whole batch's
	// current state in one query instead of one query per user.
	FindByIDs(ctx context.Context, tx *gorm.DB, ids []string) ([]entities.User, error)
	FindUsersByIDs(ctx context.Context, tx *gorm.DB, ids []string) (map[string]*entities.User, error)
}

type userRepository struct {
	*GormRepository[entities.User]
}

func NewUserRepository() UserRepository {
	return &userRepository{
		GormRepository: NewGormRepository[entities.User](),
	}
}

func (r *userRepository) FindByEmail(ctx context.Context, tx *gorm.DB, email string) (*entities.User, error) {
	var user entities.User
	if err := tx.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindUsersByIDs(ctx context.Context, tx *gorm.DB, ids []string) (map[string]*entities.User, error) {
	if len(ids) == 0 {
		return make(map[string]*entities.User), nil
	}

	var users []entities.User
	if err := tx.WithContext(ctx).Where("id IN (?) AND is_deleted = false", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	userMap := make(map[string]*entities.User)
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	return userMap, nil
}

func (r *userRepository) FindPageByConditions(ctx context.Context, tx *gorm.DB, conditions map[string]interface{}, afterID string, limit int) ([]entities.User, error) {
	var users []entities.User
	query := tx.WithContext(ctx).Model(&entities.User{})
	for field, value := range conditions {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}
	if afterID != "" {
		query = query.Where("id > ?", afterID)
	}
	if err := query.Order("id ASC").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) FindByIDs(ctx context.Context, tx *gorm.DB, ids []string) ([]entities.User, error) {
	var users []entities.User
	if len(ids) == 0 {
		return users, nil
	}
	if err := tx.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
