package media

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEpisode      = errors.New("invalid episode")
	ErrInvalidSeriesTitle  = errors.New("series title cannot be empty")
	ErrDuplicateEpisode    = errors.New("episode already exists")
	ErrMaxEpisodesReached  = errors.New("maximum episodes per series reached")
	ErrInvalidSeasonNumber = errors.New("invalid season number")
)

const (
	// MaxEpisodesPerSeries is the maximum number of episodes allowed per series
	MaxEpisodesPerSeries = 1000
)

// Series represents a TV series with multiple episodes
type Series struct {
	BaseAggregate
	title         string    // Private field to enforce encapsulation
	description   string    
	episodes      []Episode // Private to ensure modifications go through aggregate
	status        Status
	firstAirDate  time.Time
	genres        []string
	networks      []string
	filePath      string
	thumbnailPath string
}

// Status represents the current state of a media item
type Status string

const (
	StatusPending      Status = "pending"
	StatusDownloading  Status = "downloading"
	StatusTranscoding  Status = "transcoding"
	StatusNeedsTranscode Status = "needs_transcode"
	StatusReady        Status = "ready"
	StatusError        Status = "error"
)

// NewSeries creates a new Series aggregate with validation
func NewSeries(title, description string, firstAirDate time.Time, genres, networks []string) *Series {
	series := &Series{
		BaseAggregate: NewBaseAggregate(),
		title:         title,
		description:   description,
		episodes:      make([]Episode, 0),
		status:        StatusPending,
		firstAirDate:  firstAirDate,
		genres:        genres,
		networks:      networks,
	}
	return series
}

// Validate validates the series aggregate
func (s *Series) Validate() error {
	if s.title == "" {
		return ErrInvalidSeriesTitle
	}
	if len(s.episodes) > MaxEpisodesPerSeries {
		return ErrMaxEpisodesReached
	}
	return nil
}

// Title returns the series title
func (s *Series) Title() string {
	return s.title
}

// Description returns the series description
func (s *Series) Description() string {
	return s.description
}

// Episodes returns a copy of episodes to prevent external modification
func (s *Series) Episodes() []Episode {
	episodesCopy := make([]Episode, len(s.episodes))
	copy(episodesCopy, s.episodes)
	return episodesCopy
}

// EpisodeCount returns the number of episodes
func (s *Series) EpisodeCount() int {
	return len(s.episodes)
}

// FirstAirDate returns the first air date
func (s *Series) FirstAirDate() time.Time {
	return s.firstAirDate
}

// Genres returns a copy of genres
func (s *Series) Genres() []string {
	genresCopy := make([]string, len(s.genres))
	copy(genresCopy, s.genres)
	return genresCopy
}

// Networks returns a copy of networks
func (s *Series) Networks() []string {
	networksCopy := make([]string, len(s.networks))
	copy(networksCopy, s.networks)
	return networksCopy
}

// AddEpisode adds a new episode to the series with proper validation
func (s *Series) AddEpisode(episode *Episode) error {
	// Validate episode
	if episode == nil {
		return ErrInvalidEpisode
	}
	
	// Episode must belong to this series
	if episode.SeriesID != s.ID {
		episode.SeriesID = s.ID // Set it if not set
	}
	
	// Validate season number
	if episode.SeasonNumber < 1 {
		return ErrInvalidSeasonNumber
	}
	
	// Check maximum episodes limit
	if len(s.episodes) >= MaxEpisodesPerSeries {
		return ErrMaxEpisodesReached
	}
	
	// Check for duplicate episode numbers
	for _, e := range s.episodes {
		if e.SeasonNumber == episode.SeasonNumber && e.EpisodeNumber == episode.EpisodeNumber {
			return fmt.Errorf("%w: S%02dE%02d", ErrDuplicateEpisode, episode.SeasonNumber, episode.EpisodeNumber)
		}
	}

	// Add episode
	s.episodes = append(s.episodes, *episode)
	
	// Update aggregate metadata
	s.incrementVersion()
	
	return nil
}

// GetEpisode retrieves an episode by its ID
func (s *Series) GetEpisode(episodeID uuid.UUID) (*Episode, error) {
	for i := range s.episodes {
		if s.episodes[i].ID == episodeID {
			// Return a copy to prevent external modification
			episodeCopy := s.episodes[i]
			return &episodeCopy, nil
		}
	}
	return nil, ErrEpisodeNotFound
}

// RemoveEpisode removes an episode from the series
func (s *Series) RemoveEpisode(episodeID uuid.UUID) error {
	for i, e := range s.episodes {
		if e.ID == episodeID {
			// Remove episode by creating new slice without it
			s.episodes = append(s.episodes[:i], s.episodes[i+1:]...)
			s.incrementVersion()
			return nil
		}
	}
	return ErrEpisodeNotFound
}

// UpdateMetadata updates series metadata
func (s *Series) UpdateMetadata(title, description string, genres, networks []string) error {
	if title == "" {
		return ErrInvalidSeriesTitle
	}
	
	s.title = title
	s.description = description
	s.genres = genres
	s.networks = networks
	s.incrementVersion()
	
	return nil
}

// SetFilePath sets the file path for the series
func (s *Series) SetFilePath(filePath string) {
	s.filePath = filePath
	s.incrementVersion()
}

// GetFilePath returns the file path
func (s *Series) GetFilePath() string {
	return s.filePath
}

// SetThumbnailPath sets the thumbnail path
func (s *Series) SetThumbnailPath(thumbnailPath string) {
	s.thumbnailPath = thumbnailPath
	s.incrementVersion()
}

// GetThumbnailPath returns the thumbnail path
func (s *Series) GetThumbnailPath() string {
	return s.thumbnailPath
}

// UpdateEpisode updates an existing episode
func (s *Series) UpdateEpisode(episode *Episode) error {
	if episode == nil {
		return ErrInvalidEpisode
	}
	
	for i, e := range s.episodes {
		if e.ID == episode.ID {
			if episode.SeriesID != s.ID {
				return ErrInvalidEpisode
			}
			s.episodes[i] = *episode
			s.incrementVersion()
			return nil
		}
	}
	return ErrEpisodeNotFound
}

// incrementVersion increments the version and updates timestamp
func (s *Series) incrementVersion() {
	s.Version++
	s.UpdatedAt = time.Now()
}

// SetStatus updates the series status
func (s *Series) SetStatus(status Status) {
	s.status = status
	s.incrementVersion()
}

// GetStatus returns the series status
func (s *Series) GetStatus() Status {
	return s.status
}

// GetSeasonNumbers returns a sorted list of unique season numbers
func (s *Series) GetSeasonNumbers() []int {
	seasonMap := make(map[int]bool)
	for _, episode := range s.episodes {
		seasonMap[episode.SeasonNumber] = true
	}
	
	seasons := make([]int, 0, len(seasonMap))
	for season := range seasonMap {
		seasons = append(seasons, season)
	}
	
	// Sort seasons
	for i := 0; i < len(seasons); i++ {
		for j := i + 1; j < len(seasons); j++ {
			if seasons[i] > seasons[j] {
				seasons[i], seasons[j] = seasons[j], seasons[i]
			}
		}
	}
	
	return seasons
}

// GetEpisodesBySeason returns all episodes for a specific season
func (s *Series) GetEpisodesBySeason(seasonNumber int) []Episode {
	var episodes []Episode
	for _, episode := range s.episodes {
		if episode.SeasonNumber == seasonNumber {
			episodes = append(episodes, episode)
		}
	}
	return episodes
}

// HasEpisode checks if an episode exists by season and episode number
func (s *Series) HasEpisode(seasonNumber, episodeNumber int) bool {
	for _, e := range s.episodes {
		if e.SeasonNumber == seasonNumber && e.EpisodeNumber == episodeNumber {
			return true
		}
	}
	return false
}

// GetNextEpisodeNumber returns the next available episode number for a season
func (s *Series) GetNextEpisodeNumber(seasonNumber int) int {
	maxEpisode := 0
	for _, e := range s.episodes {
		if e.SeasonNumber == seasonNumber && e.EpisodeNumber > maxEpisode {
			maxEpisode = e.EpisodeNumber
		}
	}
	return maxEpisode + 1
}
