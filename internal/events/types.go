package events

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// EventType represents the type of event
type EventType string

const (
	// Media events
	EventTypeMediaDownloaded  EventType = "media.downloaded"
	EventTypeMediaTranscoded  EventType = "media.transcoded"
	EventTypeMediaAdded      EventType = "media.added"
	EventTypeMediaRemoved    EventType = "media.removed"
	
	// Progress events
	EventTypeTranscodingProgress EventType = "transcoding.progress"
	EventTypeDownloadProgress    EventType = "download.progress"
)

// Event represents a generic event with metadata
type Event struct {
	Type      EventType          `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      json.RawMessage    `json:"data"`
}

// NewEvent creates a new event with the given type and data
func NewEvent(eventType EventType, data interface{}) (*Event, error) {
	var rawData json.RawMessage
	var err error

	// Handle protobuf messages specially
	if pm, ok := data.(proto.Message); ok {
		m := protojson.MarshalOptions{
			UseProtoNames: true,
		}
		rawData, err = m.Marshal(pm)
	} else {
		rawData, err = json.Marshal(data)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      rawData,
	}, nil
}

// UnmarshalData unmarshals the event data into the given value
func (e *Event) UnmarshalData(v interface{}) error {
	// Handle protobuf messages specially
	if pm, ok := v.(proto.Message); ok {
		u := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		return u.Unmarshal(e.Data, pm)
	}
	return json.Unmarshal(e.Data, v)
}

// Subject returns the NATS subject for this event
func (e *Event) Subject() string {
	return string(e.Type)
} 