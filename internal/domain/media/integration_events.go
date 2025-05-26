package media

import (
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Integration events - these are published to other bounded contexts

// MediaAddedIntegrationEvent is published when new media is added to the library
type MediaAddedIntegrationEvent struct {
	events.BaseIntegrationEvent
	MediaID   uuid.UUID `json:"media_id"`
	MediaType string    `json:"media_type"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
}

// NewMediaAddedIntegrationEvent creates a new media added integration event
func NewMediaAddedIntegrationEvent(media Aggregate, correlationID, causationID string) *MediaAddedIntegrationEvent {
	var title string
	switch m := media.(type) {
	case *Movie:
		title = m.Title
	case *Series:
		title = m.Title
	case *Episode:
		title = m.Title
	}

	return &MediaAddedIntegrationEvent{
		BaseIntegrationEvent: events.NewBaseIntegrationEvent(
			media.GetID(),
			getAggregateType(media),
			"media.added",
			media.GetVersion(),
			correlationID,
			causationID,
		),
		MediaID:   media.GetID(),
		MediaType: getAggregateType(media),
		Title:     title,
		Status:    media.GetStatus(),
	}
}

// MediaRemovedIntegrationEvent is published when media is removed from the library
type MediaRemovedIntegrationEvent struct {
	events.BaseIntegrationEvent
	MediaID   uuid.UUID `json:"media_id"`
	MediaType string    `json:"media_type"`
	Title     string    `json:"title"`
}

// NewMediaRemovedIntegrationEvent creates a new media removed integration event
func NewMediaRemovedIntegrationEvent(mediaID uuid.UUID, mediaType, title string, version int, correlationID, causationID string) *MediaRemovedIntegrationEvent {
	return &MediaRemovedIntegrationEvent{
		BaseIntegrationEvent: events.NewBaseIntegrationEvent(
			mediaID,
			mediaType,
			"media.removed",
			version,
			correlationID,
			causationID,
		),
		MediaID:   mediaID,
		MediaType: mediaType,
		Title:     title,
	}
}

// MediaTranscodeRequestedIntegrationEvent is published when media needs transcoding
type MediaTranscodeRequestedIntegrationEvent struct {
	events.BaseIntegrationEvent
	MediaID       uuid.UUID `json:"media_id"`
	MediaType     string    `json:"media_type"`
	FilePath      string    `json:"file_path"`
	TargetFormats []string  `json:"target_formats"`
}

// NewMediaTranscodeRequestedIntegrationEvent creates a new transcode requested event
func NewMediaTranscodeRequestedIntegrationEvent(mediaID uuid.UUID, mediaType, filePath string, targetFormats []string, version int, correlationID, causationID string) *MediaTranscodeRequestedIntegrationEvent {
	return &MediaTranscodeRequestedIntegrationEvent{
		BaseIntegrationEvent: events.NewBaseIntegrationEvent(
			mediaID,
			mediaType,
			"media.transcode_requested",
			version,
			correlationID,
			causationID,
		),
		MediaID:       mediaID,
		MediaType:     mediaType,
		FilePath:      filePath,
		TargetFormats: targetFormats,
	}
}

// MediaReadyToStreamIntegrationEvent is published when media is ready for streaming
type MediaReadyToStreamIntegrationEvent struct {
	events.BaseIntegrationEvent
	MediaID       uuid.UUID     `json:"media_id"`
	MediaType     string        `json:"media_type"`
	StreamingURL  string        `json:"streaming_url"`
	Duration      time.Duration `json:"duration"`
	ThumbnailPath string        `json:"thumbnail_path"`
}

// NewMediaReadyToStreamIntegrationEvent creates a new ready to stream event
func NewMediaReadyToStreamIntegrationEvent(mediaID uuid.UUID, mediaType, streamingURL string, duration time.Duration, thumbnailPath string, version int, correlationID, causationID string) *MediaReadyToStreamIntegrationEvent {
	return &MediaReadyToStreamIntegrationEvent{
		BaseIntegrationEvent: events.NewBaseIntegrationEvent(
			mediaID,
			mediaType,
			"media.ready_to_stream",
			version,
			correlationID,
			causationID,
		),
		MediaID:       mediaID,
		MediaType:     mediaType,
		StreamingURL:  streamingURL,
		Duration:      duration,
		ThumbnailPath: thumbnailPath,
	}
}

// EpisodeAddedToSeriesIntegrationEvent is published when an episode is added to a series
type EpisodeAddedToSeriesIntegrationEvent struct {
	events.BaseIntegrationEvent
	SeriesID      uuid.UUID `json:"series_id"`
	EpisodeID     uuid.UUID `json:"episode_id"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	Title         string    `json:"title"`
}

// NewEpisodeAddedToSeriesIntegrationEvent creates a new episode added integration event
func NewEpisodeAddedToSeriesIntegrationEvent(seriesID, episodeID uuid.UUID, seasonNumber, episodeNumber int, title string, version int, correlationID, causationID string) *EpisodeAddedToSeriesIntegrationEvent {
	return &EpisodeAddedToSeriesIntegrationEvent{
		BaseIntegrationEvent: events.NewBaseIntegrationEvent(
			seriesID,
			AggregateTypeSeries,
			"episode.added_to_series",
			version,
			correlationID,
			causationID,
		),
		SeriesID:      seriesID,
		EpisodeID:     episodeID,
		SeasonNumber:  seasonNumber,
		EpisodeNumber: episodeNumber,
		Title:         title,
	}
}