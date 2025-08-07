package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/test/testutil"
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
	suite.Require().NoError(err)
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
	suite.Require().NoError(err)

	// Verify library was created
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	suite.Require().NoError(err)
	suite.Equal(library.Name, retrieved.Name)
	suite.Equal(library.Path, retrieved.Path)
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
	suite.Require().NoError(err)
	suite.NotNil(retrieved)
	suite.Equal(library.Path, retrieved.Path)
}

func (suite *LibraryRepositoryTestSuite) TestListLibraries() {
	// Arrange
	for i := range 3 {
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
	suite.Require().NoError(err)
	suite.Len(all, 3)

	// Test list enabled only
	enabled := true
	enabledLibs, err := suite.repo.ListLibraries(suite.ctx, &enabled)
	suite.Require().NoError(err)
	suite.Len(enabledLibs, 2)
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
	suite.Require().NoError(err)

	// Verify update
	retrieved, _ := suite.repo.GetLibrary(suite.ctx, library.ID)
	suite.Equal("Updated Name", retrieved.Name)
	suite.False(retrieved.Enabled)
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
	suite.Require().NoError(err)

	// Verify deletion
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	suite.Require().Error(err)
	suite.Nil(retrieved)
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
	suite.Require().NoError(err)

	// Get media
	retrieved, err := suite.repo.GetMedia(suite.ctx, media.ID)
	suite.Require().NoError(err)
	suite.Equal(media.Title, retrieved.Title)

	// Get media by path
	byPath, err := suite.repo.GetMediaByPath(suite.ctx, "/media/test.mp4")
	suite.Require().NoError(err)
	suite.Equal(media.ID, byPath.ID)

	// Update media
	media.Title = "Updated Movie"
	err = suite.repo.UpdateMedia(suite.ctx, media)
	suite.Require().NoError(err)

	// List media by library
	mediaList, err := suite.repo.ListMediaByLibrary(suite.ctx, library.ID, nil, 10, 0)
	suite.Require().NoError(err)
	suite.Len(mediaList, 1)

	// Search media
	searchResults, err := suite.repo.SearchMedia(suite.ctx, "Updated", nil, nil, &library.ID, 10, 0)
	suite.Require().NoError(err)
	suite.Len(searchResults, 1)

	// Delete media
	err = suite.repo.DeleteMedia(suite.ctx, media.ID)
	suite.Require().NoError(err)
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
	suite.Require().NoError(err)
	err = suite.repo.CreateEpisode(suite.ctx, episode2)
	suite.Require().NoError(err)

	// Get episode
	retrieved, err := suite.repo.GetEpisode(suite.ctx, episode1.ID)
	suite.Require().NoError(err)
	suite.Equal(episode1.Title, retrieved.Title)

	// List episodes by media
	episodes, err := suite.repo.ListEpisodesByMedia(suite.ctx, series.ID)
	suite.Require().NoError(err)
	suite.Len(episodes, 2)

	// Update episode
	episode1.Title = "Updated Episode 1"
	err = suite.repo.UpdateEpisode(suite.ctx, episode1)
	suite.Require().NoError(err)

	// Delete episode
	err = suite.repo.DeleteEpisode(suite.ctx, episode1.ID)
	suite.Require().NoError(err)

	// Verify only one episode remains
	episodes, _ = suite.repo.ListEpisodesByMedia(suite.ctx, series.ID)
	suite.Len(episodes, 1)
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
	suite.Require().NoError(err)

	// Update scan history
	now := time.Now()
	scan.CompletedAt = &now
	err = suite.repo.UpdateScanHistory(suite.ctx, scan)
	suite.Require().NoError(err)

	// Get latest scan
	latest, err := suite.repo.GetLatestScan(suite.ctx, library.ID)
	suite.Require().NoError(err)
	suite.Equal(scan.ID, latest.ID)
	suite.NotNil(latest.CompletedAt)

	// List scan history - method not implemented yet
	// history, err := suite.repo.ListScanHistory(suite.ctx, library.ID, 10)
	// require.NoError(suite.T(), err)
	// assert.Len(suite.T(), history, 1)
}

// TestMetadataProviderOperations - commenting out until methods are implemented
// func (suite *LibraryRepositoryTestSuite) TestMetadataProviderOperations() { //nolint:funlen
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
// 	require.NoError(suite.T(), err)

// 	// Get provider
// 	retrieved, err := suite.repo.GetMetadataProvider(suite.ctx, provider.ID)
// 	require.NoError(suite.T(), err)
// 	assert.Equal(suite.T(), provider.Name, retrieved.Name)

// 	// List providers
// 	providers, err := suite.repo.ListMetadataProviders(suite.ctx)
// 	require.NoError(suite.T(), err)
// 	assert.Len(suite.T(), providers, 1)

// 	// Update provider
// 	provider.Enabled = false
// 	err = suite.repo.UpdateMetadataProvider(suite.ctx, provider)
// 	require.NoError(suite.T(), err)

// 	// Delete provider
// 	err = suite.repo.DeleteMetadataProvider(suite.ctx, provider.ID)
// 	require.NoError(suite.T(), err)
// }

func (suite *LibraryRepositoryTestSuite) TestTransaction() {
	// Start transaction
	tx, err := suite.repo.BeginTx(suite.ctx)
	suite.Require().NoError(err)

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
	suite.Require().NoError(err)

	// Rollback transaction
	err = tx.Rollback()
	suite.Require().NoError(err)

	// Verify library was not created
	retrieved, err := suite.repo.GetLibrary(suite.ctx, library.ID)
	suite.Require().Error(err)
	suite.Nil(retrieved)
}

func TestLibraryRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(LibraryRepositoryTestSuite))
}
