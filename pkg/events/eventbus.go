package events

import (
	"context"
	"sync"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// InMemoryEventBus is an in-memory implementation of EventBus
type InMemoryEventBus struct {
	handlers map[string][]interfaces.EventHandler
	mu       sync.RWMutex
	logger   interfaces.Logger
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// LocalEventBus is an alias for InMemoryEventBus
type LocalEventBus = InMemoryEventBus

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(logger interfaces.Logger) *InMemoryEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &InMemoryEventBus{
		handlers: make(map[string][]interfaces.EventHandler),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// NewLocalEventBus creates a new local event bus (alias for NewInMemoryEventBus)
func NewLocalEventBus(logger interfaces.Logger) *LocalEventBus {
	return NewInMemoryEventBus(logger)
}

// Publish publishes an event to all subscribers
func (eb *InMemoryEventBus) Publish(ctx context.Context, event interfaces.Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.EventType()]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			eb.logger.Error("Event handler failed",
				interfaces.String("event_type", event.EventType()),
				interfaces.String("handler", handler.EventType()),
				interfaces.Error(err))
			// Continue processing other handlers
		}
	}

	return nil
}

// PublishAsync publishes an event asynchronously
func (eb *InMemoryEventBus) PublishAsync(ctx context.Context, event interfaces.Event) {
	eb.wg.Add(1)
	go func() {
		defer eb.wg.Done()
		if err := eb.Publish(ctx, event); err != nil {
			eb.logger.Error("Async event publish failed",
				interfaces.String("event_type", event.EventType()),
				interfaces.Error(err))
		}
	}()
}

// Subscribe registers a handler for a specific event type
func (eb *InMemoryEventBus) Subscribe(eventType string, handler interfaces.EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.logger.Debug("Event handler subscribed",
		interfaces.String("event_type", eventType),
		interfaces.String("handler", handler.EventType()))

	return nil
}

// Unsubscribe removes a handler for a specific event type
func (eb *InMemoryEventBus) Unsubscribe(eventType string, handler interfaces.EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if h == handler {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// Start starts the event bus
func (eb *InMemoryEventBus) Start(ctx context.Context) error {
	eb.logger.Info("Event bus started")
	return nil
}

// Stop stops the event bus
func (eb *InMemoryEventBus) Stop() error {
	eb.cancel()
	eb.wg.Wait()
	eb.logger.Info("Event bus stopped")
	return nil
}
