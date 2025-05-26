package media

import (
	"fmt"
	"time"

	"github.com/narwhalmedia/narwhal/internal/domain/specification"
)

// MediaByStatusSpecification finds media by status
type MediaByStatusSpecification struct {
	specification.BaseSpecification
	Status Status
}

// NewMediaByStatusSpecification creates a new status specification
func NewMediaByStatusSpecification(status Status) *MediaByStatusSpecification {
	return &MediaByStatusSpecification{Status: status}
}

func (s *MediaByStatusSpecification) IsSatisfiedBy(candidate interface{}) bool {
	if media, ok := candidate.(Aggregate); ok {
		return media.GetStatus() == s.Status
	}
	return false
}

func (s *MediaByStatusSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{s.Status}
}

// MediaByGenreSpecification finds media by genre
type MediaByGenreSpecification struct {
	specification.BaseSpecification
	Genre string
}

// NewMediaByGenreSpecification creates a new genre specification
func NewMediaByGenreSpecification(genre string) *MediaByGenreSpecification {
	return &MediaByGenreSpecification{Genre: genre}
}

func (s *MediaByGenreSpecification) IsSatisfiedBy(candidate interface{}) bool {
	switch m := candidate.(type) {
	case *Movie:
		for _, g := range m.Genres {
			if g == s.Genre {
				return true
			}
		}
	case *Series:
		for _, g := range m.Genres {
			if g == s.Genre {
				return true
			}
		}
	}
	return false
}

func (s *MediaByGenreSpecification) ToSQL() (string, []interface{}) {
	return "? = ANY(genres)", []interface{}{s.Genre}
}

// MediaByDateRangeSpecification finds media within a date range
type MediaByDateRangeSpecification struct {
	specification.BaseSpecification
	StartDate time.Time
	EndDate   time.Time
	DateField string // "created_at", "release_date", "first_air_date"
}

// NewMediaByDateRangeSpecification creates a new date range specification
func NewMediaByDateRangeSpecification(startDate, endDate time.Time, dateField string) *MediaByDateRangeSpecification {
	return &MediaByDateRangeSpecification{
		StartDate: startDate,
		EndDate:   endDate,
		DateField: dateField,
	}
}

func (s *MediaByDateRangeSpecification) IsSatisfiedBy(candidate interface{}) bool {
	var date time.Time
	
	switch s.DateField {
	case "created_at":
		if agg, ok := candidate.(Aggregate); ok {
			date = agg.GetCreatedAt()
		}
	case "release_date":
		if movie, ok := candidate.(*Movie); ok {
			date = movie.ReleaseDate
		}
	case "first_air_date":
		if series, ok := candidate.(*Series); ok {
			date = series.FirstAirDate
		}
	}
	
	return !date.IsZero() && date.After(s.StartDate) && date.Before(s.EndDate)
}

func (s *MediaByDateRangeSpecification) ToSQL() (string, []interface{}) {
	return fmt.Sprintf("%s BETWEEN ? AND ?", s.DateField), []interface{}{s.StartDate, s.EndDate}
}

// EpisodeBySeasonSpecification finds episodes by season number
type EpisodeBySeasonSpecification struct {
	specification.BaseSpecification
	SeasonNumber int
}

// NewEpisodeBySeasonSpecification creates a new season specification
func NewEpisodeBySeasonSpecification(seasonNumber int) *EpisodeBySeasonSpecification {
	return &EpisodeBySeasonSpecification{SeasonNumber: seasonNumber}
}

func (s *EpisodeBySeasonSpecification) IsSatisfiedBy(candidate interface{}) bool {
	if episode, ok := candidate.(*Episode); ok {
		return episode.SeasonNumber == s.SeasonNumber
	}
	return false
}

func (s *EpisodeBySeasonSpecification) ToSQL() (string, []interface{}) {
	return "season_number = ?", []interface{}{s.SeasonNumber}
}

// RecentlyAddedSpecification finds recently added media
type RecentlyAddedSpecification struct {
	specification.BaseSpecification
	Days int
}

// NewRecentlyAddedSpecification creates a new recently added specification
func NewRecentlyAddedSpecification(days int) *RecentlyAddedSpecification {
	return &RecentlyAddedSpecification{Days: days}
}

func (s *RecentlyAddedSpecification) IsSatisfiedBy(candidate interface{}) bool {
	if agg, ok := candidate.(Aggregate); ok {
		cutoff := time.Now().AddDate(0, 0, -s.Days)
		return agg.GetCreatedAt().After(cutoff)
	}
	return false
}

func (s *RecentlyAddedSpecification) ToSQL() (string, []interface{}) {
	cutoff := time.Now().AddDate(0, 0, -s.Days)
	return "created_at > ?", []interface{}{cutoff}
}

// ReadyToStreamSpecification finds media ready for streaming
type ReadyToStreamSpecification struct {
	specification.BaseSpecification
}

// NewReadyToStreamSpecification creates a new ready to stream specification
func NewReadyToStreamSpecification() *ReadyToStreamSpecification {
	return &ReadyToStreamSpecification{}
}

func (s *ReadyToStreamSpecification) IsSatisfiedBy(candidate interface{}) bool {
	if media, ok := candidate.(Aggregate); ok {
		return media.GetStatus() == StatusReady
	}
	return false
}

func (s *ReadyToStreamSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{StatusReady}
}