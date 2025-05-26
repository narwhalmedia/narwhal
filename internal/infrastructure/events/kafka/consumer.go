package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Consumer implements events.EventConsumer
type Consumer struct {
	consumer sarama.ConsumerGroup
	topic    string
	handlers map[string]events.EventHandler
	mu       sync.RWMutex
}

// NewConsumer creates a new Kafka event consumer
func NewConsumer(brokers []string, groupID, topic string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("creating consumer group: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		topic:    topic,
		handlers: make(map[string]events.EventHandler),
	}, nil
}

// RegisterHandler registers an event handler
func (c *Consumer) RegisterHandler(eventType string, handler events.EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

// Start starts consuming events
func (c *Consumer) Start(ctx context.Context) error {
	topics := []string{c.topic}

	for {
		err := c.consumer.Consume(ctx, topics, c)
		if err != nil {
			return fmt.Errorf("consuming messages: %w", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// ConsumeClaim implements sarama.ConsumerGroupHandler
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Parse message
		var eventMsg events.Message
		if err := json.Unmarshal(message.Value, &eventMsg); err != nil {
			return fmt.Errorf("unmarshaling message: %w", err)
		}

		// Get handler
		c.mu.RLock()
		handler, ok := c.handlers[eventMsg.EventType]
		c.mu.RUnlock()

		if !ok {
			// Skip unknown event types
			session.MarkMessage(message, "")
			continue
		}

		// Handle event
		if err := handler.HandleEvent(context.Background(), eventMsg); err != nil {
			return fmt.Errorf("handling event: %w", err)
		}

		// Mark message as processed
		session.MarkMessage(message, "")
	}

	return nil
}

// Setup implements sarama.ConsumerGroupHandler
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup implements sarama.ConsumerGroupHandler
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.consumer.Close()
} 