package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// IntegrationEvent represents an event that crosses bounded context boundaries
// These events are published asynchronously for other services to consume
type IntegrationEvent interface {
	Event
	// CorrelationID returns the correlation ID for tracing across services
	CorrelationID() string
	// CausationID returns the ID of the event that caused this event
	CausationID() string
	// PublishedAt returns when this event was published
	PublishedAt() *time.Time
}

// BaseIntegrationEvent provides common functionality for integration events
type BaseIntegrationEvent struct {
	BaseEvent
	correlationID string
	causationID   string
	publishedAt   *time.Time
}

// NewBaseIntegrationEvent creates a new base integration event
func NewBaseIntegrationEvent(aggregateID uuid.UUID, aggregateType, eventType string, version int, correlationID, causationID string) BaseIntegrationEvent {
	event := BaseIntegrationEvent{
		BaseEvent:     NewBaseEvent(aggregateID, aggregateType, eventType, version),
		correlationID: correlationID,
		causationID:   causationID,
	}
	
	// Set correlation ID in metadata for backward compatibility
	event.metadata["correlation_id"] = correlationID
	event.metadata["causation_id"] = causationID
	
	return event
}

// CorrelationID returns the correlation ID
func (e BaseIntegrationEvent) CorrelationID() string {
	return e.correlationID
}

// CausationID returns the causation ID
func (e BaseIntegrationEvent) CausationID() string {
	return e.causationID
}

// PublishedAt returns when this event was published
func (e BaseIntegrationEvent) PublishedAt() *time.Time {
	return e.publishedAt
}

// MarkAsPublished marks the event as published
func (e *BaseIntegrationEvent) MarkAsPublished() {
	now := time.Now()
	e.publishedAt = &now
}

// IntegrationEventPublisher publishes integration events to external message broker
type IntegrationEventPublisher interface {
	// PublishIntegrationEvent publishes an event for other services
	PublishIntegrationEvent(ctx context.Context, event IntegrationEvent) error
}

// IntegrationEventHandler handles integration events from other services
type IntegrationEventHandler interface {
	EventHandler
	// SupportedEventTypes returns the event types this handler can process
	SupportedEventTypes() []string
}

// EventEnvelope wraps an integration event with transport metadata
type EventEnvelope struct {
	EventID       uuid.UUID              `json:"event_id"`
	EventType     string                 `json:"event_type"`
	AggregateID   uuid.UUID              `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	CorrelationID string                 `json:"correlation_id"`
	CausationID   string                 `json:"causation_id"`
	Timestamp     time.Time              `json:"timestamp"`
	PublishedAt   *time.Time             `json:"published_at,omitempty"`
	Data          interface{}            `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ToEnvelope converts an integration event to an envelope for transport
func ToEnvelope(event IntegrationEvent) *EventEnvelope {
	return &EventEnvelope{
		EventID:       event.ID(),
		EventType:     event.EventType(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		Version:       event.Version(),
		CorrelationID: event.CorrelationID(),
		CausationID:   event.CausationID(),
		Timestamp:     event.CreatedAt(),
		PublishedAt:   event.PublishedAt(),
		Data:          event,
		Metadata:      event.Metadata(),
	}
}