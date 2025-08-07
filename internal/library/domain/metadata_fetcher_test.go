package domain_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// MockMetadataProvider is a mock implementation of a metadata provider.
type MockMetadataProvider struct {
	mock.Mock
}

func (m *MockMetadataProvider) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMetadataProvider) GetType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMetadataProvider) SearchMovie(ctx context.Context, query string, year int) ([]models.SearchResult, error) {
	args := m.Called(ctx, query, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SearchResult), args.Error(1)
}

func (m *MockMetadataProvider) SearchTV(ctx context.Context, query string, year int) ([]models.SearchResult, error) {
	args := m.Called(ctx, query, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SearchResult), args.Error(1)
}

func (m *MockMetadataProvider) GetMovieDetails(ctx context.Context, providerID string) (*models.Metadata, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Metadata), args.Error(1)
}

func (m *MockMetadataProvider) GetTVDetails(ctx context.Context, providerID string) (*models.Metadata, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Metadata), args.Error(1)
}

func (m *MockMetadataProvider) GetEpisodeDetails(
	ctx context.Context,
	providerID string,
	season, episode int,
) (*models.EpisodeMetadata, error) {
	args := m.Called(ctx, providerID, season, episode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EpisodeMetadata), args.Error(1)
}

type MetadataFetcherTestSuite struct {
	suite.Suite

	ctx          context.Context
	fetcher      *domain.MetadataFetcher
	mockProvider *MockMetadataProvider
}

func (suite *MetadataFetcherTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockProvider = new(MockMetadataProvider)
	suite.fetcher = domain.NewMetadataFetcher(logger.NewNoopLogger())

	// Setup default mock behavior
	suite.mockProvider.On("GetName").Return("TestProvider").Maybe()
	suite.mockProvider.On("GetType").Return("all").Maybe()

	// Register the mock provider
	suite.fetcher.RegisterProvider(suite.mockProvider)
}

func (suite *MetadataFetcherTestSuite) TearDownTest() {
	suite.mockProvider.AssertExpectations(suite.T())
}

func (suite *MetadataFetcherTestSuite) TestFetchMovieMetadata_Success() {
	// Arrange
	media := &models.Media{
		ID:    uuid.New(),
		Title: "Test Movie",
		Type:  models.MediaTypeMovie,
		Year:  2023,
	}

	searchResults := []models.SearchResult{
		{
			ProviderID:   "tmdb123",
			Title:        "Test Movie",
			Year:         2023,
			ProviderName: "TMDB",
		},
	}

	expectedMetadata := &models.Metadata{
		ID:          uuid.New(),
		MediaID:     media.ID,
		TMDBID:      "tmdb123",
		Title:       "Test Movie",
		Description: "A test movie",
		ReleaseDate: "2023-01-01",
		Rating:      8.5,
		Genres:      []string{"Action", "Drama"},
	}

	suite.mockProvider.On("GetType").Return("movie")
	suite.mockProvider.On("SearchMovie", suite.ctx, "Test Movie", 2023).Return(searchResults, nil)
	suite.mockProvider.On("GetMovieDetails", suite.ctx, "tmdb123").Return(expectedMetadata, nil)

	// Act
	metadata, err := suite.fetcher.FetchMetadata(suite.ctx, media)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(metadata)
	suite.Equal(expectedMetadata.Title, metadata.Title)
	suite.Equal(expectedMetadata.Description, metadata.Description)
	suite.Equal(expectedMetadata.TMDBID, metadata.TMDBID)
}

func (suite *MetadataFetcherTestSuite) TestFetchTVMetadata_Success() {
	// Arrange
	media := &models.Media{
		ID:    uuid.New(),
		Title: "Test Series",
		Type:  models.MediaTypeTV,
		Year:  2023,
	}

	searchResults := []models.SearchResult{
		{
			ProviderID:   "tvdb123",
			Title:        "Test Series",
			Year:         2023,
			ProviderName: "TVDB",
		},
	}

	expectedMetadata := &models.Metadata{
		ID:          uuid.New(),
		MediaID:     media.ID,
		TVDBID:      "tvdb123",
		Title:       "Test Series",
		Description: "A test TV series",
		ReleaseDate: "2023-01-01",
		Rating:      8.0,
		Genres:      []string{"Drama", "Mystery"},
	}

	suite.mockProvider.On("GetType").Return("tv")
	suite.mockProvider.On("SearchTV", suite.ctx, "Test Series", 2023).Return(searchResults, nil)
	suite.mockProvider.On("GetTVDetails", suite.ctx, "tvdb123").Return(expectedMetadata, nil)

	// Act
	metadata, err := suite.fetcher.FetchMetadata(suite.ctx, media)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(metadata)
	suite.Equal(expectedMetadata.Title, metadata.Title)
	suite.Equal(expectedMetadata.TVDBID, metadata.TVDBID)
}

func (suite *MetadataFetcherTestSuite) TestFetchMetadata_NoResults() {
	// Arrange
	media := &models.Media{
		ID:    uuid.New(),
		Title: "Unknown Movie",
		Type:  models.MediaTypeMovie,
		Year:  2023,
	}

	suite.mockProvider.On("GetType").Return("movie")
	suite.mockProvider.On("SearchMovie", suite.ctx, "Unknown Movie", 2023).Return([]models.SearchResult{}, nil)

	// Act
	metadata, err := suite.fetcher.FetchMetadata(suite.ctx, media)

	// Assert
	suite.Require().Error(err)
	suite.Nil(metadata)
	suite.Contains(err.Error(), "no metadata found")
}

func (suite *MetadataFetcherTestSuite) TestFetchMetadata_InvalidMediaType() {
	// Arrange
	media := &models.Media{
		ID:    uuid.New(),
		Title: "Test",
		Type:  "invalid",
	}

	// Act
	metadata, err := suite.fetcher.FetchMetadata(suite.ctx, media)

	// Assert
	suite.Require().Error(err)
	suite.Nil(metadata)
	suite.Contains(err.Error(), "unsupported media type")
}

func (suite *MetadataFetcherTestSuite) TestFetchEpisodeMetadata_Success() {
	// Arrange
	seriesMetadata := &models.Metadata{
		ID:     uuid.New(),
		TVDBID: "tvdb123",
	}

	expectedEpisodeMetadata := &models.EpisodeMetadata{
		ID:            uuid.New(),
		Title:         "Episode Title",
		Description:   "Episode description",
		AirDate:       "2023-01-15",
		SeasonNumber:  1,
		EpisodeNumber: 1,
	}

	suite.mockProvider.On("GetType").Return("tv")
	suite.mockProvider.On("GetEpisodeDetails", suite.ctx, "tvdb123", 1, 1).Return(expectedEpisodeMetadata, nil)

	// Act
	episodeMetadata, err := suite.fetcher.FetchEpisodeMetadata(suite.ctx, seriesMetadata, 1, 1)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(episodeMetadata)
	suite.Equal(expectedEpisodeMetadata.Title, episodeMetadata.Title)
	suite.Equal(expectedEpisodeMetadata.Description, episodeMetadata.Description)
}

func (suite *MetadataFetcherTestSuite) TestGetProviders() {
	// Arrange
	suite.mockProvider.On("GetName").Return("TestProvider")

	// Act
	providers := suite.fetcher.GetProviders()

	// Assert
	suite.Len(providers, 1)
	suite.Equal("TestProvider", providers[0].GetName())
}

func (suite *MetadataFetcherTestSuite) TestRegisterProvider_Duplicate() {
	// Arrange
	mockProvider2 := new(MockMetadataProvider)
	mockProvider2.On("GetName").Return("TestProvider")
	suite.mockProvider.On("GetName").Return("TestProvider")

	// Act - Try to register duplicate provider
	suite.fetcher.RegisterProvider(mockProvider2)

	// Assert - Should still only have one provider
	providers := suite.fetcher.GetProviders()
	suite.Len(providers, 1)
}

func (suite *MetadataFetcherTestSuite) TestFetchMetadata_MultipleProviders() {
	// Arrange
	media := &models.Media{
		ID:    uuid.New(),
		Title: "Test Movie",
		Type:  models.MediaTypeMovie,
		Year:  2023,
	}

	// Create second provider that will succeed
	mockProvider2 := new(MockMetadataProvider)
	mockProvider2.On("GetName").Return("SecondProvider")
	mockProvider2.On("GetType").Return("movie")

	// First provider returns no results
	suite.mockProvider.On("GetType").Return("movie")
	suite.mockProvider.On("SearchMovie", suite.ctx, "Test Movie", 2023).Return([]models.SearchResult{}, nil)

	// Second provider returns results
	searchResults := []models.SearchResult{
		{
			ProviderID:   "provider2_123",
			Title:        "Test Movie",
			Year:         2023,
			ProviderName: "SecondProvider",
		},
	}

	expectedMetadata := &models.Metadata{
		ID:      uuid.New(),
		MediaID: media.ID,
		Title:   "Test Movie",
	}

	mockProvider2.On("SearchMovie", suite.ctx, "Test Movie", 2023).Return(searchResults, nil)
	mockProvider2.On("GetMovieDetails", suite.ctx, "provider2_123").Return(expectedMetadata, nil)

	// Register second provider
	suite.fetcher.RegisterProvider(mockProvider2)

	// Act
	metadata, err := suite.fetcher.FetchMetadata(suite.ctx, media)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(metadata)
	suite.Equal(expectedMetadata.Title, metadata.Title)

	// Cleanup
	mockProvider2.AssertExpectations(suite.T())
}

func TestMetadataFetcherTestSuite(t *testing.T) {
	suite.Run(t, new(MetadataFetcherTestSuite))
}
