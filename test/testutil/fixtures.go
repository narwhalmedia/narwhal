package testutil

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// CreateTestUser creates a test user with default values
func CreateTestUser(username, email string) *domain.User {
	user := &domain.User{
		ID:          uuid.New(),
		Username:    username,
		Email:       email,
		DisplayName: username,
		IsActive:    true,
		IsVerified:  true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Preferences: domain.UserPreferences{
			Language:         "en",
			Theme:            "dark",
			TimeZone:         "UTC",
			AutoPlayNext:     true,
			SubtitleLanguage: "en",
			PreferredQuality: "auto",
		},
	}
	user.SetPassword("testpass123")
	return user
}

// CreateTestRole creates a test role
func CreateTestRole(name, description string) *domain.Role {
	return &domain.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestPermission creates a test permission
func CreateTestPermission(resource, action, description string) *domain.Permission {
	return &domain.Permission{
		ID:          uuid.New(),
		Resource:    resource,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestSession creates a test session
func CreateTestSession(userID uuid.UUID) *domain.Session {
	return &domain.Session{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: uuid.New().String(),
		DeviceInfo:   "Test Device",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestLibrary creates a test library
func CreateTestLibrary(name, path string, mediaType models.MediaType) *models.Library {
	return &models.Library{
		ID:           uuid.New(),
		Name:         name,
		Path:         path,
		Type:         mediaType,
		AutoScan:     true,
		ScanInterval: 60, // minutes
		LastScanned:  time.Now(),
		Created:      time.Now(),
		Updated:      time.Now(),
	}
}

// CreateTestMedia creates a test media item
func CreateTestMedia(libraryID uuid.UUID, title string, mediaType models.MediaType) *models.Media {
	return &models.Media{
		ID:          uuid.New(),
		LibraryID:   libraryID,
		Title:       title,
		Type:        mediaType,
		Path:        "/test/media/" + title + ".mp4",
		Size:        1024 * 1024 * 100, // 100MB
		Duration:    3600,               // 1 hour
		Resolution:  "1920x1080",
		Codec:       "h264",
		Bitrate:     5000,
		Added:       time.Now(),
		Modified:    time.Now(),
		LastScanned: time.Now(),
		Status:      "available",
		FilePath:    "/test/media/" + title + ".mp4",
		FileSize:    1024 * 1024 * 100,
	}
}

// CreateTestMetadata creates test metadata for a media item
func CreateTestMetadata(mediaID uuid.UUID) *models.Metadata {
	return &models.Metadata{
		ID:          uuid.New(),
		MediaID:     mediaID,
		IMDBID:      "tt" + uuid.New().String()[:7],
		TMDBID:      "12345",
		TVDBID:      "67890",
		Description: "Test media description",
		ReleaseDate: "2020-01-01",
		Rating:      8.5,
		Genres:      []string{"Action", "Drama"},
		Cast:        []string{"Actor 1", "Actor 2"},
		Directors:   []string{"Director 1"},
		PosterURL:   "https://example.com/poster.jpg",
		BackdropURL: "https://example.com/backdrop.jpg",
		TrailerURL:  "https://example.com/trailer.mp4",
	}
}

// CreateTestEpisode creates a test episode
func CreateTestEpisode(mediaID uuid.UUID, season, episode int, title string) *models.Episode {
	return &models.Episode{
		ID:            uuid.New(),
		MediaID:       mediaID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         title,
		Path:          fmt.Sprintf("/test/series/s%02de%02d.mp4", season, episode),
		Duration:      2400, // 40 minutes
		AirDate:       time.Date(2020, 1, episode, 0, 0, 0, 0, time.UTC),
		Added:         time.Now(),
	}
}