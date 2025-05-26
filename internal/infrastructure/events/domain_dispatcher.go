package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// InMemoryDomainEventDispatcher dispatches domain events synchronously in-process
type InMemoryDomainEventDispatcher struct {
	handlers map[string][]events.DomainEventHandler
	mu       sync.RWMutex
}

// NewInMemoryDomainEventDispatcher creates a new in-memory domain event dispatcher
func NewInMemoryDomainEventDispatcher() events.DomainEventDispatcher {
	return &InMemoryDomainEventDispatcher{
		handlers: make(map[string][]events.DomainEventHandler),
	}
}

// RegisterHandler registers a handler for specific event types
func (d *InMemoryDomainEventDispatcher) RegisterHandler(handler events.DomainEventHandler, eventTypes ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for _, eventType := range eventTypes {
		d.handlers[eventType] = append(d.handlers[eventType], handler)
	}
}

// Dispatch synchronously dispatches an event to all registered handlers
func (d *InMemoryDomainEventDispatcher) Dispatch(ctx context.Context, event events.DomainEvent) error {
	d.mu.RLock()
	handlers, exists := d.handlers[event.EventType()]
	d.mu.RUnlock()
	
	if !exists || len(handlers) == 0 {
		// No handlers registered for this event type
		return nil
	}
	
	// Execute all handlers synchronously
	for _, handler := range handlers {
		if handler.CanHandle(event.EventType()) {
			if err := handler.HandleDomainEvent(ctx, event); err != nil {
				// Stop on first error to maintain transactional consistency
				return fmt.Errorf("handler error for event %s: %w", event.EventType(), err)
			}
		}
	}
	
	return nil
}

// ExampleDomainEventHandler is an example implementation of a domain event handler
type ExampleDomainEventHandler struct {
	eventTypes map[string]bool
	handlerFn  func(ctx context.Context, event events.DomainEvent) error
}

// NewExampleDomainEventHandler creates a new example handler
func NewExampleDomainEventHandler(handlerFn func(ctx context.Context, event events.DomainEvent) error, eventTypes ...string) events.DomainEventHandler {
	eventTypeMap := make(map[string]bool)
	for _, et := range eventTypes {
		eventTypeMap[et] = true
	}
	
	return &ExampleDomainEventHandler{
		eventTypes: eventTypeMap,
		handlerFn:  handlerFn,
	}
}

// HandleDomainEvent processes a domain event
func (h *ExampleDomainEventHandler) HandleDomainEvent(ctx context.Context, event events.DomainEvent) error {
	return h.handlerFn(ctx, event)
}

// CanHandle returns true if this handler can process the given event type
func (h *ExampleDomainEventHandler) CanHandle(eventType string) bool {
	return h.eventTypes[eventType]
}