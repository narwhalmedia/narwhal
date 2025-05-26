package media

import (
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Domain events - these are handled within the media bounded context

// SeriesCreatedDomainEvent represents internal series creation event
type SeriesCreatedDomainEvent struct {
	events.BaseDomainEvent
	Series *Series `json:"series"`
}

// NewSeriesCreatedDomainEvent creates a new series creation domain event
func NewSeriesCreatedDomainEvent(series *Series) *SeriesCreatedDomainEvent {
	return &SeriesCreatedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			series.GetID(), 
			AggregateTypeSeries, 
			"series.created", 
			series.GetVersion(),
			BoundedContext,
		),
		Series: series,
	}
}

// EpisodeAddedDomainEvent represents internal episode addition event
type EpisodeAddedDomainEvent struct {
	events.BaseDomainEvent
	SeriesID  uuid.UUID `json:"series_id"`
	Episode   *Episode  `json:"episode"`
}

// NewEpisodeAddedDomainEvent creates a new episode addition domain event
func NewEpisodeAddedDomainEvent(series *Series, episode *Episode) *EpisodeAddedDomainEvent {
	return &EpisodeAddedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			series.GetID(), 
			AggregateTypeSeries, 
			"episode.added", 
			series.GetVersion(),
			BoundedContext,
		),
		SeriesID: series.GetID(),
		Episode:  episode,
	}
}

// EpisodeRemovedDomainEvent represents internal episode removal event
type EpisodeRemovedDomainEvent struct {
	events.BaseDomainEvent
	SeriesID  uuid.UUID `json:"series_id"`
	EpisodeID uuid.UUID `json:"episode_id"`
}

// NewEpisodeRemovedDomainEvent creates a new episode removal domain event
func NewEpisodeRemovedDomainEvent(series *Series, episodeID uuid.UUID) *EpisodeRemovedDomainEvent {
	return &EpisodeRemovedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			series.GetID(), 
			AggregateTypeSeries, 
			"episode.removed", 
			series.GetVersion(),
			BoundedContext,
		),
		SeriesID:  series.GetID(),
		EpisodeID: episodeID,
	}
}

// MediaStatusChangedDomainEvent represents internal media status change event
type MediaStatusChangedDomainEvent struct {
	events.BaseDomainEvent
	MediaID   uuid.UUID `json:"media_id"`
	MediaType string    `json:"media_type"`
	OldStatus Status    `json:"old_status"`
	NewStatus Status    `json:"new_status"`
}

// NewMediaStatusChangedDomainEvent creates a new media status change domain event
func NewMediaStatusChangedDomainEvent(media Aggregate, oldStatus, newStatus Status) *MediaStatusChangedDomainEvent {
	return &MediaStatusChangedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			media.GetID(), 
			getAggregateType(media), 
			"media.status_changed", 
			media.GetVersion(),
			BoundedContext,
		),
		MediaID:   media.GetID(),
		MediaType: getAggregateType(media),
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}
}

// MediaFileUpdatedDomainEvent represents internal media file update event
type MediaFileUpdatedDomainEvent struct {
	events.BaseDomainEvent
	MediaID       uuid.UUID     `json:"media_id"`
	MediaType     string        `json:"media_type"`
	FilePath      string        `json:"file_path"`
	ThumbnailPath string        `json:"thumbnail_path"`
	Duration      time.Duration `json:"duration"`
}

// NewMediaFileUpdatedDomainEvent creates a new media file update domain event
func NewMediaFileUpdatedDomainEvent(media Aggregate, filePath, thumbnailPath string, duration time.Duration) *MediaFileUpdatedDomainEvent {
	return &MediaFileUpdatedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			media.GetID(), 
			getAggregateType(media), 
			"media.file_updated", 
			media.GetVersion(),
			BoundedContext,
		),
		MediaID:       media.GetID(),
		MediaType:     getAggregateType(media),
		FilePath:      filePath,
		ThumbnailPath: thumbnailPath,
		Duration:      duration,
	}
}

// MovieCreatedDomainEvent represents internal movie creation event
type MovieCreatedDomainEvent struct {
	events.BaseDomainEvent
	Movie *Movie `json:"movie"`
}

// NewMovieCreatedDomainEvent creates a new movie creation domain event
func NewMovieCreatedDomainEvent(movie *Movie) *MovieCreatedDomainEvent {
	return &MovieCreatedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			movie.GetID(), 
			AggregateTypeMovie, 
			"movie.created", 
			movie.GetVersion(),
			BoundedContext,
		),
		Movie: movie,
	}
}

// SeriesDeletedDomainEvent represents internal series deletion event
type SeriesDeletedDomainEvent struct {
	events.BaseDomainEvent
	SeriesID uuid.UUID `json:"series_id"`
	Title    string    `json:"title"`
}

// NewSeriesDeletedDomainEvent creates a new series deletion domain event
func NewSeriesDeletedDomainEvent(series *Series) *SeriesDeletedDomainEvent {
	return &SeriesDeletedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			series.GetID(), 
			AggregateTypeSeries, 
			"series.deleted", 
			series.GetVersion(),
			BoundedContext,
		),
		SeriesID: series.GetID(),
		Title:    series.Title,
	}
}

// MovieDeletedDomainEvent represents internal movie deletion event
type MovieDeletedDomainEvent struct {
	events.BaseDomainEvent
	MovieID uuid.UUID `json:"movie_id"`
	Title   string    `json:"title"`
}

// NewMovieDeletedDomainEvent creates a new movie deletion domain event
func NewMovieDeletedDomainEvent(movie *Movie) *MovieDeletedDomainEvent {
	return &MovieDeletedDomainEvent{
		BaseDomainEvent: events.NewBaseDomainEvent(
			movie.GetID(), 
			AggregateTypeMovie, 
			"movie.deleted", 
			movie.GetVersion(),
			BoundedContext,
		),
		MovieID: movie.GetID(),
		Title:   movie.Title,
	}
}