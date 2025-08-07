package interfaces

import (
	"context"
)

// Event represents a domain event.
type Event interface {
	// EventType returns the type of the event
	EventType() string

	// Timestamp returns when the event occurred
	Timestamp() int64

	// AggregateID returns the ID of the aggregate that produced the event
	AggregateID() string
}

// EventHandler handles events of a specific type.
type EventHandler interface {
	// Handle processes an event
	Handle(ctx context.Context, event Event) error

	// EventType returns the type of events this handler processes
	EventType() string
}

// EventBus provides pub/sub functionality for domain events.
type EventBus interface {
	// Publish publishes an event to all subscribers
	Publish(ctx context.Context, event Event) error

	// PublishAsync publishes an event asynchronously
	PublishAsync(ctx context.Context, event Event)

	// Subscribe registers a handler for a specific event type
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType string, handler EventHandler) error

	// Start starts the event bus
	Start(ctx context.Context) error

	// Stop stops the event bus
	Stop() error
}

// EventStore provides persistence for events.
type EventStore interface {
	// Save saves an event to the store
	Save(ctx context.Context, event Event) error

	// Load loads events for an aggregate
	Load(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)

	// LoadAll loads all events of a specific type
	LoadAll(ctx context.Context, eventType string, limit, offset int) ([]Event, error)
}
