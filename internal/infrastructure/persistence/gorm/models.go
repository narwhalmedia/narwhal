package gorm

import (
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Version   int            `gorm:"not null;default:1"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// SeriesModel represents a TV series in the database
type SeriesModel struct {
	BaseModel
	Title       string `gorm:"not null;uniqueIndex"`
	Description string
	Status      string `gorm:"not null;default:'pending'"`
	Episodes    []EpisodeModel `gorm:"foreignKey:SeriesID"`
}

// EpisodeModel represents a single episode in a series
type EpisodeModel struct {
	BaseModel
	SeriesID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Title         string    `gorm:"not null"`
	Description   string
	SeasonNumber  int       `gorm:"not null"`
	EpisodeNumber int       `gorm:"not null"`
	AirDate       time.Time
	Duration      int       `gorm:"not null;default:0"`
	Status        string    `gorm:"not null;default:'pending'"`
	FilePath      string
	ThumbnailPath string
}

// MovieModel represents a movie in the database
type MovieModel struct {
	BaseModel
	Title         string   `gorm:"not null;uniqueIndex"`
	Description   string
	ReleaseDate   time.Time
	Duration      int      `gorm:"not null;default:0"`
	Status        string   `gorm:"not null;default:'pending'"`
	FilePath      string
	ThumbnailPath string
	Genres        []string `gorm:"type:text[]"`
	Director      string
	Cast          []string `gorm:"type:text[]"`
}

// ToDomain converts a SeriesModel to a domain Series
func (m *SeriesModel) ToDomain() *media.Series {
	series := media.NewSeries(m.Title, m.Description)
	series.BaseAggregate.ID = m.ID
	series.BaseAggregate.Version = m.Version
	series.BaseAggregate.CreatedAt = m.CreatedAt
	series.BaseAggregate.UpdatedAt = m.UpdatedAt
	series.Status = media.Status(m.Status)
	
	episodes := make([]media.Episode, len(m.Episodes))
	for i, e := range m.Episodes {
		episodes[i] = *e.ToDomain()
	}
	series.Episodes = episodes
	
	return series
}

// FromDomain converts a domain Series to a SeriesModel
func (m *SeriesModel) FromDomain(s *media.Series) {
	m.ID = s.ID
	m.Version = s.Version
	m.CreatedAt = s.CreatedAt
	m.UpdatedAt = s.UpdatedAt
	m.Title = s.Title
	m.Description = s.Description
	m.Status = string(s.Status)
	
	episodes := make([]EpisodeModel, len(s.Episodes))
	for i, e := range s.Episodes {
		episodes[i] = *NewEpisodeModel(&e)
	}
	m.Episodes = episodes
}

// ToDomain converts an EpisodeModel to a domain Episode
func (m *EpisodeModel) ToDomain() *media.Episode {
	episode := media.NewEpisode(m.SeriesID, m.Title, m.Description, m.SeasonNumber, m.EpisodeNumber, m.AirDate)
	episode.BaseAggregate.ID = m.ID
	episode.BaseAggregate.Version = m.Version
	episode.BaseAggregate.CreatedAt = m.CreatedAt
	episode.BaseAggregate.UpdatedAt = m.UpdatedAt
	episode.Status = media.Status(m.Status)
	episode.FilePath = m.FilePath
	episode.ThumbnailPath = m.ThumbnailPath
	episode.Duration = m.Duration
	return episode
}

// NewEpisodeModel creates a new EpisodeModel from a domain Episode
func NewEpisodeModel(e *media.Episode) *EpisodeModel {
	return &EpisodeModel{
		BaseModel: BaseModel{
			ID:        e.ID,
			Version:   e.Version,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		},
		SeriesID:      e.SeriesID,
		Title:         e.Title,
		Description:   e.Description,
		SeasonNumber:  e.SeasonNumber,
		EpisodeNumber: e.EpisodeNumber,
		AirDate:       e.AirDate,
		Duration:      e.Duration,
		Status:        string(e.Status),
		FilePath:      e.FilePath,
		ThumbnailPath: e.ThumbnailPath,
	}
}

// ToDomain converts a MovieModel to a domain Movie
func (m *MovieModel) ToDomain() *media.Movie {
	movie := media.NewMovie(m.Title, m.Description, m.ReleaseDate, m.Genres, m.Director, m.Cast)
	movie.BaseAggregate.ID = m.ID
	movie.BaseAggregate.Version = m.Version
	movie.BaseAggregate.CreatedAt = m.CreatedAt
	movie.BaseAggregate.UpdatedAt = m.UpdatedAt
	movie.Status = media.Status(m.Status)
	movie.FilePath = m.FilePath
	movie.ThumbnailPath = m.ThumbnailPath
	movie.Duration = m.Duration
	return movie
}

// FromDomain converts a domain Movie to a MovieModel
func (m *MovieModel) FromDomain(movie *media.Movie) {
	m.ID = movie.ID
	m.Version = movie.Version
	m.CreatedAt = movie.CreatedAt
	m.UpdatedAt = movie.UpdatedAt
	m.Title = movie.Title
	m.Description = movie.Description
	m.ReleaseDate = movie.ReleaseDate
	m.Duration = movie.Duration
	m.Status = string(movie.Status)
	m.FilePath = movie.FilePath
	m.ThumbnailPath = movie.ThumbnailPath
	m.Genres = movie.Genres
	m.Director = movie.Director
	m.Cast = movie.Cast
} 