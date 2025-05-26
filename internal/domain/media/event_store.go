package media

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventStore defines the interface for storing and retrieving domain events
type EventStore interface {
	// Save saves a domain event
	Save(ctx context.Context, event Event) error
	// GetEvents retrieves all events for an aggregate
	GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]Event, error)
	// GetEventsByType retrieves events of a specific type
	GetEventsByType(ctx context.Context, eventType string) ([]Event, error)
	// GetEventsByTimeRange retrieves events within a time range
	GetEventsByTimeRange(ctx context.Context, start, end time.Time) ([]Event, error)
}

// EventStoreError represents an error that occurred in the event store
type EventStoreError struct {
	Op  string
	Err error
}

// Error returns the string representation of the error
func (e *EventStoreError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

// Unwrap returns the underlying error
func (e *EventStoreError) Unwrap() error {
	return e.Err
}

// NewEventStoreError creates a new event store error
func NewEventStoreError(op string, err error) error {
	return &EventStoreError{
		Op:  op,
		Err: err,
	}
} 