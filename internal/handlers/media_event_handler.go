package handlers

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
	"github.com/narwhalmedia/narwhal/internal/sagas"
)

// MediaEventHandler handles media domain events
type MediaEventHandler struct {
	orchestrator *nats.SagaOrchestrator
	logger       *zap.Logger
}

// NewMediaEventHandler creates a new media event handler
func NewMediaEventHandler(orchestrator *nats.SagaOrchestrator, logger *zap.Logger) *MediaEventHandler {
	return &MediaEventHandler{
		orchestrator: orchestrator,
		logger:       logger.Named("media-event-handler"),
	}
}

// Handle processes media events
func (h *MediaEventHandler) Handle(ctx context.Context, event domainevents.Event) error {
	switch e := event.(type) {
	case *media.MediaCreated:
		return h.handleMediaCreated(ctx, e)
	case *media.MediaFileUpdated:
		return h.handleMediaFileUpdated(ctx, e)
	default:
		h.logger.Debug("unhandled event type",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}
}

// EventTypes returns the event types this handler processes
func (h *MediaEventHandler) EventTypes() []string {
	return []string{
		"MediaCreated",
		"MediaFileUpdated",
	}
}

// handleMediaCreated starts the media processing saga
func (h *MediaEventHandler) handleMediaCreated(ctx context.Context, event *media.MediaCreated) error {
	h.logger.Info("handling media created event",
		zap.String("media_id", event.AggregateID().String()),
		zap.String("media_type", event.MediaType),
		zap.String("title", event.Title),
	)

	// Check if download URL is provided
	if event.DownloadURL == "" {
		h.logger.Debug("no download URL provided, skipping saga",
			zap.String("media_id", event.AggregateID().String()),
		)
		return nil
	}

	// Prepare saga data
	sagaData := map[string]interface{}{
		"media_id":          event.AggregateID().String(),
		"media_type":        event.MediaType,
		"title":             event.Title,
		"download_url":      event.DownloadURL,
		"target_path":       fmt.Sprintf("/media/downloads/%s", event.AggregateID().String()),
		"output_path":       fmt.Sprintf("/media/hls/%s", event.AggregateID().String()),
		"transcode_profile": "hls_1080p", // Default profile
	}

	// Start media processing saga
	saga, err := h.orchestrator.StartSaga(ctx, "MediaProcessing", sagaData)
	if err != nil {
		return fmt.Errorf("failed to start media processing saga: %w", err)
	}

	h.logger.Info("media processing saga started",
		zap.String("saga_id", saga.ID),
		zap.String("media_id", event.AggregateID().String()),
	)

	return nil
}

// handleMediaFileUpdated handles media file updates
func (h *MediaEventHandler) handleMediaFileUpdated(ctx context.Context, event *media.MediaFileUpdated) error {
	h.logger.Info("handling media file updated event",
		zap.String("media_id", event.AggregateID().String()),
		zap.String("file_path", event.FilePath),
	)

	// Could trigger additional processing here if needed
	// For example, thumbnail generation, metadata extraction, etc.

	return nil
}

// TranscodeEventHandler handles transcode completion events
type TranscodeEventHandler struct {
	mediaService media.Service
	logger       *zap.Logger
}

// NewTranscodeEventHandler creates a new transcode event handler
func NewTranscodeEventHandler(mediaService media.Service, logger *zap.Logger) *TranscodeEventHandler {
	return &TranscodeEventHandler{
		mediaService: mediaService,
		logger:       logger.Named("transcode-event-handler"),
	}
}

// Handle processes transcode events
func (h *TranscodeEventHandler) Handle(ctx context.Context, event domainevents.Event) error {
	// This would handle events from the transcode service
	// For example: TranscodeCompleted, TranscodeFailed
	
	h.logger.Info("handling transcode event",
		zap.String("event_type", event.EventType()),
		zap.String("aggregate_id", event.AggregateID().String()),
	)

	return nil
}

// EventTypes returns the event types this handler processes
func (h *TranscodeEventHandler) EventTypes() []string {
	return []string{
		"TranscodeCompleted",
		"TranscodeFailed",
	}
}