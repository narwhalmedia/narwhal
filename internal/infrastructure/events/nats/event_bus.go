package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events"
)

// NATSEventBus implements EventBus using NATS JetStream
type NATSEventBus struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	stream string
}

// NewNATSEventBus creates a new NATS-based event bus
func NewNATSEventBus(url, stream string) (events.EventBus, error) {
	// Connect to NATS
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	
	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}
	
	// Create or update stream
	streamConfig := &nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{stream + ".*"},
		Storage:   nats.FileStorage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    7 * 24 * time.Hour, // Keep events for 7 days
		Discard:   nats.DiscardOld,
		Replicas:  1,
	}
	
	_, err = js.AddStream(streamConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create stream: %w", err)
	}
	
	return &NATSEventBus{
		conn:   conn,
		js:     js,
		stream: stream,
	}, nil
}

// Publish publishes an event to NATS
func (eb *NATSEventBus) Publish(ctx context.Context, topic string, data []byte) error {
	subject := fmt.Sprintf("%s.%s", eb.stream, topic)
	
	// Publish with JetStream for at-least-once delivery
	pubAck, err := eb.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("publish to NATS: %w", err)
	}
	
	// Wait for acknowledgment from JetStream
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for publish acknowledgment")
	default:
		if pubAck.Sequence == 0 {
			return fmt.Errorf("invalid publish acknowledgment")
		}
	}
	
	return nil
}

// Subscribe subscribes to events from NATS
func (eb *NATSEventBus) Subscribe(ctx context.Context, topic string, handler events.EventHandler) error {
	subject := fmt.Sprintf("%s.%s", eb.stream, topic)
	
	// Create durable consumer
	consumerName := fmt.Sprintf("consumer-%s", topic)
	
	// Subscribe with JetStream
	sub, err := eb.js.Subscribe(subject, func(msg *nats.Msg) {
		// Handle message in goroutine to not block
		go func() {
			// Create context with timeout for handler
			handlerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			// Call handler
			if err := handler(handlerCtx, msg.Data); err != nil {
				// Log error (in production, use proper logging)
				fmt.Printf("error handling message: %v\n", err)
				// Negative acknowledgment - message will be redelivered
				msg.Nak()
				return
			}
			
			// Acknowledge successful processing
			msg.Ack()
		}()
	}, nats.Durable(consumerName), nats.ManualAck())
	
	if err != nil {
		return fmt.Errorf("subscribe to NATS: %w", err)
	}
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Unsubscribe
	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("unsubscribe from NATS: %w", err)
	}
	
	return nil
}

// Close closes the NATS connection
func (eb *NATSEventBus) Close() error {
	eb.conn.Close()
	return nil
}