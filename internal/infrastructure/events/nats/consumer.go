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

// EventHandler processes events
type EventHandler interface {
	Handle(ctx context.Context, event domainevents.Event) error
	EventTypes() []string
}

// ConsumerGroup manages event consumption
type ConsumerGroup struct {
	client        *Client
	logger        *zap.Logger
	handlers      map[string][]EventHandler
	consumerName  string
	maxRetries    int
	ackWait       time.Duration
	maxDeliver    int
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Name         string
	MaxRetries   int
	AckWait      time.Duration
	MaxDeliver   int
	MaxAckPending int
}

// NewConsumerGroup creates a new consumer group
func NewConsumerGroup(client *Client, config ConsumerConfig, logger *zap.Logger) *ConsumerGroup {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.AckWait == 0 {
		config.AckWait = 30 * time.Second
	}
	if config.MaxDeliver == 0 {
		config.MaxDeliver = 5
	}

	return &ConsumerGroup{
		client:       client,
		logger:       logger.Named("consumer"),
		handlers:     make(map[string][]EventHandler),
		consumerName: config.Name,
		maxRetries:   config.MaxRetries,
		ackWait:      config.AckWait,
		maxDeliver:   config.MaxDeliver,
	}
}

// RegisterHandler registers an event handler
func (c *ConsumerGroup) RegisterHandler(handler EventHandler) {
	for _, eventType := range handler.EventTypes() {
		c.handlers[eventType] = append(c.handlers[eventType], handler)
		c.logger.Info("registered event handler",
			zap.String("event_type", eventType),
			zap.String("handler", fmt.Sprintf("%T", handler)),
		)
	}
}

// Start begins consuming events
func (c *ConsumerGroup) Start(ctx context.Context) error {
	// Create consumers for each stream
	streams := []string{"MEDIA_EVENTS", "DOWNLOAD_EVENTS", "TRANSCODE_EVENTS", "SAGA_EVENTS"}
	
	for _, streamName := range streams {
		if err := c.createConsumer(ctx, streamName); err != nil {
			return fmt.Errorf("failed to create consumer for stream %s: %w", streamName, err)
		}
	}

	c.logger.Info("consumer group started",
		zap.String("consumer", c.consumerName),
		zap.Int("handlers", len(c.handlers)),
	)

	// Keep running until context is cancelled
	<-ctx.Done()
	c.logger.Info("consumer group stopping")
	
	return nil
}

// createConsumer creates a durable consumer for a stream
func (c *ConsumerGroup) createConsumer(ctx context.Context, streamName string) error {
	// Consumer configuration
	consumerConfig := jetstream.ConsumerConfig{
		Name:              fmt.Sprintf("%s-%s", c.consumerName, streamName),
		Durable:           c.consumerName,
		Description:       fmt.Sprintf("Consumer for %s", c.consumerName),
		AckPolicy:         jetstream.AckExplicitPolicy,
		AckWait:           c.ackWait,
		MaxDeliver:        c.maxDeliver,
		ReplayPolicy:      jetstream.ReplayInstantPolicy,
		MaxAckPending:     100,
		DeliverPolicy:     jetstream.DeliverAllPolicy,
		FilterSubject:     "",  // Subscribe to all subjects in stream
	}

	// Create or update consumer
	consumer, err := c.client.JetStream().CreateOrUpdateConsumer(ctx, streamName, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Start consuming messages
	go c.consumeMessages(ctx, consumer, streamName)

	return nil
}

// consumeMessages processes messages from a consumer
func (c *ConsumerGroup) consumeMessages(ctx context.Context, consumer jetstream.Consumer, streamName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Fetch messages with timeout
			_, cancel := context.WithTimeout(ctx, 5*time.Second)
			msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
			cancel()

			if err != nil {
				if err != context.DeadlineExceeded {
					c.logger.Error("failed to fetch messages",
						zap.Error(err),
						zap.String("stream", streamName),
					)
					time.Sleep(1 * time.Second)
				}
				continue
			}

			for msg := range msgs.Messages() {
				c.processMessage(ctx, msg)
			}
		}
	}
}

// processMessage handles a single message
func (c *ConsumerGroup) processMessage(ctx context.Context, msg jetstream.Msg) {
	// Extract correlation ID from headers
	correlationID := ""
	if headers := msg.Headers(); headers != nil {
		if vals := headers.Values("X-Correlation-ID"); len(vals) > 0 {
			correlationID = vals[0]
		}
	}

	// Add correlation ID to context
	if correlationID != "" {
		ctx = context.WithValue(ctx, "correlation_id", correlationID)
	}

	// Parse event envelope
	var envelope EventEnvelope
	if err := json.Unmarshal(msg.Data(), &envelope); err != nil {
		c.logger.Error("failed to unmarshal event",
			zap.Error(err),
			zap.String("subject", msg.Subject()),
		)
		c.handleMessageError(ctx, msg, err)
		return
	}

	// Get handlers for event type
	handlers, ok := c.handlers[envelope.EventType]
	if !ok {
		// No handlers for this event type, acknowledge and continue
		c.logger.Debug("no handlers for event type",
			zap.String("event_type", envelope.EventType),
		)
		msg.Ack()
		return
	}

	// Process with each handler
	for _, handler := range handlers {
		if err := c.processWithHandler(ctx, handler, envelope.Data); err != nil {
			c.logger.Error("handler failed",
				zap.Error(err),
				zap.String("event_id", envelope.ID),
				zap.String("event_type", envelope.EventType),
				zap.String("handler", fmt.Sprintf("%T", handler)),
			)
			c.handleMessageError(ctx, msg, err)
			return
		}
	}

	// All handlers succeeded, acknowledge message
	if err := msg.Ack(); err != nil {
		c.logger.Error("failed to acknowledge message",
			zap.Error(err),
			zap.String("event_id", envelope.ID),
		)
	}
}

// processWithHandler executes a handler with retry logic
func (c *ConsumerGroup) processWithHandler(ctx context.Context, handler EventHandler, event domainevents.Event) error {
	var lastErr error
	
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			c.logger.Info("retrying handler",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		// Execute handler
		if err := handler.Handle(ctx, event); err != nil {
			lastErr = err
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("handler failed after %d attempts: %w", c.maxRetries, lastErr)
}

// handleMessageError handles message processing errors
func (c *ConsumerGroup) handleMessageError(ctx context.Context, msg jetstream.Msg, err error) {
	metadata, _ := msg.Metadata()
	
	// Check if message has exceeded max deliveries
	if metadata != nil && metadata.NumDelivered >= uint64(c.maxDeliver) {
		// Send to dead letter queue
		c.sendToDeadLetterQueue(ctx, msg, err)
		
		// Acknowledge to remove from stream
		msg.Ack()
	} else {
		// Negative acknowledgment for retry
		msg.Nak()
	}
}

// sendToDeadLetterQueue sends failed messages to DLQ
func (c *ConsumerGroup) sendToDeadLetterQueue(ctx context.Context, msg jetstream.Msg, originalErr error) {
	metadata, _ := msg.Metadata()
	
	dlqMessage := DeadLetterMessage{
		OriginalSubject: msg.Subject(),
		OriginalData:    msg.Data(),
		Error:           originalErr.Error(),
		Timestamp:       time.Now(),
		NumDelivered:    0,
		Consumer:        c.consumerName,
	}

	if metadata != nil {
		dlqMessage.NumDelivered = metadata.NumDelivered
		dlqMessage.Stream = metadata.Stream
		dlqMessage.Consumer = metadata.Consumer
	}

	data, err := json.Marshal(dlqMessage)
	if err != nil {
		c.logger.Error("failed to marshal DLQ message",
			zap.Error(err),
		)
		return
	}

	// Publish to DLQ
	subject := fmt.Sprintf("dlq.%s", c.consumerName)
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := c.client.JetStream().Publish(pubCtx, subject, data); err != nil {
		c.logger.Error("failed to send message to DLQ",
			zap.Error(err),
			zap.String("subject", subject),
		)
	} else {
		c.logger.Warn("message sent to dead letter queue",
			zap.String("original_subject", msg.Subject()),
			zap.String("error", originalErr.Error()),
			zap.Uint64("deliveries", dlqMessage.NumDelivered),
		)
	}
}

// DeadLetterMessage represents a message in the dead letter queue
type DeadLetterMessage struct {
	OriginalSubject string    `json:"original_subject"`
	OriginalData    []byte    `json:"original_data"`
	Error           string    `json:"error"`
	Timestamp       time.Time `json:"timestamp"`
	NumDelivered    uint64    `json:"num_delivered"`
	Stream          string    `json:"stream"`
	Consumer        string    `json:"consumer"`
}