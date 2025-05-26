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

func TestSeriesRepository(t *testing.T) {
	db := setupTestDB(t)

	// Use the GORM series repository from the gorm package
	// This test is now obsolete as series repository is implemented in the gorm package
	t.Skip("Series repository is now implemented in the gorm package")
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		// Create test series
		series := &media.Series{
			ID:          uuid.New(),
			Title:       "Test Series",
			Description: "Test Description",
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, series)
		require.NoError(t, err)

		// Get series by ID
		retrieved, err := repo.Get(ctx, series.ID)
		require.NoError(t, err)
		assert.Equal(t, series.ID, retrieved.ID)
		assert.Equal(t, series.Title, retrieved.Title)
		assert.Equal(t, series.Description, retrieved.Description)
		assert.Equal(t, series.Status, retrieved.Status)
	})

	t.Run("GetByTitle", func(t *testing.T) {
		// Create test series
		series := &media.Series{
			ID:          uuid.New(),
			Title:       "Unique Series",
			Description: "Test Description",
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, series)
		require.NoError(t, err)

		// Get series by title
		retrieved, err := repo.GetByTitle(ctx, series.Title)
		require.NoError(t, err)
		assert.Equal(t, series.ID, retrieved.ID)
		assert.Equal(t, series.Title, retrieved.Title)
	})

	t.Run("Update", func(t *testing.T) {
		// Create test series
		series := &media.Series{
			ID:          uuid.New(),
			Title:       "Update Test Series",
			Description: "Original Description",
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, series)
		require.NoError(t, err)

		// Update series
		series.Description = "Updated Description"
		series.Status = media.StatusCompleted
		err = repo.Update(ctx, series)
		require.NoError(t, err)

		// Get updated series
		retrieved, err := repo.Get(ctx, series.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", retrieved.Description)
		assert.Equal(t, media.StatusCompleted, retrieved.Status)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create test series
		series := &media.Series{
			ID:          uuid.New(),
			Title:       "Delete Test Series",
			Description: "Test Description",
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, series)
		require.NoError(t, err)

		// Delete series
		err = repo.Delete(ctx, series.ID)
		require.NoError(t, err)

		// Try to get deleted series
		_, err = repo.Get(ctx, series.ID)
		assert.Error(t, err)
	})

	t.Run("Episodes", func(t *testing.T) {
		// Create test series
		series := &media.Series{
			ID:          uuid.New(),
			Title:       "Episodes Test Series",
			Description: "Test Description",
			Status:      media.StatusActive,
		}

		err := repo.Create(ctx, series)
		require.NoError(t, err)

		// Create test episodes
		episode1 := &media.Episode{
			ID:           uuid.New(),
			SeriesID:     series.ID,
			Title:        "Episode 1",
			Description:  "First Episode",
			SeasonNumber: 1,
			EpisodeNumber: 1,
			AirDate:      time.Now(),
			Status:       media.StatusActive,
		}

		episode2 := &media.Episode{
			ID:           uuid.New(),
			SeriesID:     series.ID,
			Title:        "Episode 2",
			Description:  "Second Episode",
			SeasonNumber: 1,
			EpisodeNumber: 2,
			AirDate:      time.Now(),
			Status:       media.StatusActive,
		}

		// Add episodes
		err = repo.AddEpisode(ctx, episode1)
		require.NoError(t, err)
		err = repo.AddEpisode(ctx, episode2)
		require.NoError(t, err)

		// Get episodes
		episodes, err := repo.GetEpisodes(ctx, series.ID)
		require.NoError(t, err)
		assert.Len(t, episodes, 2)

		// Update episode
		episode1.Status = media.StatusCompleted
		err = repo.UpdateEpisode(ctx, episode1)
		require.NoError(t, err)

		// Get updated episode
		retrieved, err := repo.GetEpisode(ctx, episode1.ID)
		require.NoError(t, err)
		assert.Equal(t, media.StatusCompleted, retrieved.Status)

		// Remove episode
		err = repo.RemoveEpisode(ctx, episode2.ID)
		require.NoError(t, err)

		// Get remaining episodes
		episodes, err = repo.GetEpisodes(ctx, series.ID)
		require.NoError(t, err)
		assert.Len(t, episodes, 1)
	})
}

func setupTestDB(t *testing.T) *sql.DB {
	// TODO: Implement test database setup
	// This should create a test database, run migrations, and return a connection
	return nil
} 