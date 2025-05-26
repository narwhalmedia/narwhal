package gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/domain/specification"
	"gorm.io/gorm"
)

// MovieRepository implements media.MovieRepository
type MovieRepository struct {
	db *gorm.DB
}

// NewMovieRepository creates a new GORM movie repository
func NewMovieRepository(db *gorm.DB) media.MovieRepository {
	return &MovieRepository{db: db}
}

// Save persists a movie to the database
func (r *MovieRepository) Save(ctx context.Context, movie *media.Movie) error {
	model := &MovieModel{}
	model.FromDomain(movie)

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}

	// Update the domain model with the latest version
	movie.BaseAggregate.Version = model.Version
	movie.BaseAggregate.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByID retrieves a movie by its ID
func (r *MovieRepository) FindByID(ctx context.Context, id uuid.UUID) (*media.Movie, error) {
	var model MovieModel
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

// FindByTitle retrieves a movie by its title
func (r *MovieRepository) FindByTitle(ctx context.Context, title string) (*media.Movie, error) {
	var model MovieModel
	result := r.db.WithContext(ctx).First(&model, "title = ?", title)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

// Delete removes a movie from the database
func (r *MovieRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&MovieModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindAll returns all movies
func (r *MovieRepository) FindAll(ctx context.Context) ([]*media.Movie, error) {
	var models []MovieModel
	result := r.db.WithContext(ctx).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	movies := make([]*media.Movie, len(models))
	for i, model := range models {
		movies[i] = model.ToDomain()
	}
	return movies, nil
}

// FindByStatus finds movies by status
func (r *MovieRepository) FindByStatus(ctx context.Context, status media.Status) ([]*media.Movie, error) {
	var models []MovieModel
	result := r.db.WithContext(ctx).Where("status = ?", status).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	movies := make([]*media.Movie, len(models))
	for i, model := range models {
		movies[i] = model.ToDomain()
	}
	return movies, nil
}

// FindByGenre finds movies by genre
func (r *MovieRepository) FindByGenre(ctx context.Context, genre string) ([]*media.Movie, error) {
	var models []MovieModel
	result := r.db.WithContext(ctx).Where("? = ANY(genres)", genre).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	movies := make([]*media.Movie, len(models))
	for i, model := range models {
		movies[i] = model.ToDomain()
	}
	return movies, nil
}

// FindByDirector finds movies by director
func (r *MovieRepository) FindByDirector(ctx context.Context, director string) ([]*media.Movie, error) {
	var models []MovieModel
	result := r.db.WithContext(ctx).Where("director = ?", director).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	movies := make([]*media.Movie, len(models))
	for i, model := range models {
		movies[i] = model.ToDomain()
	}
	return movies, nil
}

// FindBySpecification finds movies matching the specification
func (r *MovieRepository) FindBySpecification(ctx context.Context, spec specification.Specification) ([]*media.Movie, error) {
	sql, params := spec.ToSQL()
	
	var models []MovieModel
	result := r.db.WithContext(ctx).Where(sql, params...).Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("find by specification: %w", result.Error)
	}
	
	movies := make([]*media.Movie, len(models))
	for i, model := range models {
		movies[i] = model.ToDomain()
	}
	
	return movies, nil
}

// CountBySpecification counts movies matching the specification
func (r *MovieRepository) CountBySpecification(ctx context.Context, spec specification.Specification) (int64, error) {
	sql, params := spec.ToSQL()
	
	var count int64
	result := r.db.WithContext(ctx).Model(&MovieModel{}).Where(sql, params...).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("count by specification: %w", result.Error)
	}
	
	return count, nil
}