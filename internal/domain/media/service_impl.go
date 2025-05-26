package media

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Service defines the media service interface
type Service interface {
	// Series operations
	CreateSeries(ctx context.Context, series *Series) error
	GetSeries(ctx context.Context, id uuid.UUID) (*Series, error)
	GetSeriesByTitle(ctx context.Context, title string) (*Series, error)
	UpdateSeries(ctx context.Context, series *Series) error
	DeleteSeries(ctx context.Context, id uuid.UUID) error
	ListSeries(ctx context.Context) ([]*Series, error)
	
	// Episode operations
	AddEpisode(ctx context.Context, seriesID uuid.UUID, episode *Episode) error
	GetEpisode(ctx context.Context, id uuid.UUID) (*Episode, error)
	UpdateEpisode(ctx context.Context, episode *Episode) error
	DeleteEpisode(ctx context.Context, id uuid.UUID) error
	ListEpisodesBySeries(ctx context.Context, seriesID uuid.UUID) ([]*Episode, error)
	ListEpisodesBySeason(ctx context.Context, seriesID uuid.UUID, season int) ([]*Episode, error)
	
	// Movie operations  
	CreateMovie(ctx context.Context, movie *Movie) error
	GetMovie(ctx context.Context, id uuid.UUID) (*Movie, error)
	GetMovieByTitle(ctx context.Context, title string) (*Movie, error)
	UpdateMovie(ctx context.Context, movie *Movie) error
	DeleteMovie(ctx context.Context, id uuid.UUID) error
	ListMovies(ctx context.Context) ([]*Movie, error)
	
	// Playback operations
	MarkAsWatched(ctx context.Context, mediaID uuid.UUID, userID uuid.UUID) error
	GetWatchProgress(ctx context.Context, mediaID uuid.UUID, userID uuid.UUID) (*WatchProgress, error)
	UpdateWatchProgress(ctx context.Context, progress *WatchProgress) error
}

// service implements the Service interface
type service struct {
	seriesRepo      SeriesRepository
	movieRepo       MovieRepository
	eventStore      events.EventStore
	eventPub        events.EventPublisher
}

// NewService creates a new media service
func NewService(
	seriesRepo SeriesRepository,
	movieRepo MovieRepository,
	eventStore events.EventStore,
	eventPub events.EventPublisher,
) Service {
	return &service{
		seriesRepo: seriesRepo,
		movieRepo:  movieRepo,
		eventStore: eventStore,
		eventPub:   eventPub,
	}
}

// Series operations

func (s *service) CreateSeries(ctx context.Context, series *Series) error {
	// Check if series with same title exists
	existing, err := s.seriesRepo.FindByTitle(ctx, series.Title)
	if err != nil && err != ErrSeriesNotFound {
		return fmt.Errorf("checking existing series: %w", err)
	}
	if existing != nil {
		return ErrSeriesAlreadyExists
	}

	// Save series
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("creating series: %w", err)
	}

	// Create and save event
	event := NewSeriesCreated(series)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) GetSeries(ctx context.Context, id uuid.UUID) (*Series, error) {
	series, err := s.seriesRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Episodes are already loaded with the series

	return series, nil
}

func (s *service) GetSeriesByTitle(ctx context.Context, title string) (*Series, error) {
	series, err := s.seriesRepo.FindByTitle(ctx, title)
	if err != nil {
		return nil, err
	}

	// Episodes are already loaded with the series
	return series, nil
}

func (s *service) UpdateSeriesStatus(ctx context.Context, id uuid.UUID, status Status) error {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return err
	}

	// Update status
	oldStatus := series.Status
	series.Status = status
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("updating series: %w", err)
	}

	// Create and save event
	event := NewMediaStatusChanged(series, oldStatus, status)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) DeleteSeries(ctx context.Context, id uuid.UUID) error {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return err
	}

	// Delete series
	if err := s.seriesRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting series: %w", err)
	}

	// Create and save event
	event := NewSeriesDeleted(series)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

// Episode operations

func (s *service) AddEpisode(ctx context.Context, seriesID uuid.UUID, episode *Episode) error {
	// Check if series exists
	series, err := s.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return err
	}

	// Add episode
	series.AddEpisode(*episode)
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("saving series with new episode: %w", err)
	}

	// Create and save event
	event := NewEpisodeAdded(series, episode)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) GetEpisode(ctx context.Context, id uuid.UUID) (*Episode, error) {
	// TODO: Implement GetEpisode - need to add to repository
	return nil, fmt.Errorf("not implemented")
}

func (s *service) GetEpisodes(ctx context.Context, seriesID uuid.UUID) ([]*Episode, error) {
	// TODO: Implement GetEpisodes - need to add to repository
	return nil, fmt.Errorf("not implemented")
}

func (s *service) UpdateEpisodeStatus(ctx context.Context, id uuid.UUID, status Status) error {
	episode, err := s.GetEpisode(ctx, id)
	if err != nil {
		return err
	}

	// Update status
	episode.Status = status
	// TODO: UpdateEpisode needs to be implemented in repository
	return fmt.Errorf("not implemented")
}

func (s *service) UpdateEpisodeFile(ctx context.Context, id uuid.UUID, filePath string) error {
	// TODO: Implement UpdateEpisodeFile
	return fmt.Errorf("not implemented")
}

func (s *service) RemoveEpisode(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement RemoveEpisode
	return fmt.Errorf("not implemented")
}

// Movie operations

func (s *service) CreateMovie(ctx context.Context, movie *Movie) error {
	// Check if movie with same title exists
	existing, err := s.movieRepo.FindByTitle(ctx, movie.Title)
	if err != nil && err != ErrMovieNotFound {
		return fmt.Errorf("checking existing movie: %w", err)
	}
	if existing != nil {
		return ErrDuplicateMovie
	}

	// Create movie
	if err := s.movieRepo.Save(ctx, movie); err != nil {
		return fmt.Errorf("creating movie: %w", err)
	}

	// Create and save event
	event := NewMovieCreated(movie)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) GetMovie(ctx context.Context, id uuid.UUID) (*Movie, error) {
	return s.movieRepo.FindByID(ctx, id)
}

func (s *service) GetMovieByTitle(ctx context.Context, title string) (*Movie, error) {
	return s.movieRepo.FindByTitle(ctx, title)
}

func (s *service) UpdateMovieStatus(ctx context.Context, id uuid.UUID, status Status) error {
	movie, err := s.GetMovie(ctx, id)
	if err != nil {
		return err
	}

	// Update status
	oldStatus := movie.Status
	movie.Status = status
	if err := s.movieRepo.Save(ctx, movie); err != nil {
		return fmt.Errorf("updating movie: %w", err)
	}

	// Create and save event
	event := NewMediaStatusChanged(movie, oldStatus, status)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) UpdateMovieFile(ctx context.Context, id uuid.UUID, filePath string) error {
	movie, err := s.GetMovie(ctx, id)
	if err != nil {
		return err
	}

	// Update file path
	movie.UpdateFilePath(filePath)
	if err := s.movieRepo.Save(ctx, movie); err != nil {
		return fmt.Errorf("updating movie: %w", err)
	}

	// Create and save event
	event := NewMediaFileUpdated(movie, filePath, "", 0)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}

func (s *service) DeleteMovie(ctx context.Context, id uuid.UUID) error {
	movie, err := s.GetMovie(ctx, id)
	if err != nil {
		return err
	}

	// Delete movie
	if err := s.movieRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting movie: %w", err)
	}

	// Create and save event
	event := NewMovieDeleted(movie)
	if err := s.eventStore.Save(ctx, event); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	// Publish event
	if err := s.eventPub.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
} 

func (s *service) DeleteEpisode(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement episode deletion
	return fmt.Errorf("not implemented")
}

func (s *service) UpdateEpisode(ctx context.Context, episode *Episode) error {
	// TODO: Implement episode update
	return fmt.Errorf("not implemented")
}

func (s *service) ListEpisodesBySeries(ctx context.Context, seriesID uuid.UUID) ([]*Episode, error) {
	// TODO: Implement episode listing by series
	return nil, fmt.Errorf("not implemented")
}

func (s *service) ListEpisodesBySeason(ctx context.Context, seriesID uuid.UUID, season int) ([]*Episode, error) {
	// TODO: Implement episode listing by season
	return nil, fmt.Errorf("not implemented")
}

func (s *service) MarkAsWatched(ctx context.Context, mediaID uuid.UUID, userID uuid.UUID) error {
	// TODO: Implement mark as watched
	return fmt.Errorf("not implemented")
}

func (s *service) GetWatchProgress(ctx context.Context, mediaID uuid.UUID, userID uuid.UUID) (*WatchProgress, error) {
	// TODO: Implement get watch progress
	return nil, fmt.Errorf("not implemented")
}

func (s *service) UpdateWatchProgress(ctx context.Context, progress *WatchProgress) error {
	// TODO: Implement update watch progress
	return fmt.Errorf("not implemented")
}

func (s *service) UpdateSeries(ctx context.Context, series *Series) error {
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("updating series: %w", err)
	}
	return nil
}

func (s *service) ListSeries(ctx context.Context) ([]*Series, error) {
	return s.seriesRepo.FindAll(ctx)
}

func (s *service) UpdateMovie(ctx context.Context, movie *Movie) error {
	if err := s.movieRepo.Save(ctx, movie); err != nil {
		return fmt.Errorf("updating movie: %w", err)
	}
	return nil
}

func (s *service) ListMovies(ctx context.Context) ([]*Movie, error) {
	return s.movieRepo.FindAll(ctx)
}
