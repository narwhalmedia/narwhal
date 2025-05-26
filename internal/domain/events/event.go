package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event
type Event interface {
	ID() uuid.UUID
	AggregateID() uuid.UUID
	AggregateType() string
	EventType() string
	Version() int
	CreatedAt() time.Time
	Metadata() map[string]interface{}
}

// BaseEvent provides common event functionality
type BaseEvent struct {
	id            uuid.UUID
	aggregateID   uuid.UUID
	aggregateType string
	eventType     string
	version       int
	createdAt     time.Time
	metadata      map[string]interface{}
}

// NewBaseEvent creates a new base event
func NewBaseEvent(aggregateID uuid.UUID, aggregateType, eventType string, version int) BaseEvent {
	return BaseEvent{
		id:            uuid.New(),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		eventType:     eventType,
		version:       version,
		createdAt:     time.Now(),
		metadata:      make(map[string]interface{}),
	}
}

// ID returns the event ID
func (e BaseEvent) ID() uuid.UUID {
	return e.id
}

// AggregateID returns the aggregate ID
func (e BaseEvent) AggregateID() uuid.UUID {
	return e.aggregateID
}

// AggregateType returns the aggregate type
func (e BaseEvent) AggregateType() string {
	return e.aggregateType
}

// EventType returns the event type
func (e BaseEvent) EventType() string {
	return e.eventType
}

// Version returns the event version
func (e BaseEvent) Version() int {
	return e.version
}

// CreatedAt returns the event creation time
func (e BaseEvent) CreatedAt() time.Time {
	return e.createdAt
}

// Metadata returns the event metadata
func (e BaseEvent) Metadata() map[string]interface{} {
	return e.metadata
}

// EventStore defines the interface for event storage
type EventStore interface {
	Save(ctx context.Context, event Event) error
	GetEvents(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]Event, error)
	GetEventsByType(ctx context.Context, eventType string) ([]Event, error)
	GetEventsByTimeRange(ctx context.Context, start, end time.Time) ([]Event, error)
}

// EventPublisher defines the interface for event publishing
type EventPublisher interface {
	PublishEvent(ctx context.Context, event Event) error
}

// EventConsumer defines the interface for event consumption
type EventConsumer interface {
	RegisterHandler(eventType string, handler EventHandler)
	Start(ctx context.Context) error
	Close() error
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	HandleEvent(ctx context.Context, message Message) error
}

// Message represents an event message
type Message struct {
	ID            uuid.UUID               `json:"id"`
	AggregateID   uuid.UUID               `json:"aggregate_id"`
	AggregateType string                  `json:"aggregate_type"`
	EventType     string                  `json:"event_type"`
	Version       int                     `json:"version"`
	Data          interface{}             `json:"data"`
	Metadata      map[string]interface{}  `json:"metadata"`
	CreatedAt     time.Time               `json:"created_at"`
}

// UnmarshalEvent unmarshals an event from JSON
func UnmarshalEvent(eventType string, data []byte) (Event, error) {
	// TODO: Implement event unmarshaling based on event type
	return nil, nil
} 