package gorm

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("not found")
)

// SeriesRepository implements media.SeriesRepository
type SeriesRepository struct {
	db *gorm.DB
}

// NewSeriesRepository creates a new GORM series repository
func NewSeriesRepository(db *gorm.DB) media.SeriesRepository {
	return &SeriesRepository{db: db}
}

// Save persists a series to the database
func (r *SeriesRepository) Save(ctx context.Context, series *media.Series) error {
	model := &SeriesModel{}
	model.FromDomain(series)

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}

	// Update the domain model with the latest version
	series.BaseAggregate.Version = model.Version
	series.BaseAggregate.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByID retrieves a series by its ID
func (r *SeriesRepository) FindByID(ctx context.Context, id uuid.UUID) (*media.Series, error) {
	var model SeriesModel
	result := r.db.WithContext(ctx).Preload("Episodes").First(&model, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

// FindByTitle retrieves a series by its title
func (r *SeriesRepository) FindByTitle(ctx context.Context, title string) (*media.Series, error) {
	var model SeriesModel
	result := r.db.WithContext(ctx).Preload("Episodes").First(&model, "title = ?", title)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

// Delete removes a series from the database
func (r *SeriesRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&SeriesModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindAll returns all series
func (r *SeriesRepository) FindAll(ctx context.Context) ([]*media.Series, error) {
	var models []SeriesModel
	result := r.db.WithContext(ctx).Preload("Episodes").Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	series := make([]*media.Series, len(models))
	for i, model := range models {
		series[i] = model.ToDomain()
	}
	return series, nil
}

// FindByStatus finds series by status
func (r *SeriesRepository) FindByStatus(ctx context.Context, status media.Status) ([]*media.Series, error) {
	var models []SeriesModel
	result := r.db.WithContext(ctx).Preload("Episodes").Where("status = ?", string(status)).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	series := make([]*media.Series, len(models))
	for i, model := range models {
		series[i] = model.ToDomain()
	}
	return series, nil
} 