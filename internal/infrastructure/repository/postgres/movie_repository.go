package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"gorm.io/gorm"
)

// MovieRepository implements media.MovieRepository
type MovieRepository struct {
	db *gorm.DB
}

// NewMovieRepository creates a new PostgreSQL movie repository
func NewMovieRepository(db *gorm.DB) media.MovieRepository {
	return &MovieRepository{db: db}
}

// Save persists a movie to the database
func (r *MovieRepository) Save(ctx context.Context, movie *media.Movie) error {
	genresJSON, err := json.Marshal(movie.Genres)
	if err != nil {
		return fmt.Errorf("marshaling genres: %w", err)
	}

	movieModel := Movie{
		ID:            movie.ID(),
		Title:         movie.Title,
		Description:   movie.Description,
		ReleaseDate:   movie.ReleaseDate,
		Genres:        string(genresJSON),
		Director:      movie.Director,
		Status:        string(movie.Status),
		FilePath:      movie.FilePath,
		ThumbnailPath: movie.ThumbnailPath,
		Duration:      int(movie.Duration),
		Version:       movie.Version() + 1,
		CreatedAt:     movie.CreatedAt(),
		UpdatedAt:     time.Now(),
	}

	// Use optimistic locking with version check
	result := r.db.WithContext(ctx).Model(&Movie{}).
		Where("id = ? AND version = ?", movie.ID(), movie.Version()).
		Updates(&movieModel)

	if result.Error != nil {
		return fmt.Errorf("saving movie: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// Try insert if update failed
		movieModel.Version = 1
		if err := r.db.WithContext(ctx).Create(&movieModel).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("concurrent modification detected")
			}
			return fmt.Errorf("creating movie: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a movie by its ID
func (r *MovieRepository) FindByID(ctx context.Context, id uuid.UUID) (media.Aggregate, error) {
	var movieModel Movie
	
	if err := r.db.WithContext(ctx).First(&movieModel, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("movie not found")
		}
		return nil, fmt.Errorf("querying movie: %w", err)
	}

	var genres []string
	if movieModel.Genres != "" {
		if err := json.Unmarshal([]byte(movieModel.Genres), &genres); err != nil {
			return nil, fmt.Errorf("unmarshaling genres: %w", err)
		}
	}

	movie := media.NewMovie(movieModel.Title, movieModel.Description, movieModel.ReleaseDate, genres, movieModel.Director)
	movie.BaseAggregate.ID = movieModel.ID
	movie.BaseAggregate.Version = movieModel.Version
	movie.BaseAggregate.CreatedAt = movieModel.CreatedAt
	movie.BaseAggregate.UpdatedAt = movieModel.UpdatedAt
	movie.Status = media.Status(movieModel.Status)
	movie.FilePath = movieModel.FilePath
	movie.ThumbnailPath = movieModel.ThumbnailPath
	movie.Duration = time.Duration(movieModel.Duration)

	return movie, nil
}

// FindByTitle retrieves a movie by its title
func (r *MovieRepository) FindByTitle(ctx context.Context, title string) (*media.Movie, error) {
	var movieModel Movie
	
	if err := r.db.WithContext(ctx).First(&movieModel, "title = ?", title).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying movie: %w", err)
	}

	aggregate, err := r.FindByID(ctx, movieModel.ID)
	if err != nil {
		return nil, err
	}

	movie, ok := aggregate.(*media.Movie)
	if !ok {
		return nil, fmt.Errorf("invalid aggregate type")
	}

	return movie, nil
}

// Delete removes a movie from the database
func (r *MovieRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&Movie{}, "id = ?", id)
	
	if result.Error != nil {
		return fmt.Errorf("deleting movie: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

// Create creates a new movie
func (r *MovieRepository) Create(ctx context.Context, movie *media.Movie) error {
	movieModel := Movie{
		ID:          movie.ID,
		Title:       movie.Title,
		Description: movie.Description,
		ReleaseDate: movie.ReleaseDate,
		Genres:      "[]", // Default empty array
		Status:      string(movie.Status),
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if movie.Runtime > 0 {
		movieModel.Duration = int(movie.Runtime)
	}

	if err := r.db.WithContext(ctx).Create(&movieModel).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return media.ErrMovieAlreadyExists
		}
		return err
	}

	return nil
}

// Get retrieves a movie by ID
func (r *MovieRepository) Get(ctx context.Context, id uuid.UUID) (*media.Movie, error) {
	var movieModel Movie
	
	if err := r.db.WithContext(ctx).First(&movieModel, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, media.ErrMovieNotFound
		}
		return nil, err
	}

	movie := &media.Movie{
		ID:          movieModel.ID,
		Title:       movieModel.Title,
		Description: movieModel.Description,
		ReleaseDate: movieModel.ReleaseDate,
		Runtime:     time.Duration(movieModel.Duration),
		Status:      media.Status(movieModel.Status),
		FilePath:    movieModel.FilePath,
	}

	return movie, nil
}

// GetByTitle retrieves a movie by title
func (r *MovieRepository) GetByTitle(ctx context.Context, title string) (*media.Movie, error) {
	var movieModel Movie
	
	if err := r.db.WithContext(ctx).First(&movieModel, "title = ?", title).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, media.ErrMovieNotFound
		}
		return nil, err
	}

	movie := &media.Movie{
		ID:          movieModel.ID,
		Title:       movieModel.Title,
		Description: movieModel.Description,
		ReleaseDate: movieModel.ReleaseDate,
		Runtime:     time.Duration(movieModel.Duration),
		Status:      media.Status(movieModel.Status),
		FilePath:    movieModel.FilePath,
	}

	return movie, nil
}

// Update updates a movie
func (r *MovieRepository) Update(ctx context.Context, movie *media.Movie) error {
	updates := map[string]interface{}{
		"title":        movie.Title,
		"description":  movie.Description,
		"release_date": movie.ReleaseDate,
		"duration":     int(movie.Runtime),
		"status":       string(movie.Status),
		"updated_at":   time.Now(),
	}

	result := r.db.WithContext(ctx).Model(&Movie{}).
		Where("id = ?", movie.ID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return media.ErrMovieNotFound
	}

	return nil
}

// UpdateFile updates a movie's file path
func (r *MovieRepository) UpdateFile(ctx context.Context, id uuid.UUID, filePath string) error {
	result := r.db.WithContext(ctx).Model(&Movie{}).
		Where("id = ?", id).
		Update("file_path", filePath)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return media.ErrMovieNotFound
	}

	return nil
}