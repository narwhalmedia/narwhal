package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/narwhalmedia/narwhal/internal/config"
	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
)

func TestPublisher_PublishEvent(t *testing.T) {
	// Skip if NATS is not available
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:           "nats://localhost:4222",
			ClientID:      "test-publisher",
			DurableName:   "test-durable",
			MaxReconnect:  5,
			ReconnectWait: 1 * time.Second,
		},
	}

	logger := zaptest.NewLogger(t)

	// Create client
	client, cleanup, err := nats.NewClient(cfg, logger)
	if err != nil {
		t.Skip("NATS not available:", err)
	}
	defer cleanup()

	// Create publisher
	publisher := nats.NewPublisher(client, logger)

	// Create test event
	event := &media.SeriesCreated{
		BaseEvent: domainevents.BaseEvent{
			EventID:       uuid.New(),
			AggregateID:   uuid.New(),
			AggregateType: "Series",
			EventType:     "SeriesCreated",
			EventVersion:  1,
			OccurredAt:    time.Now(),
		},
		Title:       "Test Series",
		Description: "Test Description",
	}

	// Publish event
	ctx := context.Background()
	err = publisher.PublishEvent(ctx, event)
	require.NoError(t, err)
}

func TestConsumerGroup_HandleEvents(t *testing.T) {
	// Skip if NATS is not available
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:           "nats://localhost:4222",
			ClientID:      "test-consumer",
			DurableName:   "test-durable",
			MaxReconnect:  5,
			ReconnectWait: 1 * time.Second,
		},
	}

	logger := zaptest.NewLogger(t)

	// Create client
	client, cleanup, err := nats.NewClient(cfg, logger)
	if err != nil {
		t.Skip("NATS not available:", err)
	}
	defer cleanup()

	// Create publisher
	publisher := nats.NewPublisher(client, logger)

	// Create consumer
	consumerConfig := nats.ConsumerConfig{
		Name:       "test-consumer",
		MaxRetries: 1,
		AckWait:    5 * time.Second,
		MaxDeliver: 2,
	}
	consumer := nats.NewConsumerGroup(client, consumerConfig, logger)

	// Track handled events
	handledEvents := make(chan domainevents.Event, 1)

	// Register test handler
	handler := &testEventHandler{
		handleFunc: func(ctx context.Context, event domainevents.Event) error {
			handledEvents <- event
			return nil
		},
		eventTypes: []string{"SeriesCreated"},
	}
	consumer.RegisterHandler(handler)

	// Start consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := consumer.Start(ctx)
		assert.NoError(t, err)
	}()

	// Give consumer time to start
	time.Sleep(1 * time.Second)

	// Publish test event
	event := &media.SeriesCreated{
		BaseEvent: domainevents.BaseEvent{
			EventID:       uuid.New(),
			AggregateID:   uuid.New(),
			AggregateType: "Series",
			EventType:     "SeriesCreated",
			EventVersion:  1,
			OccurredAt:    time.Now(),
		},
		Title:       "Test Series",
		Description: "Test Description",
	}

	err = publisher.PublishEvent(ctx, event)
	require.NoError(t, err)

	// Wait for event to be handled
	select {
	case handled := <-handledEvents:
		assert.Equal(t, event.ID(), handled.ID())
		assert.Equal(t, event.EventType(), handled.EventType())
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event to be handled")
	}
}

// testEventHandler is a test implementation of EventHandler
type testEventHandler struct {
	handleFunc func(context.Context, domainevents.Event) error
	eventTypes []string
}

func (h *testEventHandler) Handle(ctx context.Context, event domainevents.Event) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, event)
	}
	return nil
}

func (h *testEventHandler) EventTypes() []string {
	return h.eventTypes
}