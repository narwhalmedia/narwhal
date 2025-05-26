package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/narwhalmedia/narwhal/api/proto/media/v1"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

// MediaService implements the media.v1.MediaServiceServer interface
type MediaService struct {
	mediav1.UnimplementedMediaServiceServer
	service media.Service
}

// NewMediaService creates a new media service
func NewMediaService(service media.Service) *MediaService {
	return &MediaService{
		service: service,
	}
}

// CreateSeries creates a new series
func (s *MediaService) CreateSeries(ctx context.Context, req *mediav1.CreateSeriesRequest) (*mediav1.Series, error) {
	// Create domain series object
	series := media.NewSeries(req.Title, req.Description)
	
	// Save series through service
	if err := s.service.CreateSeries(ctx, series); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoSeries(series), nil
}

// GetSeries retrieves a series by ID
func (s *MediaService) GetSeries(ctx context.Context, req *mediav1.GetSeriesRequest) (*mediav1.Series, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	series, err := s.service.GetSeries(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoSeries(series), nil
}

// GetSeriesByTitle retrieves a series by title
func (s *MediaService) GetSeriesByTitle(ctx context.Context, req *mediav1.GetSeriesByTitleRequest) (*mediav1.Series, error) {
	series, err := s.service.GetSeriesByTitle(ctx, req.Title)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoSeries(series), nil
}

// UpdateSeriesStatus updates a series status
func (s *MediaService) UpdateSeriesStatus(ctx context.Context, req *mediav1.UpdateSeriesStatusRequest) (*mediav1.Series, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	// Get the series first
	series, err := s.service.GetSeries(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Update the status
	series.UpdateStatus(toDomainStatus(req.Status))

	// Save the updated series
	if err := s.service.UpdateSeries(ctx, series); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoSeries(series), nil
}

// AddEpisode adds a new episode to a series
func (s *MediaService) AddEpisode(ctx context.Context, req *mediav1.AddEpisodeRequest) (*mediav1.Episode, error) {
	seriesID, err := uuid.Parse(req.SeriesId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	// Create new episode
	episode := media.NewEpisode(
		seriesID,
		req.Title,
		req.Description,
		int(req.SeasonNumber),
		int(req.EpisodeNumber),
		req.AirDate.AsTime(),
	)

	err = s.service.AddEpisode(ctx, seriesID, episode)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoEpisode(episode), nil
}

// RemoveEpisode removes an episode from a series
func (s *MediaService) RemoveEpisode(ctx context.Context, req *mediav1.RemoveEpisodeRequest) (*mediav1.Empty, error) {
	seriesID, err := uuid.Parse(req.SeriesId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	episodeID, err := uuid.Parse(req.EpisodeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid episode ID")
	}

	if err := s.service.RemoveEpisode(ctx, seriesID, episodeID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &mediav1.Empty{}, nil
}

// UpdateEpisodeStatus updates an episode's status
func (s *MediaService) UpdateEpisodeStatus(ctx context.Context, req *mediav1.UpdateEpisodeStatusRequest) (*mediav1.Episode, error) {
	seriesID, err := uuid.Parse(req.SeriesId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	episodeID, err := uuid.Parse(req.EpisodeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid episode ID")
	}

	if err := s.service.UpdateEpisodeStatus(ctx, seriesID, episodeID, toDomainStatus(req.Status)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	series, err := s.service.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	episode, err := series.GetEpisode(episodeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoEpisode(episode), nil
}

// UpdateEpisodeFile updates an episode's file information
func (s *MediaService) UpdateEpisodeFile(ctx context.Context, req *mediav1.UpdateEpisodeFileRequest) (*mediav1.Episode, error) {
	seriesID, err := uuid.Parse(req.SeriesId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid series ID")
	}

	episodeID, err := uuid.Parse(req.EpisodeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid episode ID")
	}

	if err := s.service.UpdateEpisodeFile(
		ctx,
		seriesID,
		episodeID,
		req.FilePath,
		req.ThumbnailPath,
		req.Duration.AsDuration(),
	); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	series, err := s.service.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	episode, err := series.GetEpisode(episodeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoEpisode(episode), nil
}

// CreateMovie creates a new movie
func (s *MediaService) CreateMovie(ctx context.Context, req *mediav1.CreateMovieRequest) (*mediav1.Movie, error) {
	movie, err := s.service.CreateMovie(
		ctx,
		req.Title,
		req.Description,
		req.ReleaseDate.AsTime(),
		req.Genres,
		req.Director,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoMovie(movie), nil
}

// GetMovie retrieves a movie by ID
func (s *MediaService) GetMovie(ctx context.Context, req *mediav1.GetMovieRequest) (*mediav1.Movie, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid movie ID")
	}

	movie, err := s.service.GetMovie(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoMovie(movie), nil
}

// GetMovieByTitle retrieves a movie by title
func (s *MediaService) GetMovieByTitle(ctx context.Context, req *mediav1.GetMovieByTitleRequest) (*mediav1.Movie, error) {
	movie, err := s.service.GetMovieByTitle(ctx, req.Title)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoMovie(movie), nil
}

// UpdateMovieStatus updates a movie's status
func (s *MediaService) UpdateMovieStatus(ctx context.Context, req *mediav1.UpdateMovieStatusRequest) (*mediav1.Movie, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid movie ID")
	}

	if err := s.service.UpdateMovieStatus(ctx, id, toDomainStatus(req.Status)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	movie, err := s.service.GetMovie(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoMovie(movie), nil
}

// UpdateMovieFile updates a movie's file information
func (s *MediaService) UpdateMovieFile(ctx context.Context, req *mediav1.UpdateMovieFileRequest) (*mediav1.Movie, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid movie ID")
	}

	if err := s.service.UpdateMovieFile(
		ctx,
		id,
		req.FilePath,
		req.ThumbnailPath,
		req.Duration.AsDuration(),
	); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	movie, err := s.service.GetMovie(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoMovie(movie), nil
}

// Helper functions for converting between domain and proto types

func toProtoSeries(series *media.Series) *mediav1.Series {
	episodes := make([]*mediav1.Episode, len(series.Episodes))
	for i, episode := range series.Episodes {
		episodes[i] = toProtoEpisode(episode)
	}

	return &mediav1.Series{
		Id:          series.GetID().String(),
		Title:       series.Title,
		Description: series.Description,
		Episodes:    episodes,
		Status:      toProtoStatus(series.Status),
		CreatedAt:   timestamppb.New(series.GetCreatedAt()),
		UpdatedAt:   timestamppb.New(series.GetUpdatedAt()),
	}
}

func toProtoEpisode(episode *media.Episode) *mediav1.Episode {
	return &mediav1.Episode{
		Id:            episode.GetID().String(),
		SeriesId:      episode.SeriesID.String(),
		Title:         episode.Title,
		Description:   episode.Description,
		SeasonNumber:  int32(episode.SeasonNumber),
		EpisodeNumber: int32(episode.EpisodeNumber),
		AirDate:       timestamppb.New(episode.AirDate),
		Duration:      durationpb.New(episode.Duration),
		Status:        toProtoStatus(episode.Status),
		FilePath:      episode.FilePath,
		ThumbnailPath: episode.ThumbnailPath,
		CreatedAt:     timestamppb.New(episode.GetCreatedAt()),
		UpdatedAt:     timestamppb.New(episode.GetUpdatedAt()),
	}
}

func toProtoMovie(movie *media.Movie) *mediav1.Movie {
	return &mediav1.Movie{
		Id:            movie.GetID().String(),
		Title:         movie.Title,
		Description:   movie.Description,
		ReleaseDate:   timestamppb.New(movie.ReleaseDate),
		Genres:        movie.Genres,
		Director:      movie.Director,
		Duration:      durationpb.New(movie.Duration),
		Status:        toProtoStatus(movie.Status),
		FilePath:      movie.FilePath,
		ThumbnailPath: movie.ThumbnailPath,
		CreatedAt:     timestamppb.New(movie.GetCreatedAt()),
		UpdatedAt:     timestamppb.New(movie.GetUpdatedAt()),
	}
}

func toProtoStatus(status media.Status) mediav1.MediaStatus {
	switch status {
	case media.StatusPending:
		return mediav1.MediaStatus_MEDIA_STATUS_PENDING
	case media.StatusDownloading:
		return mediav1.MediaStatus_MEDIA_STATUS_DOWNLOADING
	case media.StatusTranscoding:
		return mediav1.MediaStatus_MEDIA_STATUS_TRANSCODING
	case media.StatusReady:
		return mediav1.MediaStatus_MEDIA_STATUS_READY
	case media.StatusError:
		return mediav1.MediaStatus_MEDIA_STATUS_ERROR
	default:
		return mediav1.MediaStatus_MEDIA_STATUS_UNSPECIFIED
	}
}

func toDomainStatus(status mediav1.MediaStatus) media.Status {
	switch status {
	case mediav1.MediaStatus_MEDIA_STATUS_PENDING:
		return media.StatusPending
	case mediav1.MediaStatus_MEDIA_STATUS_DOWNLOADING:
		return media.StatusDownloading
	case mediav1.MediaStatus_MEDIA_STATUS_TRANSCODING:
		return media.StatusTranscoding
	case mediav1.MediaStatus_MEDIA_STATUS_READY:
		return media.StatusReady
	case mediav1.MediaStatus_MEDIA_STATUS_ERROR:
		return media.StatusError
	default:
		return media.StatusPending
	}
} 