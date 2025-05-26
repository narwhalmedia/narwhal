package media

import (
	"context"
	"fmt"

	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// EventHandler handles media events
type EventHandler struct {
	seriesRepo SeriesRepository
	movieRepo  MovieRepository
}

// NewEventHandler creates a new media event handler
func NewEventHandler(seriesRepo SeriesRepository, movieRepo MovieRepository) *EventHandler {
	return &EventHandler{
		seriesRepo: seriesRepo,
		movieRepo:  movieRepo,
	}
}

// HandleEvent handles media events
func (h *EventHandler) HandleEvent(ctx context.Context, msg events.Message) error {
	switch msg.EventType {
	case "series.created":
		return h.handleSeriesCreated(ctx, msg)
	case "episode.added":
		return h.handleEpisodeAdded(ctx, msg)
	case "episode.removed":
		return h.handleEpisodeRemoved(ctx, msg)
	case "media.status_changed":
		return h.handleMediaStatusChanged(ctx, msg)
	case "media.file_updated":
		return h.handleMediaFileUpdated(ctx, msg)
	case "movie.created":
		return h.handleMovieCreated(ctx, msg)
	default:
		return fmt.Errorf("unknown event type: %s", msg.EventType)
	}
}

func (h *EventHandler) handleSeriesCreated(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*SeriesCreated)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	// Save series
	if err := h.seriesRepo.Save(ctx, event.Series); err != nil {
		return fmt.Errorf("saving series: %w", err)
	}

	return nil
}

func (h *EventHandler) handleEpisodeAdded(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*EpisodeAdded)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	// Save series with new episode
	if err := h.seriesRepo.Save(ctx, event.Series); err != nil {
		return fmt.Errorf("saving series: %w", err)
	}

	return nil
}

func (h *EventHandler) handleEpisodeRemoved(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*EpisodeRemoved)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	// Get series
	series, err := h.seriesRepo.FindByID(ctx, event.SeriesID)
	if err != nil {
		return fmt.Errorf("finding series: %w", err)
	}

	// Remove episode
	if err := series.RemoveEpisode(event.EpisodeID); err != nil {
		return fmt.Errorf("removing episode: %w", err)
	}

	// Save series
	if err := h.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("saving series: %w", err)
	}

	return nil
}

func (h *EventHandler) handleMediaStatusChanged(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*MediaStatusChanged)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	switch event.MediaType {
	case "series":
		// Get series
		series, err := h.seriesRepo.FindByID(ctx, event.MediaID)
		if err != nil {
			return fmt.Errorf("finding series: %w", err)
		}

		// Update status
		series.UpdateStatus(event.NewStatus)

		// Save series
		if err := h.seriesRepo.Save(ctx, series); err != nil {
			return fmt.Errorf("saving series: %w", err)
		}

	case "movie":
		// Get movie
		movie, err := h.movieRepo.FindByID(ctx, event.MediaID)
		if err != nil {
			return fmt.Errorf("finding movie: %w", err)
		}

		// Update status
		movie.UpdateStatus(event.NewStatus)

		// Save movie
		if err := h.movieRepo.Save(ctx, movie); err != nil {
			return fmt.Errorf("saving movie: %w", err)
		}

	default:
		return fmt.Errorf("unknown media type: %s", event.MediaType)
	}

	return nil
}

func (h *EventHandler) handleMediaFileUpdated(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*MediaFileUpdated)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	switch event.MediaType {
	case "series":
		// Get series
		series, err := h.seriesRepo.FindByID(ctx, event.MediaID)
		if err != nil {
			return fmt.Errorf("finding series: %w", err)
		}

		// Update file info
		series.UpdateFilePath(event.FilePath)
		series.UpdateThumbnailPath(event.ThumbnailPath)
		series.UpdateDuration(int(event.Duration))

		// Save series
		if err := h.seriesRepo.Save(ctx, series); err != nil {
			return fmt.Errorf("saving series: %w", err)
		}

	case "movie":
		// Get movie
		movie, err := h.movieRepo.FindByID(ctx, event.MediaID)
		if err != nil {
			return fmt.Errorf("finding movie: %w", err)
		}

		// Update file info
		movie.UpdateFilePath(event.FilePath)
		movie.UpdateThumbnailPath(event.ThumbnailPath)
		movie.UpdateDuration(int(event.Duration))

		// Save movie
		if err := h.movieRepo.Save(ctx, movie); err != nil {
			return fmt.Errorf("saving movie: %w", err)
		}

	default:
		return fmt.Errorf("unknown media type: %s", event.MediaType)
	}

	return nil
}

func (h *EventHandler) handleMovieCreated(ctx context.Context, msg events.Message) error {
	event, ok := msg.Data.(*MovieCreated)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	// Save movie
	if err := h.movieRepo.Save(ctx, event.Movie); err != nil {
		return fmt.Errorf("saving movie: %w", err)
	}

	return nil
} 