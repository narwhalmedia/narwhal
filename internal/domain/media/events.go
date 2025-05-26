package media

import (
	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

const (
	// BoundedContext for media domain
	BoundedContext = "media"
	
	// Aggregate types
	AggregateTypeMovie   = "movie"
	AggregateTypeSeries  = "series"
	AggregateTypeEpisode = "episode"
)

// getAggregateType returns the type of the aggregate based on its concrete type
func getAggregateType(agg Aggregate) string {
	switch agg.(type) {
	case *Movie:
		return AggregateTypeMovie
	case *Series:
		return AggregateTypeSeries
	case *Episode:
		return AggregateTypeEpisode
	default:
		return "unknown"
	}
}
	return e.AggregateType
}

// GetEventType returns the type of the event
func (e BaseEvent) GetEventType() string {
	return e.EventType
}

// GetTimestamp returns when the event occurred
func (e BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// NewBaseEvent creates a new base event
func NewBaseEvent(aggregateID uuid.UUID, aggregateType, eventType string) BaseEvent {
	return BaseEvent{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		Timestamp:     time.Now(),
	}
}

// SeriesCreated represents a series creation event
type SeriesCreated struct {
	events.BaseEvent
	Series *Series
}

// NewSeriesCreated creates a new series creation event
func NewSeriesCreated(series *Series) *SeriesCreated {
	return &SeriesCreated{
		BaseEvent: events.NewBaseEvent(series.GetID(), "series", "series.created", series.GetVersion()),
		Series:    series,
	}
}

// EpisodeAdded represents an episode addition event
type EpisodeAdded struct {
	events.BaseEvent
	Series  *Series
	Episode *Episode
}

// NewEpisodeAdded creates a new episode addition event
func NewEpisodeAdded(series *Series, episode *Episode) *EpisodeAdded {
	return &EpisodeAdded{
		BaseEvent: events.NewBaseEvent(series.GetID(), "series", "episode.added", series.GetVersion()),
		Series:    series,
		Episode:   episode,
	}
}

// EpisodeRemoved represents an episode removal event
type EpisodeRemoved struct {
	events.BaseEvent
	SeriesID  uuid.UUID
	EpisodeID uuid.UUID
}

// NewEpisodeRemoved creates a new episode removal event
func NewEpisodeRemoved(series *Series, episodeID uuid.UUID) *EpisodeRemoved {
	return &EpisodeRemoved{
		BaseEvent:  events.NewBaseEvent(series.GetID(), "series", "episode.removed", series.GetVersion()),
		SeriesID:   series.GetID(),
		EpisodeID:  episodeID,
	}
}

// MediaStatusChanged represents a media status change event
type MediaStatusChanged struct {
	events.BaseEvent
	MediaID     uuid.UUID
	MediaType   string
	OldStatus   Status
	NewStatus   Status
}

// NewMediaStatusChanged creates a new media status change event
func NewMediaStatusChanged(media Aggregate, oldStatus, newStatus Status) *MediaStatusChanged {
	return &MediaStatusChanged{
		BaseEvent:  events.NewBaseEvent(media.GetID(), getAggregateType(media), "media.status_changed", media.GetVersion()),
		MediaID:    media.GetID(),
		MediaType:  getAggregateType(media),
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
	}
}

// MediaFileUpdated represents a media file update event
type MediaFileUpdated struct {
	events.BaseEvent
	MediaID       uuid.UUID
	MediaType     string
	FilePath      string
	ThumbnailPath string
	Duration      time.Duration
}

// NewMediaFileUpdated creates a new media file update event
func NewMediaFileUpdated(media Aggregate, filePath, thumbnailPath string, duration time.Duration) *MediaFileUpdated {
	return &MediaFileUpdated{
		BaseEvent:     events.NewBaseEvent(media.GetID(), getAggregateType(media), "media.file_updated", media.GetVersion()),
		MediaID:       media.GetID(),
		MediaType:     getAggregateType(media),
		FilePath:      filePath,
		ThumbnailPath: thumbnailPath,
		Duration:      duration,
	}
}

// MovieCreated represents a movie creation event
type MovieCreated struct {
	events.BaseEvent
	Movie *Movie
}

// NewMovieCreated creates a new movie creation event
func NewMovieCreated(movie *Movie) *MovieCreated {
	return &MovieCreated{
		BaseEvent: events.NewBaseEvent(movie.GetID(), "movie", "movie.created", movie.GetVersion()),
		Movie:     movie,
	}
} 

// SeriesDeleted represents a series deletion event
type SeriesDeleted struct {
	events.BaseEvent
	SeriesID uuid.UUID `json:"series_id"`
	Title    string    `json:"title"`
}

// NewSeriesDeleted creates a new series deletion event
func NewSeriesDeleted(series *Series) *SeriesDeleted {
	return &SeriesDeleted{
		BaseEvent: events.NewBaseEvent(series.GetID(), "series", "series.deleted", series.GetVersion()),
		SeriesID:  series.GetID(),
		Title:     series.Title,
	}
}

// MovieDeleted represents a movie deletion event
type MovieDeleted struct {
	events.BaseEvent
	MovieID uuid.UUID `json:"movie_id"`
	Title   string    `json:"title"`
}

// NewMovieDeleted creates a new movie deletion event
func NewMovieDeleted(movie *Movie) *MovieDeleted {
	return &MovieDeleted{
		BaseEvent: events.NewBaseEvent(movie.GetID(), "movie", "movie.deleted", movie.GetVersion()),
		MovieID:   movie.GetID(),
		Title:     movie.Title,
	}
}
