package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Publisher implements the EventPublisher interface using NATS JetStream
type Publisher struct {
	client *Client
	logger *zap.Logger
}

// NewPublisher creates a new NATS event publisher
func NewPublisher(client *Client, logger *zap.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger.Named("publisher"),
	}
}

// PublishEvent publishes a domain event to NATS
func (p *Publisher) PublishEvent(ctx context.Context, event domainevents.Event) error {
	// Determine the subject based on event type
	subject := p.getSubjectForEvent(event)
	
	// Create event envelope with metadata
	envelope := EventEnvelope{
		ID:            event.ID().String(),
		AggregateID:   event.AggregateID().String(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		EventVersion:  event.Version(),
		OccurredAt:    event.CreatedAt(),
		Data:          event,
	}
	
	// Marshal event to JSON
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// Publish to JetStream with deduplication ID
	pubOpts := []jetstream.PublishOpt{
		jetstream.WithMsgID(event.ID().String()), // Deduplication
		jetstream.WithExpectLastMsgID(""),        // No specific expectation
	}
	
	// Publish with timeout
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	ack, err := p.client.JetStream().Publish(pubCtx, subject, data, pubOpts...)
	
	if err != nil {
		p.logger.Error("failed to publish event",
			zap.Error(err),
			zap.String("event_id", event.ID().String()),
			zap.String("event_type", event.EventType()),
			zap.String("subject", subject),
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	p.logger.Info("event published",
		zap.String("event_id", event.ID().String()),
		zap.String("event_type", event.EventType()),
		zap.String("subject", subject),
		zap.Uint64("sequence", ack.Sequence),
		zap.String("stream", ack.Stream),
	)
	
	return nil
}

// getSubjectForEvent determines the NATS subject for an event
func (p *Publisher) getSubjectForEvent(event domainevents.Event) string {
	// Map aggregate types to subject prefixes
	switch event.AggregateType() {
	case "Series", "Episode", "Movie":
		return fmt.Sprintf("media.%s.%s", event.AggregateType(), event.EventType())
	case "Download":
		return fmt.Sprintf("download.%s", event.EventType())
	case "TranscodeJob":
		return fmt.Sprintf("transcode.%s", event.EventType())
	default:
		// Default to aggregate type as prefix
		return fmt.Sprintf("%s.%s", event.AggregateType(), event.EventType())
	}
}

// EventEnvelope wraps an event with metadata for transport
type EventEnvelope struct {
	ID            string              `json:"id"`
	AggregateID   string              `json:"aggregate_id"`
	AggregateType string              `json:"aggregate_type"`
	EventType     string              `json:"event_type"`
	EventVersion  int                 `json:"event_version"`
	OccurredAt    time.Time           `json:"occurred_at"`
	Data          domainevents.Event  `json:"data"`
}

// getCorrelationID extracts correlation ID from context
func getCorrelationID(ctx context.Context) string {
	if val := ctx.Value("correlation_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	if val := ctx.Value("request_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}