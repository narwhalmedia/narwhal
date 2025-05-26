package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

func TestMovieRepository(t *testing.T) {
	db := setupTestDB(t)

	repo := NewMovieRepository(db)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		// Create test movie
		movie := &media.Movie{
			ID:          uuid.New(),
			Title:       "Test Movie",
			Description: "Test Description",
			ReleaseDate: time.Now(),
			Runtime:     120,
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, movie)
		require.NoError(t, err)

		// Get movie by ID
		retrieved, err := repo.Get(ctx, movie.ID)
		require.NoError(t, err)
		assert.Equal(t, movie.ID, retrieved.ID)
		assert.Equal(t, movie.Title, retrieved.Title)
		assert.Equal(t, movie.Description, retrieved.Description)
		assert.Equal(t, movie.Runtime, retrieved.Runtime)
		assert.Equal(t, movie.Status, retrieved.Status)
		assert.True(t, movie.ReleaseDate.Equal(retrieved.ReleaseDate))
	})

	t.Run("GetByTitle", func(t *testing.T) {
		// Create test movie
		movie := &media.Movie{
			ID:          uuid.New(),
			Title:       "Unique Movie",
			Description: "Test Description",
			ReleaseDate: time.Now(),
			Runtime:     120,
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, movie)
		require.NoError(t, err)

		// Get movie by title
		retrieved, err := repo.GetByTitle(ctx, movie.Title)
		require.NoError(t, err)
		assert.Equal(t, movie.ID, retrieved.ID)
		assert.Equal(t, movie.Title, retrieved.Title)
	})

	t.Run("Update", func(t *testing.T) {
		// Create test movie
		movie := &media.Movie{
			ID:          uuid.New(),
			Title:       "Update Test Movie",
			Description: "Original Description",
			ReleaseDate: time.Now(),
			Runtime:     120,
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, movie)
		require.NoError(t, err)

		// Update movie
		movie.Description = "Updated Description"
		movie.Runtime = 150
		movie.Status = media.StatusCompleted
		err = repo.Update(ctx, movie)
		require.NoError(t, err)

		// Get updated movie
		retrieved, err := repo.Get(ctx, movie.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", retrieved.Description)
		assert.Equal(t, 150, retrieved.Runtime)
		assert.Equal(t, media.StatusCompleted, retrieved.Status)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create test movie
		movie := &media.Movie{
			ID:          uuid.New(),
			Title:       "Delete Test Movie",
			Description: "Test Description",
			ReleaseDate: time.Now(),
			Runtime:     120,
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, movie)
		require.NoError(t, err)

		// Delete movie
		err = repo.Delete(ctx, movie.ID)
		require.NoError(t, err)

		// Try to get deleted movie
		_, err = repo.Get(ctx, movie.ID)
		assert.Error(t, err)
	})

	t.Run("UpdateFile", func(t *testing.T) {
		// Create test movie
		movie := &media.Movie{
			ID:          uuid.New(),
			Title:       "File Test Movie",
			Description: "Test Description",
			ReleaseDate: time.Now(),
			Runtime:     120,
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, movie)
		require.NoError(t, err)

		// Update file path
		filePath := "/path/to/movie.mp4"
		err = repo.UpdateFile(ctx, movie.ID, filePath)
		require.NoError(t, err)

		// Get movie and check file path
		retrieved, err := repo.Get(ctx, movie.ID)
		require.NoError(t, err)
		assert.Equal(t, filePath, retrieved.FilePath)
	})
}

func setupTestDB(t *testing.T) *sql.DB {
	// TODO: Implement test database setup
	// This should create a test database, run migrations, and return a connection
	return nil
} 