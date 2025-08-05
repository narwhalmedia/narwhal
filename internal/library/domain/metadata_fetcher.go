package domain

import (
	"context"
	"fmt"
	"sync"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// MetadataProvider interface for external metadata providers
type MetadataProvider interface {
	GetName() string
	GetType() string
	SearchMovie(ctx context.Context, query string, year int) ([]models.SearchResult, error)
	SearchTV(ctx context.Context, query string, year int) ([]models.SearchResult, error)
	GetMovieDetails(ctx context.Context, providerID string) (*models.Metadata, error)
	GetTVDetails(ctx context.Context, providerID string) (*models.Metadata, error)
	GetEpisodeDetails(ctx context.Context, providerID string, season, episode int) (*models.EpisodeMetadata, error)
}

// MetadataFetcher manages metadata providers and fetching
type MetadataFetcher struct {
	providers []MetadataProvider
	mu        sync.RWMutex
	logger    interfaces.Logger
}

// NewMetadataFetcher creates a new metadata fetcher
func NewMetadataFetcher(logger interfaces.Logger) *MetadataFetcher {
	return &MetadataFetcher{
		providers: make([]MetadataProvider, 0),
		logger:    logger,
	}
}

// RegisterProvider registers a metadata provider
func (f *MetadataFetcher) RegisterProvider(provider MetadataProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if provider already exists
	for i, p := range f.providers {
		if p.GetName() == provider.GetName() {
			// Replace existing provider
			f.providers[i] = provider
			return
		}
	}

	// Add new provider
	f.providers = append(f.providers, provider)
	f.logger.Info("Registered metadata provider",
		interfaces.String("provider", provider.GetName()),
		interfaces.String("type", provider.GetType()))
}

// GetProviders returns all registered providers
func (f *MetadataFetcher) GetProviders() []MetadataProvider {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]MetadataProvider, len(f.providers))
	copy(providers, f.providers)
	return providers
}

// FetchMetadata fetches metadata for a media item
func (f *MetadataFetcher) FetchMetadata(ctx context.Context, media *models.Media) (*models.Metadata, error) {
	f.mu.RLock()
	providers := make([]MetadataProvider, len(f.providers))
	copy(providers, f.providers)
	f.mu.RUnlock()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no metadata providers registered")
	}

	// Try each provider
	for _, provider := range providers {
		var searchResults []models.SearchResult
		var err error

		// Search based on media type
		switch media.Type {
		case models.MediaTypeMovie:
			if provider.GetType() != "movie" && provider.GetType() != "all" {
				continue
			}
			searchResults, err = provider.SearchMovie(ctx, media.Title, media.Year)
		case models.MediaTypeTV, models.MediaTypeSeries:
			if provider.GetType() != "tv" && provider.GetType() != "all" {
				continue
			}
			searchResults, err = provider.SearchTV(ctx, media.Title, media.Year)
		default:
			return nil, fmt.Errorf("unsupported media type: %s", media.Type)
		}

		if err != nil {
			f.logger.Error("Provider search failed",
				interfaces.String("provider", provider.GetName()),
				interfaces.String("error", err.Error()))
			continue
		}

		if len(searchResults) == 0 {
			continue
		}

		// Use the first result
		result := searchResults[0]

		// Get detailed metadata
		var metadata *models.Metadata
		switch media.Type {
		case models.MediaTypeMovie:
			metadata, err = provider.GetMovieDetails(ctx, result.ProviderID)
		case models.MediaTypeTV, models.MediaTypeSeries:
			metadata, err = provider.GetTVDetails(ctx, result.ProviderID)
		}

		if err != nil {
			f.logger.Error("Failed to get metadata details",
				interfaces.String("provider", provider.GetName()),
				interfaces.String("error", err.Error()))
			continue
		}

		if metadata != nil {
			metadata.MediaID = media.ID
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("no metadata found for media: %s", media.Title)
}

// FetchEpisodeMetadata fetches metadata for a specific episode
func (f *MetadataFetcher) FetchEpisodeMetadata(ctx context.Context, seriesMetadata *models.Metadata, season, episode int) (*models.EpisodeMetadata, error) {
	f.mu.RLock()
	providers := make([]MetadataProvider, len(f.providers))
	copy(providers, f.providers)
	f.mu.RUnlock()

	// Try each provider that supports TV
	for _, provider := range providers {
		if provider.GetType() != "tv" && provider.GetType() != "all" {
			continue
		}

		// Try different provider IDs
		var providerID string
		if seriesMetadata.TVDBID != "" {
			providerID = seriesMetadata.TVDBID
		} else if seriesMetadata.TMDBID != "" {
			providerID = seriesMetadata.TMDBID
		} else {
			continue
		}

		episodeMetadata, err := provider.GetEpisodeDetails(ctx, providerID, season, episode)
		if err != nil {
			f.logger.Error("Failed to get episode metadata",
				interfaces.String("provider", provider.GetName()),
				interfaces.String("error", err.Error()))
			continue
		}

		if episodeMetadata != nil {
			return episodeMetadata, nil
		}
	}

	return nil, fmt.Errorf("no episode metadata found for S%02dE%02d", season, episode)
}
