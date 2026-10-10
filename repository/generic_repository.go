package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// GenericRepository defines the basic CRUD operations for GORM
type GenericRepository[T any] interface {
	Create(ctx context.Context, db *gorm.DB, entity *T) error
	FindById(ctx context.Context, db *gorm.DB, id interface{}) (*T, error)
	Update(ctx context.Context, db *gorm.DB, id interface{}, entity *T) error
	Delete(ctx context.Context, db *gorm.DB, id interface{}) error
	FindByField(ctx context.Context, db *gorm.DB, fieldName string, fieldValue interface{}) (*T, error)
	FindAll(ctx context.Context, db *gorm.DB) ([]T, error)
	CreateMany(ctx context.Context, db *gorm.DB, entities []T) error
	UpdateFields(ctx context.Context, db *gorm.DB, id interface{}, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, db *gorm.DB, id interface{}) error
	Count(ctx context.Context, db *gorm.DB) (int64, error)
	Exists(ctx context.Context, db *gorm.DB, id interface{}) (bool, error)
	UpdateFieldForMultipleEntries(ctx context.Context, db *gorm.DB, ids []interface{}, field string, value interface{}) error
	FindOneWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) (*T, error)
	FindWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) ([]T, error)
	CountWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) (int64, error)
}

// GormRepository is a concrete implementation of GenericRepository for GORM
type GormRepository[T any] struct {
	TableName string
}

// NewGormRepository creates a new repository for a given GORM model
func NewGormRepository[T any]() *GormRepository[T] {
	var entity T
	tableName := getTableName(entity)
	return &GormRepository[T]{
		TableName: tableName,
	}
}

// getTableName extracts the table name from the entity type
func getTableName[T any](entity T) string {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	// Use GORM's naming strategy to get table name
	// This will convert struct name to snake_case (e.g., User -> users)
	namingStrategy := schema.NamingStrategy{}
	return namingStrategy.TableName(entityType.Name())
}

// Create inserts a new entity into the database
func (r *GormRepository[T]) Create(ctx context.Context, db *gorm.DB, entity *T) error {
	if entity == nil {
		return errs.NewBadRequestError("Entity cannot be nil", nil)
	}

	result := db.WithContext(ctx).Create(entity)
	if result.Error != nil {
		logrus.Errorf("Error creating entity: %v", result.Error)
		return errs.NewInternalServerError("Failed to create entity", result.Error)
	}
	return nil
}

// FindById retrieves an entity by its primary key from the database.
//
// The lookup is written as an explicit primary-key clause rather than
// GORM's `First(&entity, id)` shorthand: that shorthand treats a
// non-numeric string argument as a raw SQL condition instead of a primary
// key value, which silently breaks every entity with a uuid string primary
// key (e.g. entities.DSRRequest). clause.PrimaryColumn resolves to whatever
// column is actually declared as the primary key for T, so this also stays
// correct for entities whose primary key is not literally named "id"
// (e.g. entities.UGCStatusLookup uses status_id).
func (r *GormRepository[T]) FindById(ctx context.Context, db *gorm.DB, id interface{}) (*T, error) {
	var entity T
	result := db.WithContext(ctx).Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).First(&entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logrus.Errorf("Error finding entity by ID: %v", result.Error)
		return nil, errs.NewInternalServerError("Failed to find entity", result.Error)
	}
	return &entity, nil
}

// Update modifies an existing entity in the database
func (r *GormRepository[T]) Update(ctx context.Context, db *gorm.DB, id interface{}, entity *T) error {
	if entity == nil {
		return errs.NewBadRequestError("Entity cannot be nil", nil)
	}
	result := db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(entity)
	if result.Error != nil {
		logrus.Errorf("Error updating entity: %v", result.Error)
		return errs.NewInternalServerError("Failed to update entity", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.NewNotFoundError("No entity found to update", nil)
	}
	return nil
}

// Delete removes an entity from the database by its ID
func (r *GormRepository[T]) Delete(ctx context.Context, db *gorm.DB, id interface{}) error {
	var entity T
	result := db.WithContext(ctx).Delete(&entity, id)
	if result.Error != nil {
		logrus.Errorf("Error deleting entity: %v", result.Error)
		return errs.NewInternalServerError("Failed to delete entity", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.NewNotFoundError(fmt.Sprintf("No entity found with id: %v", id), nil)
	}
	return nil
}

// FindByField retrieves entities by a specific field value
func (r *GormRepository[T]) FindByField(ctx context.Context, db *gorm.DB, fieldName string, fieldValue interface{}) (*T, error) {
	var entity T
	result := db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", fieldName), fieldValue).First(&entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logrus.Errorf("Error finding entities by field %s: %v", fieldName, result.Error)
		return nil, errs.NewInternalServerError(fmt.Sprintf("Failed to find entities by field %s", fieldName), result.Error)
	}
	return &entity, nil
}

// FindAll retrieves all entities from the database
func (r *GormRepository[T]) FindAll(ctx context.Context, db *gorm.DB) ([]T, error) {
	var entities []T
	result := db.WithContext(ctx).Find(&entities)
	if result.Error != nil {
		logrus.Errorf("Error finding all entities: %v", result.Error)
		return nil, errs.NewInternalServerError("Failed to find entities", result.Error)
	}
	return entities, nil
}

// CreateMany inserts multiple entities into the database
func (r *GormRepository[T]) CreateMany(ctx context.Context, db *gorm.DB, entities []T) error {
	if len(entities) == 0 {
		return errs.NewBadRequestError("No entities provided for creation", nil)
	}

	result := db.WithContext(ctx).Create(&entities)
	if result.Error != nil {
		logrus.Errorf("Error creating multiple entities: %v", result.Error)
		return errs.NewInternalServerError("Failed to create multiple entities", result.Error)
	}
	return nil
}

// UpdateFields updates specific fields of an entity
func (r *GormRepository[T]) UpdateFields(ctx context.Context, db *gorm.DB, id interface{}, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return errs.NewBadRequestError("No fields provided for update", nil)
	}

	var entity T
	result := db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		logrus.Errorf("Error updating entity fields: %v", result.Error)
		return errs.NewInternalServerError("Failed to update entity fields", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.NewNotFoundError(fmt.Sprintf("No entity found with id: %v", id), nil)
	}
	return nil
}

// SoftDelete performs a soft delete on an entity (sets deleted_at timestamp)
func (r *GormRepository[T]) SoftDelete(ctx context.Context, db *gorm.DB, id interface{}) error {
	var entity T
	result := db.WithContext(ctx).Delete(&entity, id)
	if result.Error != nil {
		logrus.Errorf("Error soft deleting entity: %v", result.Error)
		return errs.NewInternalServerError("Failed to soft delete entity", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.NewNotFoundError(fmt.Sprintf("No entity found with id: %v", id), nil)
	}
	return nil
}

// Count returns the total number of entities in the table
func (r *GormRepository[T]) Count(ctx context.Context, db *gorm.DB) (int64, error) {
	var count int64
	var entity T
	result := db.WithContext(ctx).Model(&entity).Count(&count)
	if result.Error != nil {
		logrus.Errorf("Error counting entities: %v", result.Error)
		return 0, errs.NewInternalServerError("Failed to count entities", result.Error)
	}
	return count, nil
}

// Exists checks if an entity with the given ID exists
func (r *GormRepository[T]) Exists(ctx context.Context, db *gorm.DB, id interface{}) (bool, error) {
	var count int64
	var entity T
	result := db.WithContext(ctx).Model(&entity).Where("id = ?", id).Count(&count)
	if result.Error != nil {
		logrus.Errorf("Error checking entity existence: %v", result.Error)
		return false, errs.NewInternalServerError("Failed to check entity existence", result.Error)
	}
	return count > 0, nil
}

// Additional utility methods similar to MongoDB implementation

// UpdateFieldForMultipleEntries updates a specific field for multiple entities
func (r *GormRepository[T]) UpdateFieldForMultipleEntries(ctx context.Context, db *gorm.DB, ids []interface{}, field string, value interface{}) error {
	if len(ids) == 0 {
		return errs.NewBadRequestError("No entity IDs provided for update", nil)
	}

	var entity T
	result := db.WithContext(ctx).Model(&entity).Where("id IN ?", ids).Update(field, value)
	if result.Error != nil {
		logrus.Errorf("Error updating field %s for multiple entities: %v", field, result.Error)
		return errs.NewInternalServerError(fmt.Sprintf("Failed to update field %s for multiple entities", field), result.Error)
	}
	return nil
}

// FindWithConditions retrieves entities based on multiple conditions
func (r *GormRepository[T]) FindWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) ([]T, error) {
	if len(conditions) == 0 {
		return nil, errs.NewBadRequestError("No conditions provided for search", nil)
	}

	var entities []T
	query := db.WithContext(ctx)

	for field, value := range conditions {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	result := query.Find(&entities)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("Failed to find entities with conditions", result.Error)
	}
	return entities, nil
}

// FindOneWithConditions retrieves a single entity based on multiple conditions
func (r *GormRepository[T]) FindOneWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) (*T, error) {
	if len(conditions) == 0 {
		return nil, errs.NewBadRequestError("No conditions provided for search", nil)
	}

	var entity T
	query := db.WithContext(ctx)

	for field, value := range conditions {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	result := query.First(&entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errs.NewInternalServerError("Failed to find entity with conditions", result.Error)
	}
	return &entity, nil
}

func (r *GormRepository[T]) CountWithConditions(ctx context.Context, db *gorm.DB, conditions map[string]interface{}) (int64, error) {
	var count int64
	var entity T
	result := db.WithContext(ctx).Model(&entity).Where(conditions).Count(&count)
	if result.Error != nil {
		return 0, errs.NewInternalServerError("Failed to count entities with conditions", result.Error)
	}
	return count, nil
}
