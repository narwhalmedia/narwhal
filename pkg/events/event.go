package events

import (
	"time"
)

// BaseEvent is a basic implementation of the Event interface
type BaseEvent struct {
	Type  string                 `json:"type"`
	Time  int64                  `json:"timestamp"`
	AggID string                 `json:"aggregate_id"`
	Data  map[string]interface{} `json:"data"`
}

// NewEvent creates a new event
func NewEvent(eventType string, data map[string]interface{}) *BaseEvent {
	return &BaseEvent{
		Type:  eventType,
		Time:  time.Now().UnixNano(),
		Data:  data,
		AggID: "",
	}
}

// NewAggregateEvent creates a new event with an aggregate ID
func NewAggregateEvent(eventType string, aggregateID string, data map[string]interface{}) *BaseEvent {
	return &BaseEvent{
		Type:  eventType,
		Time:  time.Now().UnixNano(),
		AggID: aggregateID,
		Data:  data,
	}
}

// EventType returns the type of the event
func (e *BaseEvent) EventType() string {
	return e.Type
}

// Timestamp returns when the event occurred
func (e *BaseEvent) Timestamp() int64 {
	return e.Time
}

// AggregateID returns the ID of the aggregate that produced the event
func (e *BaseEvent) AggregateID() string {
	return e.AggID
}
