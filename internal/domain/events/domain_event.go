package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents an event that occurred within a bounded context
// These events are handled in-process and represent state changes within the domain
type DomainEvent interface {
	Event
	// OccurredWithin returns the bounded context where this event occurred
	OccurredWithin() string
}

// BaseDomainEvent provides common functionality for domain events
type BaseDomainEvent struct {
	BaseEvent
	boundedContext string
}

// NewBaseDomainEvent creates a new base domain event
func NewBaseDomainEvent(aggregateID uuid.UUID, aggregateType, eventType string, version int, boundedContext string) BaseDomainEvent {
	return BaseDomainEvent{
		BaseEvent:      NewBaseEvent(aggregateID, aggregateType, eventType, version),
		boundedContext: boundedContext,
	}
}

// OccurredWithin returns the bounded context where this event occurred
func (e BaseDomainEvent) OccurredWithin() string {
	return e.boundedContext
}

// DomainEventHandler handles domain events within the same bounded context
type DomainEventHandler interface {
	// HandleDomainEvent processes a domain event synchronously
	HandleDomainEvent(ctx context.Context, event DomainEvent) error
	// CanHandle returns true if this handler can process the given event type
	CanHandle(eventType string) bool
}

// DomainEventDispatcher dispatches domain events to registered handlers
type DomainEventDispatcher interface {
	// RegisterHandler registers a handler for specific event types
	RegisterHandler(handler DomainEventHandler, eventTypes ...string)
	// Dispatch synchronously dispatches an event to all registered handlers
	Dispatch(ctx context.Context, event DomainEvent) error
}

// domainEventDispatcher is the default implementation
type domainEventDispatcher struct {
	handlers map[string][]DomainEventHandler
}

// NewDomainEventDispatcher creates a new domain event dispatcher
func NewDomainEventDispatcher() DomainEventDispatcher {
	return &domainEventDispatcher{
		handlers: make(map[string][]DomainEventHandler),
	}
}

func (d *domainEventDispatcher) RegisterHandler(handler DomainEventHandler, eventTypes ...string) {
	for _, eventType := range eventTypes {
		d.handlers[eventType] = append(d.handlers[eventType], handler)
	}
}

func (d *domainEventDispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	handlers, exists := d.handlers[event.EventType()]
	if !exists {
		return nil // No handlers registered
	}

	// Execute all handlers synchronously
	for _, handler := range handlers {
		if handler.CanHandle(event.EventType()) {
			if err := handler.HandleDomainEvent(ctx, event); err != nil {
				return err // Stop on first error
			}
		}
	}

	return nil
}