package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
	"gorm.io/gorm"
)

// Event represents an event in the database
type Event struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AggregateID   uuid.UUID `gorm:"type:uuid;not null;index"`
	AggregateType string    `gorm:"not null;index"`
	EventType     string    `gorm:"not null;index"`
	Version       int       `gorm:"not null"`
	Data          string    `gorm:"type:jsonb;not null"`
	Metadata      string    `gorm:"type:jsonb"`
	CreatedAt     time.Time `gorm:"not null;index"`
}

// TableName specifies the table name for Event
func (Event) TableName() string {
	return "events"
}

// EventStore implements events.EventStore
type EventStore struct {
	db *gorm.DB
}

// NewEventStore creates a new PostgreSQL event store
func NewEventStore(db *gorm.DB) *EventStore {
	return &EventStore{db: db}
}

// Save persists an event to the database
func (s *EventStore) Save(ctx context.Context, event events.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event data: %w", err)
	}

	metadata, err := json.Marshal(event.Metadata())
	if err != nil {
		return fmt.Errorf("marshaling event metadata: %w", err)
	}

	eventModel := Event{
		ID:            event.ID(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Data:          string(data),
		Metadata:      string(metadata),
		CreatedAt:     event.CreatedAt(),
	}

	if err := s.db.WithContext(ctx).Create(&eventModel).Error; err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	return nil
}

// GetEvents retrieves events for an aggregate
func (s *EventStore) GetEvents(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]events.Event, error) {
	var eventModels []Event

	err := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND aggregate_type = ?", aggregateID, aggregateType).
		Order("version ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}

	var eventList []events.Event
	for _, model := range eventModels {
		event, err := events.UnmarshalEvent(model.EventType, []byte(model.Data))
		if err != nil {
			return nil, fmt.Errorf("unmarshaling event: %w", err)
		}
		eventList = append(eventList, event)
	}

	return eventList, nil
}

// GetEventsByType retrieves events of a specific type
func (s *EventStore) GetEventsByType(ctx context.Context, eventType string) ([]events.Event, error) {
	var eventModels []Event

	err := s.db.WithContext(ctx).
		Where("event_type = ?", eventType).
		Order("created_at ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}

	var eventList []events.Event
	for _, model := range eventModels {
		event, err := events.UnmarshalEvent(eventType, []byte(model.Data))
		if err != nil {
			return nil, fmt.Errorf("unmarshaling event: %w", err)
		}
		eventList = append(eventList, event)
	}

	return eventList, nil
}

// GetEventsByTimeRange retrieves events within a time range
func (s *EventStore) GetEventsByTimeRange(ctx context.Context, start, end time.Time) ([]events.Event, error) {
	var eventModels []Event

	err := s.db.WithContext(ctx).
		Where("created_at BETWEEN ? AND ?", start, end).
		Order("created_at ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}

	var eventList []events.Event
	for _, model := range eventModels {
		event, err := events.UnmarshalEvent(model.EventType, []byte(model.Data))
		if err != nil {
			return nil, fmt.Errorf("unmarshaling event: %w", err)
		}
		eventList = append(eventList, event)
	}

	return eventList, nil
} 