package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LibraryRepositoryTestSuite struct {
	suite.Suite
	container *testutil.PostgresContainer
	repo      repository.Repository
	ctx       context.Context
}

func (suite *LibraryRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.container = testutil.SetupPostgresContainer(suite.T())

	// Run migrations
	err := suite.container.MigrateModels(
		&repository.Library{},
		&repository.MediaItem{},
		&repository.Episode{},
		&repository.MetadataProvider{},
		&repository.ScanHistory{},
	)
	require.NoError(suite.T(), err)
}

func (suite *LibraryRepositoryTestSuite) SetupTest() {
	// Create repository
	var err error
	suite.repo, err = repository.NewGormRepository(suite.container.DB)
	suite.Require().NoError(err)

	// Clean tables before each test
	suite.container.TruncateTables("episodes", "media_items", "scan_histories", "metadata_providers", "libraries")
}

func (suite *LibraryRepositoryTestSuite) TestCreateLibrary() {
	// Arrange
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Test Library",
		Path:         "/test/path",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	err := suite.repo.CreateLibrary(suite.ctx, library)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify library was created
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), library.Name, retrieved.Name)
	assert.Equal(suite.T(), library.Path, retrieved.Path)
}

func (suite *LibraryRepositoryTestSuite) TestGetLibraryByPath() {
	// Arrange
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Test Library",
		Path:         "/unique/path",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	// Act
	retrieved, err := suite.repo.GetLibraryByPath(suite.ctx, "/unique/path")

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved)
	assert.Equal(suite.T(), library.Path, retrieved.Path)
}

func (suite *LibraryRepositoryTestSuite) TestListLibraries() {
	// Arrange
	for i := 0; i < 3; i++ {
		library := &domain.Library{
			ID:           uuid.New(),
			Name:         fmt.Sprintf("Library %d", i),
			Path:         fmt.Sprintf("/path/%d", i),
			Type:         "movie",
			Enabled:      i%2 == 0, // Alternate enabled/disabled
			ScanInterval: 3600,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		suite.repo.CreateLibrary(suite.ctx, library)
	}

	// Test list all
	all, err := suite.repo.ListLibraries(suite.ctx, nil)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), all, 3)

	// Test list enabled only
	enabled := true
	enabledLibs, err := suite.repo.ListLibraries(suite.ctx, &enabled)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), enabledLibs, 2)
}

func (suite *LibraryRepositoryTestSuite) TestUpdateLibrary() {
	// Arrange
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Original Name",
		Path:         "/original/path",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	// Act
	library.Name = "Updated Name"
	library.Enabled = false
	err := suite.repo.UpdateLibrary(suite.ctx, library)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify update
	retrieved, _ := suite.repo.GetLibrary(suite.ctx, library.ID)
	assert.Equal(suite.T(), "Updated Name", retrieved.Name)
	assert.False(suite.T(), retrieved.Enabled)
}

func (suite *LibraryRepositoryTestSuite) TestDeleteLibrary() {
	// Arrange
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "To Delete",
		Path:         "/delete/me",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	// Act
	err := suite.repo.DeleteLibrary(suite.ctx, library.ID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify deletion
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
}

func (suite *LibraryRepositoryTestSuite) TestMediaOperations() {
	// Create library first
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Media Library",
		Path:         "/media",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	// Create media
	media := &models.Media{
		ID:          uuid.New(),
		LibraryID:   library.ID,
		Title:       "Test Movie",
		Type:        models.MediaTypeMovie,
		Path:        "/media/test.mp4",
		Size:        1024 * 1024 * 100,
		Duration:    7200,
		Resolution:  "1920x1080",
		Codec:       "h264",
		Bitrate:     5000,
		Added:       time.Now(),
		Modified:    time.Now(),
		LastScanned: time.Now(),
		Status:      "available",
		FilePath:    "/media/test.mp4",
		FileSize:    1024 * 1024 * 100,
	}

	// Create media
	err := suite.repo.CreateMedia(suite.ctx, media)
	assert.NoError(suite.T(), err)

	// Get media
	retrieved, err := suite.repo.GetMedia(suite.ctx, media.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), media.Title, retrieved.Title)

	// Get media by path
	byPath, err := suite.repo.GetMediaByPath(suite.ctx, "/media/test.mp4")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), media.ID, byPath.ID)

	// Update media
	media.Title = "Updated Movie"
	err = suite.repo.UpdateMedia(suite.ctx, media)
	assert.NoError(suite.T(), err)

	// List media by library
	mediaList, err := suite.repo.ListMediaByLibrary(suite.ctx, library.ID, nil, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), mediaList, 1)

	// Search media
	searchResults, err := suite.repo.SearchMedia(suite.ctx, "Updated", nil, nil, &library.ID, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), searchResults, 1)

	// Delete media
	err = suite.repo.DeleteMedia(suite.ctx, media.ID)
	assert.NoError(suite.T(), err)
}

func (suite *LibraryRepositoryTestSuite) TestEpisodeOperations() {
	// Create library and series
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Series Library",
		Path:         "/series",
		Type:         "tv_show",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	series := &models.Media{
		ID:          uuid.New(),
		LibraryID:   library.ID,
		Title:       "Test Series",
		Type:        models.MediaTypeSeries,
		Path:        "/series/test",
		Added:       time.Now(),
		Modified:    time.Now(),
		LastScanned: time.Now(),
		Status:      "available",
	}
	suite.repo.CreateMedia(suite.ctx, series)

	// Create episodes
	episode1 := testutil.CreateTestEpisode(series.ID, 1, 1, "Episode 1")
	episode2 := testutil.CreateTestEpisode(series.ID, 1, 2, "Episode 2")

	err := suite.repo.CreateEpisode(suite.ctx, episode1)
	assert.NoError(suite.T(), err)
	err = suite.repo.CreateEpisode(suite.ctx, episode2)
	assert.NoError(suite.T(), err)

	// Get episode
	retrieved, err := suite.repo.GetEpisode(suite.ctx, episode1.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), episode1.Title, retrieved.Title)

	// List episodes by media
	episodes, err := suite.repo.ListEpisodesByMedia(suite.ctx, series.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), episodes, 2)

	// Update episode
	episode1.Title = "Updated Episode 1"
	err = suite.repo.UpdateEpisode(suite.ctx, episode1)
	assert.NoError(suite.T(), err)

	// Delete episode
	err = suite.repo.DeleteEpisode(suite.ctx, episode1.ID)
	assert.NoError(suite.T(), err)

	// Verify only one episode remains
	episodes, _ = suite.repo.ListEpisodesByMedia(suite.ctx, series.ID)
	assert.Len(suite.T(), episodes, 1)
}

func (suite *LibraryRepositoryTestSuite) TestScanHistoryOperations() {
	// Create library
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Scan Library",
		Path:         "/scan",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.repo.CreateLibrary(suite.ctx, library)

	// Create scan history
	scan := &domain.ScanResult{
		ID:           uuid.New(),
		LibraryID:    library.ID,
		StartedAt:    time.Now(),
		FilesScanned: 100,
		FilesAdded:   10,
		FilesUpdated: 5,
		FilesDeleted: 2,
	}

	err := suite.repo.CreateScanHistory(suite.ctx, scan)
	assert.NoError(suite.T(), err)

	// Update scan history
	now := time.Now()
	scan.CompletedAt = &now
	err = suite.repo.UpdateScanHistory(suite.ctx, scan)
	assert.NoError(suite.T(), err)

	// Get latest scan
	latest, err := suite.repo.GetLatestScan(suite.ctx, library.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), scan.ID, latest.ID)
	assert.NotNil(suite.T(), latest.CompletedAt)

	// List scan history - method not implemented yet
	// history, err := suite.repo.ListScanHistory(suite.ctx, library.ID, 10)
	// assert.NoError(suite.T(), err)
	// assert.Len(suite.T(), history, 1)
}

// TestMetadataProviderOperations - commenting out until methods are implemented
// func (suite *LibraryRepositoryTestSuite) TestMetadataProviderOperations() {
// 	// Create provider
// 	provider := &domain.MetadataProvider{
// 		ID:           uuid.New(),
// 		Name:         "TMDB",
// 		ProviderType: "tmdb",
// 		APIKey:       "test-api-key",
// 		Enabled:      true,
// 		Priority:     1,
// 		CreatedAt:    time.Now(),
// 		UpdatedAt:    time.Now(),
// 	}

// 	err := suite.repo.CreateMetadataProvider(suite.ctx, provider)
// 	assert.NoError(suite.T(), err)

// 	// Get provider
// 	retrieved, err := suite.repo.GetMetadataProvider(suite.ctx, provider.ID)
// 	assert.NoError(suite.T(), err)
// 	assert.Equal(suite.T(), provider.Name, retrieved.Name)

// 	// List providers
// 	providers, err := suite.repo.ListMetadataProviders(suite.ctx)
// 	assert.NoError(suite.T(), err)
// 	assert.Len(suite.T(), providers, 1)

// 	// Update provider
// 	provider.Enabled = false
// 	err = suite.repo.UpdateMetadataProvider(suite.ctx, provider)
// 	assert.NoError(suite.T(), err)

// 	// Delete provider
// 	err = suite.repo.DeleteMetadataProvider(suite.ctx, provider.ID)
// 	assert.NoError(suite.T(), err)
// }

func (suite *LibraryRepositoryTestSuite) TestTransaction() {
	// Start transaction
	tx, err := suite.repo.BeginTx(suite.ctx)
	assert.NoError(suite.T(), err)

	// Create library in transaction
	library := &domain.Library{
		ID:           uuid.New(),
		Name:         "Transaction Test",
		Path:         "/tx/test",
		Type:         "movie",
		Enabled:      true,
		ScanInterval: 3600,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = tx.CreateLibrary(suite.ctx, library)
	assert.NoError(suite.T(), err)

	// Rollback transaction
	err = tx.Rollback()
	assert.NoError(suite.T(), err)

	// Verify library was not created
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
}

func TestLibraryRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(LibraryRepositoryTestSuite))
}
