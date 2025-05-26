package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	// Publish publishes an event to the specified subject
	Publish(ctx context.Context, subject string, event interface{}) error
	
	// Subscribe subscribes to events on the specified subject
	Subscribe(ctx context.Context, subject string, handler func([]byte) error) error
	
	// Close closes the event bus connection
	Close() error
}

// NATSEventBus implements EventBus using NATS JetStream
type NATSEventBus struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	logger *zap.Logger
}

// NewNATSEventBus creates a new NATS event bus
func NewNATSEventBus(url string, logger *zap.Logger) (*NATSEventBus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create streams for our event types
	streams := []string{
		"media.downloaded",
		"media.transcoded",
		"media.added",
		"media.removed",
		"transcoding.progress",
		"download.progress",
	}

	for _, stream := range streams {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     stream,
			Subjects: []string{stream + ".>"},
			Storage:  nats.FileStorage,
			Retention: nats.WorkQueuePolicy,
		})
		if err != nil && err != nats.ErrStreamNameAlreadyInUse {
			nc.Close()
			return nil, fmt.Errorf("failed to create stream %s: %w", stream, err)
		}
	}

	return &NATSEventBus{
		nc:     nc,
		js:     js,
		logger: logger,
	}, nil
}

// Publish publishes an event to the specified subject
func (b *NATSEventBus) Publish(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = b.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	b.logger.Debug("published event",
		zap.String("subject", subject),
		zap.Any("event", event))
	return nil
}

// Subscribe subscribes to events on the specified subject
func (b *NATSEventBus) Subscribe(ctx context.Context, subject string, handler func([]byte) error) error {
	sub, err := b.js.Subscribe(subject, func(msg *nats.Msg) {
		err := handler(msg.Data)
		if err != nil {
			b.logger.Error("failed to handle event",
				zap.String("subject", subject),
				zap.Error(err))
			// Don't ack the message so it can be retried
			return
		}
		msg.Ack()
	}, nats.AckWait(5*time.Second))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start a goroutine to handle context cancellation
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	return nil
}

// Close closes the NATS connection
func (b *NATSEventBus) Close() error {
	b.nc.Close()
	return nil
} 