package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// IntegrationEventPublisher is an adapter that publishes integration events
type IntegrationEventPublisher struct {
	eventBus EventBus
}

// NewIntegrationEventPublisher creates a new integration event publisher
func NewIntegrationEventPublisher(eventBus EventBus) events.IntegrationEventPublisher {
	return &IntegrationEventPublisher{
		eventBus: eventBus,
	}
}

// PublishIntegrationEvent publishes an integration event to the event bus
func (p *IntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event events.IntegrationEvent) error {
	// Convert to event envelope
	envelope := events.ToEnvelope(event)
	
	// Marshal to JSON
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal integration event: %w", err)
	}
	
	// Determine topic/subject based on event type
	topic := getTopicForEvent(event.EventType())
	
	// Publish to event bus
	if err := p.eventBus.Publish(ctx, topic, data); err != nil {
		return fmt.Errorf("publish to event bus: %w", err)
	}
	
	return nil
}

// EventBus is the interface for the underlying message broker
type EventBus interface {
	Publish(ctx context.Context, topic string, data []byte) error
	Subscribe(ctx context.Context, topic string, handler EventHandler) error
	Close() error
}

// EventHandler handles incoming events
type EventHandler func(ctx context.Context, data []byte) error

// getTopicForEvent maps event types to topics/subjects
func getTopicForEvent(eventType string) string {
	// Map event types to topics
	// This could be configurable or use a more sophisticated routing
	switch eventType {
	case "media.added":
		return "media.events.added"
	case "media.removed":
		return "media.events.removed"
	case "media.transcode_requested":
		return "transcode.events.requested"
	case "media.ready_to_stream":
		return "stream.events.ready"
	case "episode.added_to_series":
		return "media.events.episode_added"
	default:
		// Default topic pattern
		return fmt.Sprintf("events.%s", eventType)
	}
}