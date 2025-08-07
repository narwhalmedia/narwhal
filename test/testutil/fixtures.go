package testutil

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/narwhalmedia/narwhal/pkg/models"
)

// CreateTestUser creates a test user with default values.
func CreateTestUser(username, email string) *models.User {
	user := &models.User{
		ID:                      uuid.New(),
		Username:                username,
		Email:                   email,
		DisplayName:             username,
		IsActive:                true,
		IsVerified:              true,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
		PrefLanguage:            "en",
		PrefTheme:               "dark",
		PrefTimeZone:            "UTC",
		PrefAutoPlayNext:        true,
		PrefSubtitleLanguage:    "en",
		PrefPreferredQuality:    "auto",
		PrefEnableNotifications: true,
	}
	user.SetPassword("testpass123")
	return user
}

// CreateTestRole creates a test role.
func CreateTestRole(name, description string) *models.Role {
	return &models.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestPermission creates a test permission.
func CreateTestPermission(resource, action, description string) *models.Permission {
	return &models.Permission{
		ID:          uuid.New(),
		Resource:    resource,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestSession creates a test session.
func CreateTestSession(userID uuid.UUID) *models.Session {
	return &models.Session{
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

// CreateTestLibrary creates a test library.
func CreateTestLibrary(name, path string, mediaType models.MediaType) *models.Library {
	now := time.Now()
	return &models.Library{
		ID:           uuid.New(),
		Name:         name,
		Path:         path,
		Type:         mediaType,
		Enabled:      true,
		ScanInterval: 3600, // seconds
		LastScanAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestMedia creates a test media item.
func CreateTestMedia(libraryID uuid.UUID, title string, mediaType models.MediaType) *models.Media {
	now := time.Now()
	return &models.Media{
		ID:             uuid.New(),
		LibraryID:      libraryID,
		Title:          title,
		Type:           mediaType,
		FilePath:       "/test/media/" + title + ".mp4",
		FileSize:       1024 * 1024 * 100, // 100MB
		Runtime:        60,                // 1 hour in minutes
		Resolution:     "1920x1080",
		VideoCodec:     "h264",
		Bitrate:        5000,
		CreatedAt:      now,
		UpdatedAt:      now,
		FileModifiedAt: &now,
		Status:         "available",
	}
}

// CreateTestMetadata creates test metadata for a media item.
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

// CreateTestEpisode creates a test episode.
func CreateTestEpisode(mediaID uuid.UUID, season, episode int, title string) *models.Episode {
	now := time.Now()
	airDate := time.Date(2020, 1, episode, 0, 0, 0, 0, time.UTC)
	return &models.Episode{
		ID:            uuid.New(),
		MediaID:       mediaID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         title,
		FilePath:      fmt.Sprintf("/test/series/s%02de%02d.mp4", season, episode),
		Runtime:       40, // 40 minutes
		AirDate:       &airDate,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
