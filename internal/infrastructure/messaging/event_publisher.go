package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

// EventPublisher defines the interface for publishing integration events
type EventPublisher interface {
	// PublishEvent publishes an integration event
	PublishEvent(ctx context.Context, event IntegrationEvent) error
}

// IntegrationEvent represents an event that needs to be published to external systems
type IntegrationEvent struct {
	ID        uuid.UUID          `json:"id"`
	Type      string            `json:"type"`
	Timestamp int64             `json:"timestamp"`
	Data      json.RawMessage   `json:"data"`
	Metadata  map[string]string `json:"metadata"`
}

// NewIntegrationEvent creates a new integration event
func NewIntegrationEvent(eventType string, data interface{}, metadata map[string]string) (*IntegrationEvent, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling event data: %w", err)
	}

	return &IntegrationEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Timestamp: media.Now().Unix(),
		Data:      dataJSON,
		Metadata:  metadata,
	}, nil
}

// KafkaEventPublisher implements EventPublisher using Kafka
type KafkaEventPublisher struct {
	producer KafkaProducer
}

// NewKafkaEventPublisher creates a new Kafka event publisher
func NewKafkaEventPublisher(producer KafkaProducer) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		producer: producer,
	}
}

// PublishEvent publishes an integration event to Kafka
func (p *KafkaEventPublisher) PublishEvent(ctx context.Context, event IntegrationEvent) error {
	// Convert event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	// Publish to Kafka
	return p.producer.SendMessage(ctx, event.Type, eventJSON)
}

// KafkaProducer defines the interface for Kafka message production
type KafkaProducer interface {
	// SendMessage sends a message to a Kafka topic
	SendMessage(ctx context.Context, topic string, message []byte) error
}

// EventConverter converts domain events to integration events
type EventConverter struct {
	publisher EventPublisher
}

// NewEventConverter creates a new event converter
func NewEventConverter(publisher EventPublisher) *EventConverter {
	return &EventConverter{
		publisher: publisher,
	}
}

// ConvertAndPublish converts a domain event to an integration event and publishes it
func (c *EventConverter) ConvertAndPublish(ctx context.Context, event media.Event) error {
	var integrationEvent *IntegrationEvent
	var err error

	switch e := event.(type) {
	case *media.MovieCreated:
		integrationEvent, err = c.convertMovieCreated(e)
	case *media.SeriesCreated:
		integrationEvent, err = c.convertSeriesCreated(e)
	case *media.MediaStatusChanged:
		integrationEvent, err = c.convertMediaStatusChanged(e)
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}

	if err != nil {
		return fmt.Errorf("converting event: %w", err)
	}

	return c.publisher.PublishEvent(ctx, *integrationEvent)
}

// convertMovieCreated converts a MovieCreated domain event to an integration event
func (c *EventConverter) convertMovieCreated(event *media.MovieCreated) (*IntegrationEvent, error) {
	data := struct {
		MovieID     uuid.UUID `json:"movie_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		ReleaseDate string    `json:"release_date"`
		Genres      []string  `json:"genres"`
		Director    string    `json:"director"`
	}{
		MovieID:     event.GetAggregateID(),
		Title:       event.Title,
		Description: event.Description,
		ReleaseDate: event.ReleaseDate.Format("2006-01-02"),
		Genres:      event.Genres,
		Director:    event.Director,
	}

	return NewIntegrationEvent("movie.created", data, nil)
}

// convertSeriesCreated converts a SeriesCreated domain event to an integration event
func (c *EventConverter) convertSeriesCreated(event *media.SeriesCreated) (*IntegrationEvent, error) {
	data := struct {
		SeriesID    uuid.UUID `json:"series_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
	}{
		SeriesID:    event.GetAggregateID(),
		Title:       event.Title,
		Description: event.Description,
	}

	return NewIntegrationEvent("series.created", data, nil)
}

// convertMediaStatusChanged converts a MediaStatusChanged domain event to an integration event
func (c *EventConverter) convertMediaStatusChanged(event *media.MediaStatusChanged) (*IntegrationEvent, error) {
	data := struct {
		MediaID   uuid.UUID     `json:"media_id"`
		MediaType string        `json:"media_type"`
		OldStatus media.Status  `json:"old_status"`
		NewStatus media.Status  `json:"new_status"`
	}{
		MediaID:   event.GetAggregateID(),
		MediaType: event.GetAggregateType(),
		OldStatus: event.OldStatus,
		NewStatus: event.NewStatus,
	}

	return NewIntegrationEvent("media.status_changed", data, nil)
} 