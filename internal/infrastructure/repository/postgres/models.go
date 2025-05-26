package postgres

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Movie represents a movie in the database
type Movie struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title         string         `gorm:"not null"`
	Description   string         `gorm:"type:text"`
	ReleaseDate   time.Time      `gorm:"type:date"`
	Genres        string         `gorm:"type:jsonb"`
	Director      string         
	Status        string         `gorm:"not null"`
	FilePath      string         
	ThumbnailPath string         
	Duration      int            
	Version       int            `gorm:"not null;default:1"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for Movie
func (Movie) TableName() string {
	return "movies"
}

// Series represents a TV series in the database
type Series struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string         `gorm:"not null"`
	Description string         `gorm:"type:text"`
	Status      string         `gorm:"not null"`
	Version     int            `gorm:"not null;default:1"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Episodes    []Episode      `gorm:"foreignKey:SeriesID"`
}

// TableName specifies the table name for Series
func (Series) TableName() string {
	return "series"
}

// Episode represents a single episode in a series
type Episode struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SeriesID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	Title         string         `gorm:"not null"`
	Description   string         `gorm:"type:text"`
	SeasonNumber  int            `gorm:"not null"`
	EpisodeNumber int            `gorm:"not null"`
	AirDate       *time.Time     `gorm:"type:date"`
	Status        string         `gorm:"not null"`
	FilePath      string         
	ThumbnailPath string         
	Duration      int            
	Version       int            `gorm:"not null;default:1"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	Series        Series         `gorm:"foreignKey:SeriesID"`
}

// TableName specifies the table name for Episode
func (Episode) TableName() string {
	return "episodes"
}

// Event represents a domain event in the event store
type Event struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AggregateID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	AggregateType string         `gorm:"not null;index"`
	EventType     string         `gorm:"not null;index"`
	EventData     string         `gorm:"type:jsonb;not null"`
	EventVersion  int            `gorm:"not null"`
	Timestamp     time.Time      `gorm:"not null;index"`
	CreatedAt     time.Time      `gorm:"not null"`
}

// TableName specifies the table name for Event
func (Event) TableName() string {
	return "events"
}