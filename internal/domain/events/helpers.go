package events

import (
	"github.com/google/uuid"
)

// NewDomainEvent creates a new domain event with the BaseEvent properly initialized
func NewDomainEvent(eventType, aggregateType string, aggregateID string, version int) BaseEvent {
	id, err := uuid.Parse(aggregateID)
	if err != nil {
		// If parsing fails, create a new UUID
		id = uuid.New()
	}
	
	return NewBaseEvent(id, aggregateType, eventType, version)
}