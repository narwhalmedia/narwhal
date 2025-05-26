package gorm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
	"gorm.io/gorm"
)

// EventModel represents an event in the database
type EventModel struct {
	BaseModel
	AggregateID   uuid.UUID `gorm:"type:uuid;not null;index"`
	AggregateType string    `gorm:"not null;index"`
	EventType     string    `gorm:"not null;index"`
	Data          []byte    `gorm:"type:jsonb;not null"`
	Metadata      []byte    `gorm:"type:jsonb"`
}

// EventStore implements events.EventStore
type EventStore struct {
	db *gorm.DB
}

// NewEventStore creates a new GORM event store
func NewEventStore(db *gorm.DB) events.EventStore {
	return &EventStore{db: db}
}

// Save persists a single domain event
func (s *EventStore) Save(ctx context.Context, event events.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	metadata, err := json.Marshal(event.Metadata())
	if err != nil {
		return err
	}

	model := EventModel{
		BaseModel: BaseModel{
			ID:        event.ID(),
			Version:   event.Version(),
			CreatedAt: event.CreatedAt(),
			UpdatedAt: time.Now(),
		},
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Data:          data,
		Metadata:      metadata,
	}

	result := s.db.WithContext(ctx).Create(&model)
	return result.Error
}

// SaveEvents persists domain events
func (s *EventStore) SaveEvents(ctx context.Context, aggregateID uuid.UUID, aggregateType string, events []events.Event) error {
	models := make([]EventModel, len(events))
	for i, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		metadata, err := json.Marshal(event.Metadata())
		if err != nil {
			return err
		}

		models[i] = EventModel{
			BaseModel: BaseModel{
				ID:        event.ID(),
				Version:   event.Version(),
				CreatedAt: event.CreatedAt(),
				UpdatedAt: time.Now(),
			},
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			EventType:     event.EventType(),
			Data:          data,
			Metadata:      metadata,
		}
	}

	result := s.db.WithContext(ctx).Create(&models)
	return result.Error
}

// GetEvents retrieves all events for an aggregate
func (s *EventStore) GetEvents(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]events.Event, error) {
	var models []EventModel
	result := s.db.WithContext(ctx).
		Where("aggregate_id = ?", aggregateID).
		Order("version ASC").
		Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	domainEvents := make([]events.Event, len(models))
	for i, model := range models {
		event, err := s.unmarshalEvent(model)
		if err != nil {
			return nil, err
		}
		domainEvents[i] = event
	}

	return domainEvents, nil
}

// GetEventsByType retrieves events of a specific type
func (s *EventStore) GetEventsByType(ctx context.Context, eventType string) ([]events.Event, error) {
	var models []EventModel
	result := s.db.WithContext(ctx).
		Where("event_type = ?", eventType).
		Order("created_at ASC").
		Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	domainEvents := make([]events.Event, len(models))
	for i, model := range models {
		event, err := s.unmarshalEvent(model)
		if err != nil {
			return nil, err
		}
		domainEvents[i] = event
	}

	return domainEvents, nil
}

// GetEventsByTimeRange retrieves events within a time range
func (s *EventStore) GetEventsByTimeRange(ctx context.Context, start, end time.Time) ([]events.Event, error) {
	var models []EventModel
	result := s.db.WithContext(ctx).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at ASC").
		Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	domainEvents := make([]events.Event, len(models))
	for i, model := range models {
		event, err := s.unmarshalEvent(model)
		if err != nil {
			return nil, err
		}
		domainEvents[i] = event
	}

	return domainEvents, nil
}

// unmarshalEvent converts an EventModel to a domain Event
func (s *EventStore) unmarshalEvent(model EventModel) (events.Event, error) {
	// This is a simplified version. In a real implementation, you would need to
	// map the event type to the correct domain event struct and unmarshal into it.
	var event events.BaseEvent
	if err := json.Unmarshal(model.Data, &event); err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
		return nil, err
	}

	// For now, we can't set metadata on BaseEvent as it has private fields
	// This would need to be refactored to properly handle event deserialization
	return &event, nil
} 