package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	"github.com/narwhalmedia/narwhal/test/testutil"
)

type EncryptionTestSuite struct {
	suite.Suite

	container *testutil.PostgresContainer
	repo      repository.Repository
	ctx       context.Context
}

func TestEncryptionSuite(t *testing.T) {
	suite.Run(t, new(EncryptionTestSuite))
}

func (suite *EncryptionTestSuite) SetupSuite() {
	// Set encryption key
	os.Setenv("NARWHAL_ENCRYPTION_KEY", "test-encryption-key-for-testing")

	suite.container = testutil.SetupPostgresContainer(suite.T())

	// Run migrations
	err := suite.container.DB.AutoMigrate(
		&repository.Library{},
		&repository.MediaItem{},
		&repository.Episode{},
		&repository.MetadataProvider{},
		&repository.ScanHistory{},
	)
	suite.Require().NoError(err)
}

func (suite *EncryptionTestSuite) SetupTest() {
	var err error
	suite.repo, err = repository.NewGormRepository(suite.container.DB)
	suite.Require().NoError(err)

	suite.ctx = context.Background()

	// Clean tables before each test
	suite.container.TruncateTables("metadata_providers")
}

func (suite *EncryptionTestSuite) TestAPIKeyEncryption() {
	// Create a provider with an API key
	provider := &domain.MetadataProviderConfig{
		Name:         "TMDB",
		ProviderType: "tmdb",
		APIKey:       "sk-test-api-key-123456789",
		Enabled:      true,
		Priority:     1,
	}

	// Create provider
	err := suite.repo.CreateProvider(suite.ctx, provider)
	suite.Require().NoError(err)
	suite.Require().NotEqual(uuid.Nil, provider.ID)

	// Query the database directly to verify encryption
	var dbProvider repository.MetadataProvider
	err = suite.container.DB.First(&dbProvider, "id = ?", provider.ID).Error
	suite.Require().NoError(err)

	// API key should be encrypted in database
	suite.NotEqual("sk-test-api-key-123456789", dbProvider.APIKey)
	suite.NotEmpty(dbProvider.APIKey)

	// Retrieve provider through repository
	retrieved, err := suite.repo.GetProvider(suite.ctx, provider.ID)
	suite.Require().NoError(err)

	// API key should be decrypted
	suite.Equal("sk-test-api-key-123456789", retrieved.APIKey)
}

func (suite *EncryptionTestSuite) TestUpdateProviderEncryption() {
	// Create a provider
	provider := &domain.MetadataProviderConfig{
		Name:         "TVDB",
		ProviderType: "tvdb",
		APIKey:       "original-api-key",
		Enabled:      true,
		Priority:     2,
	}

	err := suite.repo.CreateProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Update the API key
	provider.APIKey = "updated-api-key-456"
	err = suite.repo.UpdateProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Query the database directly
	var dbProvider repository.MetadataProvider
	err = suite.container.DB.First(&dbProvider, "id = ?", provider.ID).Error
	suite.Require().NoError(err)

	// Updated API key should be encrypted
	suite.NotEqual("updated-api-key-456", dbProvider.APIKey)
	suite.NotEqual("original-api-key", dbProvider.APIKey)

	// Retrieve and verify decryption
	retrieved, err := suite.repo.GetProvider(suite.ctx, provider.ID)
	suite.Require().NoError(err)
	suite.Equal("updated-api-key-456", retrieved.APIKey)
}

func (suite *EncryptionTestSuite) TestEmptyAPIKey() {
	// Create a provider with empty API key
	provider := &domain.MetadataProviderConfig{
		Name:         "MusicBrainz",
		ProviderType: "musicbrainz",
		APIKey:       "",
		Enabled:      true,
		Priority:     3,
	}

	err := suite.repo.CreateProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Retrieve and verify empty key is handled correctly
	retrieved, err := suite.repo.GetProvider(suite.ctx, provider.ID)
	suite.Require().NoError(err)
	suite.Empty(retrieved.APIKey)
}

func (suite *EncryptionTestSuite) TestListProvidersDecryption() {
	// Create multiple providers
	providers := []*domain.MetadataProviderConfig{
		{
			Name:         "Provider1",
			ProviderType: "type1",
			APIKey:       "key1-secret",
			Enabled:      true,
			Priority:     1,
		},
		{
			Name:         "Provider2",
			ProviderType: "type2",
			APIKey:       "key2-secret",
			Enabled:      true,
			Priority:     2,
		},
	}

	for _, p := range providers {
		err := suite.repo.CreateProvider(suite.ctx, p)
		suite.Require().NoError(err)
	}

	// List all providers
	retrieved, err := suite.repo.ListProviders(suite.ctx, nil, nil)
	suite.Require().NoError(err)
	suite.Require().Len(retrieved, 2)

	// Verify all API keys are decrypted
	for i, p := range retrieved {
		switch p.Name {
		case "Provider1":
			suite.Equal("key1-secret", p.APIKey)
		case "Provider2":
			suite.Equal("key2-secret", p.APIKey)
		}
		suite.NotEmpty(providers[i].ID)
	}
}
