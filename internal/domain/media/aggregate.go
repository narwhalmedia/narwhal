package media

import (
	"time"

	"github.com/google/uuid"
)

// Aggregate is the base interface for all aggregates in the media domain
type Aggregate interface {
	GetID() uuid.UUID
	GetVersion() int
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetStatus() Status
	SetStatus(status Status)
	Validate() error
}

// BaseAggregate provides common fields for all aggregates
type BaseAggregate struct {
	ID        uuid.UUID `json:"id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetID returns the aggregate's ID
func (a BaseAggregate) GetID() uuid.UUID {
	return a.ID
}

// GetVersion returns the aggregate's version
func (a BaseAggregate) GetVersion() int {
	return a.Version
}

// GetCreatedAt returns the aggregate's creation time
func (a BaseAggregate) GetCreatedAt() time.Time {
	return a.CreatedAt
}

// GetUpdatedAt returns the aggregate's last update time
func (a BaseAggregate) GetUpdatedAt() time.Time {
	return a.UpdatedAt
}

// NewBaseAggregate creates a new base aggregate with a new UUID and current timestamps
func NewBaseAggregate() BaseAggregate {
	now := time.Now()
	return BaseAggregate{
		ID:        uuid.New(),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MediaType constants
const (
	MediaTypeSeries  = "series"
	MediaTypeMovie   = "movie"
	MediaTypeEpisode = "episode"
) 