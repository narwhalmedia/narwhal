package media

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/events"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

// ApplicationService handles use case orchestration for media operations
type ApplicationService struct {
	// Domain service for business logic
	domainService media.Service
	
	// Repositories
	seriesRepo media.SeriesRepository
	movieRepo  media.MovieRepository
	
	// Event handling
	eventStore              events.EventStore
	domainEventDispatcher   events.DomainEventDispatcher
	integrationEventPub     events.IntegrationEventPublisher
	
	// Unit of Work for transaction management
	unitOfWork UnitOfWork
}

// NewApplicationService creates a new media application service
func NewApplicationService(
	domainService media.Service,
	seriesRepo media.SeriesRepository,
	movieRepo media.MovieRepository,
	eventStore events.EventStore,
	domainEventDispatcher events.DomainEventDispatcher,
	integrationEventPub events.IntegrationEventPublisher,
	unitOfWork UnitOfWork,
) *ApplicationService {
	return &ApplicationService{
		domainService:         domainService,
		seriesRepo:            seriesRepo,
		movieRepo:             movieRepo,
		eventStore:            eventStore,
		domainEventDispatcher: domainEventDispatcher,
		integrationEventPub:   integrationEventPub,
		unitOfWork:            unitOfWork,
	}
}

// CreateSeries creates a new series with proper event handling
func (s *ApplicationService) CreateSeries(ctx context.Context, cmd CreateSeriesCommand) error {
	// Extract correlation ID from context
	correlationID := getCorrelationID(ctx)
	
	// Begin unit of work
	tx, err := s.unitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Create series aggregate
	series := media.NewSeries(
		cmd.Title,
		cmd.Description,
		cmd.FirstAirDate,
		cmd.Genres,
		cmd.Networks,
	)
	
	// Apply domain validation
	if err := series.Validate(); err != nil {
		return fmt.Errorf("invalid series: %w", err)
	}
	
	// Check for duplicates through repository
	existing, err := s.seriesRepo.FindByTitle(ctx, series.Title)
	if err != nil && err != media.ErrSeriesNotFound {
		return fmt.Errorf("checking existing series: %w", err)
	}
	if existing != nil {
		return media.ErrSeriesAlreadyExists
	}
	
	// Save series
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("saving series: %w", err)
	}
	
	// Create domain event
	domainEvent := media.NewSeriesCreatedDomainEvent(series)
	
	// Save to event store
	if err := s.eventStore.Save(ctx, domainEvent); err != nil {
		return fmt.Errorf("saving domain event: %w", err)
	}
	
	// Dispatch domain event synchronously (within transaction)
	if err := s.domainEventDispatcher.Dispatch(ctx, domainEvent); err != nil {
		return fmt.Errorf("dispatching domain event: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	
	// After successful commit, publish integration event
	integrationEvent := media.NewMediaAddedIntegrationEvent(
		series,
		correlationID,
		domainEvent.ID().String(),
	)
	
	// Mark as published before sending
	integrationEvent.MarkAsPublished()
	
	// Publish asynchronously (fire and forget)
	go func() {
		if err := s.integrationEventPub.PublishIntegrationEvent(context.Background(), integrationEvent); err != nil {
			// Log error but don't fail the operation
			// In production, this might trigger a retry mechanism
			fmt.Printf("failed to publish integration event: %v\n", err)
		}
	}()
	
	return nil
}

// AddEpisode adds an episode to a series
func (s *ApplicationService) AddEpisode(ctx context.Context, cmd AddEpisodeCommand) error {
	correlationID := getCorrelationID(ctx)
	
	tx, err := s.unitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Load series aggregate
	series, err := s.seriesRepo.FindByID(ctx, cmd.SeriesID)
	if err != nil {
		return fmt.Errorf("finding series: %w", err)
	}
	
	// Create episode
	episode := media.NewEpisode(
		cmd.Title,
		cmd.Description,
		cmd.SeasonNumber,
		cmd.EpisodeNumber,
		cmd.AirDate,
	)
	
	// Add episode through aggregate root
	if err := series.AddEpisode(episode); err != nil {
		return fmt.Errorf("adding episode: %w", err)
	}
	
	// Save updated series
	if err := s.seriesRepo.Save(ctx, series); err != nil {
		return fmt.Errorf("saving series: %w", err)
	}
	
	// Create domain event
	domainEvent := media.NewEpisodeAddedDomainEvent(series, episode)
	
	// Save to event store
	if err := s.eventStore.Save(ctx, domainEvent); err != nil {
		return fmt.Errorf("saving domain event: %w", err)
	}
	
	// Dispatch domain event
	if err := s.domainEventDispatcher.Dispatch(ctx, domainEvent); err != nil {
		return fmt.Errorf("dispatching domain event: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	
	// Publish integration event
	integrationEvent := media.NewEpisodeAddedToSeriesIntegrationEvent(
		series.GetID(),
		episode.GetID(),
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.Title,
		series.GetVersion(),
		correlationID,
		domainEvent.ID().String(),
	)
	integrationEvent.MarkAsPublished()
	
	go func() {
		if err := s.integrationEventPub.PublishIntegrationEvent(context.Background(), integrationEvent); err != nil {
			fmt.Printf("failed to publish integration event: %v\n", err)
		}
	}()
	
	return nil
}

// CreateMovie creates a new movie
func (s *ApplicationService) CreateMovie(ctx context.Context, cmd CreateMovieCommand) error {
	correlationID := getCorrelationID(ctx)
	
	tx, err := s.unitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Create movie aggregate
	movie := media.NewMovie(
		cmd.Title,
		cmd.Description,
		cmd.ReleaseDate,
		cmd.Genres,
		cmd.Director,
		cmd.Cast,
	)
	
	// Apply domain validation
	if err := movie.Validate(); err != nil {
		return fmt.Errorf("invalid movie: %w", err)
	}
	
	// Check for duplicates
	existing, err := s.movieRepo.FindByTitle(ctx, movie.Title)
	if err != nil && err != media.ErrMovieNotFound {
		return fmt.Errorf("checking existing movie: %w", err)
	}
	if existing != nil {
		return media.ErrMovieAlreadyExists
	}
	
	// Save movie
	if err := s.movieRepo.Save(ctx, movie); err != nil {
		return fmt.Errorf("saving movie: %w", err)
	}
	
	// Create domain event
	domainEvent := media.NewMovieCreatedDomainEvent(movie)
	
	// Save to event store
	if err := s.eventStore.Save(ctx, domainEvent); err != nil {
		return fmt.Errorf("saving domain event: %w", err)
	}
	
	// Dispatch domain event
	if err := s.domainEventDispatcher.Dispatch(ctx, domainEvent); err != nil {
		return fmt.Errorf("dispatching domain event: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	
	// Publish integration event
	integrationEvent := media.NewMediaAddedIntegrationEvent(
		movie,
		correlationID,
		domainEvent.ID().String(),
	)
	integrationEvent.MarkAsPublished()
	
	go func() {
		if err := s.integrationEventPub.PublishIntegrationEvent(context.Background(), integrationEvent); err != nil {
			fmt.Printf("failed to publish integration event: %v\n", err)
		}
	}()
	
	return nil
}

// UpdateMediaStatus updates the status of a media item
func (s *ApplicationService) UpdateMediaStatus(ctx context.Context, cmd UpdateMediaStatusCommand) error {
	correlationID := getCorrelationID(ctx)
	
	tx, err := s.unitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	var domainEvent events.DomainEvent
	var integrationEvent events.IntegrationEvent
	
	// Load the appropriate aggregate based on media type
	switch cmd.MediaType {
	case media.AggregateTypeMovie:
		movie, err := s.movieRepo.FindByID(ctx, cmd.MediaID)
		if err != nil {
			return fmt.Errorf("finding movie: %w", err)
		}
		
		oldStatus := movie.GetStatus()
		movie.SetStatus(cmd.NewStatus)
		
		if err := s.movieRepo.Save(ctx, movie); err != nil {
			return fmt.Errorf("saving movie: %w", err)
		}
		
		domainEvent = media.NewMediaStatusChangedDomainEvent(movie, oldStatus, cmd.NewStatus)
		
		// If status changed to NeedsTranscode, create transcode request
		if cmd.NewStatus == media.StatusNeedsTranscode && cmd.FilePath != "" {
			integrationEvent = media.NewMediaTranscodeRequestedIntegrationEvent(
				movie.GetID(),
				media.AggregateTypeMovie,
				cmd.FilePath,
				[]string{"1080p", "720p", "480p"}, // Default HLS variants
				movie.GetVersion(),
				correlationID,
				domainEvent.ID().String(),
			)
		}
		
	case media.AggregateTypeSeries:
		series, err := s.seriesRepo.FindByID(ctx, cmd.MediaID)
		if err != nil {
			return fmt.Errorf("finding series: %w", err)
		}
		
		oldStatus := series.GetStatus()
		series.SetStatus(cmd.NewStatus)
		
		if err := s.seriesRepo.Save(ctx, series); err != nil {
			return fmt.Errorf("saving series: %w", err)
		}
		
		domainEvent = media.NewMediaStatusChangedDomainEvent(series, oldStatus, cmd.NewStatus)
		
	default:
		return fmt.Errorf("unsupported media type: %s", cmd.MediaType)
	}
	
	// Save domain event
	if err := s.eventStore.Save(ctx, domainEvent); err != nil {
		return fmt.Errorf("saving domain event: %w", err)
	}
	
	// Dispatch domain event
	if err := s.domainEventDispatcher.Dispatch(ctx, domainEvent); err != nil {
		return fmt.Errorf("dispatching domain event: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	
	// Publish integration event if created
	if integrationEvent != nil {
		go func() {
			if err := s.integrationEventPub.PublishIntegrationEvent(context.Background(), integrationEvent); err != nil {
				fmt.Printf("failed to publish integration event: %v\n", err)
			}
		}()
	}
	
	return nil
}

// Helper function to extract correlation ID from context
func getCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	// Generate new correlation ID if not present
	return uuid.New().String()
}