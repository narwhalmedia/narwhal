package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pkgerrors "github.com/narwhalmedia/narwhal/pkg/errors"
	"gorm.io/gorm"
)

// Create creates a new entity in the database.
func Create[T any](ctx context.Context, db *gorm.DB, entity *T) error {
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		if pkgerrors.IsDuplicateError(err) {
			return pkgerrors.Conflict("entity already exists")
		}
		return err
	}
	return nil
}

// FindByID finds an entity by its ID. It preloads specified associations.
func FindByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID, preloads ...string) (*T, error) {
	var entity T
	query := db.WithContext(ctx)
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	if err := query.First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("entity not found")
		}
		return nil, err
	}
	return &entity, nil
}

// FindOneBy finds a single entity by a query condition. It preloads specified associations.
func FindOneBy[T any](ctx context.Context, db *gorm.DB, query string, args ...interface{}) (*T, error) {
	var entity T
	if err := db.WithContext(ctx).Where(query, args...).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("entity not found")
		}
		return nil, err
	}
	return &entity, nil
}

// Update updates an entity in the database.
func Update[T any](ctx context.Context, db *gorm.DB, entity *T) error {
	return db.WithContext(ctx).Save(entity).Error
}

// Delete removes an entity from the database by its ID.
func Delete[T any](ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	var entity T
	result := db.WithContext(ctx).Delete(&entity, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("entity not found for deletion")
	}
	return nil
}

// List retrieves a list of entities with optional filters, pagination, and preloads.
func List[T any](ctx context.Context, db *gorm.DB, limit, offset int, preloads ...string) ([]*T, error) {
	var entities []*T
	query := db.WithContext(ctx)
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	if err := query.Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// Count returns the total number of entities.
func Count[T any](ctx context.Context, db *gorm.DB) (int64, error) {
	var count int64
	var entity T
	if err := db.WithContext(ctx).Model(&entity).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
