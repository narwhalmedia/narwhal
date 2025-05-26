package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// EventHandler is a function that handles an event
type EventHandler func(ctx context.Context, event *Event) error

// EventManager manages event subscriptions and publishing
type EventManager struct {
	bus     EventBus
	logger  *zap.Logger
	handlers map[EventType][]EventHandler
	mu      sync.RWMutex
}

// NewEventManager creates a new event manager
func NewEventManager(bus EventBus, logger *zap.Logger) *EventManager {
	return &EventManager{
		bus:     bus,
		logger:  logger,
		handlers: make(map[EventType][]EventHandler),
	}
}

// Subscribe subscribes to events of the given type
func (m *EventManager) Subscribe(ctx context.Context, eventType EventType, handler EventHandler) error {
	m.mu.Lock()
	m.handlers[eventType] = append(m.handlers[eventType], handler)
	m.mu.Unlock()

	return m.bus.Subscribe(ctx, string(eventType), func(data []byte) error {
		var event Event
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal event: %w", err)
		}

		m.mu.RLock()
		handlers := m.handlers[eventType]
		m.mu.RUnlock()

		for _, h := range handlers {
			if err := h(ctx, &event); err != nil {
				m.logger.Error("failed to handle event",
					zap.String("type", string(eventType)),
					zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// Publish publishes an event
func (m *EventManager) Publish(ctx context.Context, eventType EventType, data interface{}) error {
	event, err := NewEvent(eventType, data)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	data, err = json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return m.bus.Publish(ctx, event.Subject(), data)
}

// Close closes the event manager
func (m *EventManager) Close() error {
	return m.bus.Close()
} 