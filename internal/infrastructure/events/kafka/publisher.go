package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Publisher implements events.EventPublisher
type Publisher struct {
	producer sarama.SyncProducer
	topic    string
}

// NewPublisher creates a new Kafka event publisher
func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("creating producer: %w", err)
	}

	return &Publisher{
		producer: producer,
		topic:    topic,
	}, nil
}

// PublishEvent publishes an event to Kafka
func (p *Publisher) PublishEvent(ctx context.Context, event events.Event) error {
	// Create message
	message := &events.Message{
		ID:            event.ID(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Data:          event,
		Metadata:      event.Metadata(),
		CreatedAt:     event.CreatedAt(),
	}

	// Marshal message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	// Create Kafka message
	kafkaMsg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.AggregateID().String()),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_type"),
				Value: []byte(event.EventType()),
			},
			{
				Key:   []byte("aggregate_type"),
				Value: []byte(event.AggregateType()),
			},
		},
	}

	// Send message
	_, _, err = p.producer.SendMessage(kafkaMsg)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

// Close closes the publisher
func (p *Publisher) Close() error {
	return p.producer.Close()
} 