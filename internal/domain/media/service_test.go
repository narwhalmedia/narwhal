package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/narwhalmedia/narwhal/internal/domain/events"
)

type mockSeriesRepo struct {
	mock.Mock
}

func (m *mockSeriesRepo) Create(ctx context.Context, series *Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *mockSeriesRepo) Get(ctx context.Context, id uuid.UUID) (*Series, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Series), args.Error(1)
}

func (m *mockSeriesRepo) GetByTitle(ctx context.Context, title string) (*Series, error) {
	args := m.Called(ctx, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Series), args.Error(1)
}

func (m *mockSeriesRepo) Update(ctx context.Context, series *Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *mockSeriesRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockSeriesRepo) AddEpisode(ctx context.Context, episode *Episode) error {
	args := m.Called(ctx, episode)
	return args.Error(0)
}

func (m *mockSeriesRepo) GetEpisode(ctx context.Context, id uuid.UUID) (*Episode, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Episode), args.Error(1)
}

func (m *mockSeriesRepo) GetEpisodes(ctx context.Context, seriesID uuid.UUID) ([]*Episode, error) {
	args := m.Called(ctx, seriesID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Episode), args.Error(1)
}

func (m *mockSeriesRepo) UpdateEpisode(ctx context.Context, episode *Episode) error {
	args := m.Called(ctx, episode)
	return args.Error(0)
}

func (m *mockSeriesRepo) UpdateEpisodeFile(ctx context.Context, id uuid.UUID, filePath string) error {
	args := m.Called(ctx, id, filePath)
	return args.Error(0)
}

func (m *mockSeriesRepo) RemoveEpisode(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockMovieRepo struct {
	mock.Mock
}

func (m *mockMovieRepo) Create(ctx context.Context, movie *Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *mockMovieRepo) Get(ctx context.Context, id uuid.UUID) (*Movie, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Movie), args.Error(1)
}

func (m *mockMovieRepo) GetByTitle(ctx context.Context, title string) (*Movie, error) {
	args := m.Called(ctx, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Movie), args.Error(1)
}

func (m *mockMovieRepo) Update(ctx context.Context, movie *Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *mockMovieRepo) UpdateFile(ctx context.Context, id uuid.UUID, filePath string) error {
	args := m.Called(ctx, id, filePath)
	return args.Error(0)
}

func (m *mockMovieRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockEventStore struct {
	mock.Mock
}

func (m *mockEventStore) Save(ctx context.Context, event events.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type mockEventPublisher struct {
	mock.Mock
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event events.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestService(t *testing.T) {
	ctx := context.Background()

	t.Run("Series Operations", func(t *testing.T) {
		seriesRepo := new(mockSeriesRepo)
		movieRepo := new(mockMovieRepo)
		eventStore := new(mockEventStore)
		eventPub := new(mockEventPublisher)

		service := NewService(seriesRepo, movieRepo, eventStore, eventPub)

		t.Run("Create Series", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			seriesRepo.On("GetByTitle", ctx, series.Title).Return(nil, ErrSeriesNotFound)
			seriesRepo.On("Create", ctx, series).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*SeriesCreated")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*SeriesCreated")).Return(nil)

			err := service.CreateSeries(ctx, series)
			require.NoError(t, err)
		})

		t.Run("Get Series", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			seriesRepo.On("Get", ctx, series.ID).Return(series, nil)
			seriesRepo.On("GetEpisodes", ctx, series.ID).Return([]*Episode{}, nil)

			retrieved, err := service.GetSeries(ctx, series.ID)
			require.NoError(t, err)
			assert.Equal(t, series.ID, retrieved.ID)
			assert.Equal(t, series.Title, retrieved.Title)
		})

		t.Run("Update Series Status", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			seriesRepo.On("Get", ctx, series.ID).Return(series, nil)
			seriesRepo.On("GetEpisodes", ctx, series.ID).Return([]*Episode{}, nil)
			seriesRepo.On("Update", ctx, mock.AnythingOfType("*Series")).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)

			err := service.UpdateSeriesStatus(ctx, series.ID, StatusCompleted)
			require.NoError(t, err)
		})

		t.Run("Delete Series", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			seriesRepo.On("Get", ctx, series.ID).Return(series, nil)
			seriesRepo.On("GetEpisodes", ctx, series.ID).Return([]*Episode{}, nil)
			seriesRepo.On("Delete", ctx, series.ID).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*SeriesDeleted")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*SeriesDeleted")).Return(nil)

			err := service.DeleteSeries(ctx, series.ID)
			require.NoError(t, err)
		})
	})

	t.Run("Episode Operations", func(t *testing.T) {
		seriesRepo := new(mockSeriesRepo)
		movieRepo := new(mockMovieRepo)
		eventStore := new(mockEventStore)
		eventPub := new(mockEventPublisher)

		service := NewService(seriesRepo, movieRepo, eventStore, eventPub)

		t.Run("Add Episode", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			episode := &Episode{
				ID:           uuid.New(),
				SeriesID:     series.ID,
				Title:        "Test Episode",
				Description:  "Test Description",
				SeasonNumber: 1,
				EpisodeNumber: 1,
				AirDate:      time.Now(),
				Status:       StatusActive,
			}

			seriesRepo.On("Get", ctx, series.ID).Return(series, nil)
			seriesRepo.On("GetEpisodes", ctx, series.ID).Return([]*Episode{}, nil)
			seriesRepo.On("AddEpisode", ctx, episode).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*EpisodeAdded")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*EpisodeAdded")).Return(nil)

			err := service.AddEpisode(ctx, episode)
			require.NoError(t, err)
		})

		t.Run("Update Episode Status", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			episode := &Episode{
				ID:           uuid.New(),
				SeriesID:     series.ID,
				Title:        "Test Episode",
				Description:  "Test Description",
				SeasonNumber: 1,
				EpisodeNumber: 1,
				AirDate:      time.Now(),
				Status:       StatusActive,
			}

			seriesRepo.On("GetEpisode", ctx, episode.ID).Return(episode, nil)
			seriesRepo.On("UpdateEpisode", ctx, mock.AnythingOfType("*Episode")).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)

			err := service.UpdateEpisodeStatus(ctx, episode.ID, StatusCompleted)
			require.NoError(t, err)
		})

		t.Run("Update Episode File", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			episode := &Episode{
				ID:           uuid.New(),
				SeriesID:     series.ID,
				Title:        "Test Episode",
				Description:  "Test Description",
				SeasonNumber: 1,
				EpisodeNumber: 1,
				AirDate:      time.Now(),
				Status:       StatusActive,
			}

			filePath := "/path/to/episode.mp4"

			seriesRepo.On("GetEpisode", ctx, episode.ID).Return(episode, nil)
			seriesRepo.On("UpdateEpisodeFile", ctx, episode.ID, filePath).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MediaFileUpdated")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MediaFileUpdated")).Return(nil)

			err := service.UpdateEpisodeFile(ctx, episode.ID, filePath)
			require.NoError(t, err)
		})

		t.Run("Remove Episode", func(t *testing.T) {
			series := &Series{
				ID:          uuid.New(),
				Title:       "Test Series",
				Description: "Test Description",
				Status:      StatusActive,
			}

			episode := &Episode{
				ID:           uuid.New(),
				SeriesID:     series.ID,
				Title:        "Test Episode",
				Description:  "Test Description",
				SeasonNumber: 1,
				EpisodeNumber: 1,
				AirDate:      time.Now(),
				Status:       StatusActive,
			}

			seriesRepo.On("GetEpisode", ctx, episode.ID).Return(episode, nil)
			seriesRepo.On("RemoveEpisode", ctx, episode.ID).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*EpisodeRemoved")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*EpisodeRemoved")).Return(nil)

			err := service.RemoveEpisode(ctx, episode.ID)
			require.NoError(t, err)
		})
	})

	t.Run("Movie Operations", func(t *testing.T) {
		seriesRepo := new(mockSeriesRepo)
		movieRepo := new(mockMovieRepo)
		eventStore := new(mockEventStore)
		eventPub := new(mockEventPublisher)

		service := NewService(seriesRepo, movieRepo, eventStore, eventPub)

		t.Run("Create Movie", func(t *testing.T) {
			movie := &Movie{
				ID:          uuid.New(),
				Title:       "Test Movie",
				Description: "Test Description",
				ReleaseDate: time.Now(),
				Runtime:     120,
				Status:      StatusActive,
			}

			movieRepo.On("GetByTitle", ctx, movie.Title).Return(nil, ErrMovieNotFound)
			movieRepo.On("Create", ctx, movie).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MovieCreated")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MovieCreated")).Return(nil)

			err := service.CreateMovie(ctx, movie)
			require.NoError(t, err)
		})

		t.Run("Get Movie", func(t *testing.T) {
			movie := &Movie{
				ID:          uuid.New(),
				Title:       "Test Movie",
				Description: "Test Description",
				ReleaseDate: time.Now(),
				Runtime:     120,
				Status:      StatusActive,
			}

			movieRepo.On("Get", ctx, movie.ID).Return(movie, nil)

			retrieved, err := service.GetMovie(ctx, movie.ID)
			require.NoError(t, err)
			assert.Equal(t, movie.ID, retrieved.ID)
			assert.Equal(t, movie.Title, retrieved.Title)
		})

		t.Run("Update Movie Status", func(t *testing.T) {
			movie := &Movie{
				ID:          uuid.New(),
				Title:       "Test Movie",
				Description: "Test Description",
				ReleaseDate: time.Now(),
				Runtime:     120,
				Status:      StatusActive,
			}

			movieRepo.On("Get", ctx, movie.ID).Return(movie, nil)
			movieRepo.On("Update", ctx, mock.AnythingOfType("*Movie")).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MediaStatusChanged")).Return(nil)

			err := service.UpdateMovieStatus(ctx, movie.ID, StatusCompleted)
			require.NoError(t, err)
		})

		t.Run("Update Movie File", func(t *testing.T) {
			movie := &Movie{
				ID:          uuid.New(),
				Title:       "Test Movie",
				Description: "Test Description",
				ReleaseDate: time.Now(),
				Runtime:     120,
				Status:      StatusActive,
			}

			filePath := "/path/to/movie.mp4"

			movieRepo.On("Get", ctx, movie.ID).Return(movie, nil)
			movieRepo.On("UpdateFile", ctx, movie.ID, filePath).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MediaFileUpdated")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MediaFileUpdated")).Return(nil)

			err := service.UpdateMovieFile(ctx, movie.ID, filePath)
			require.NoError(t, err)
		})

		t.Run("Delete Movie", func(t *testing.T) {
			movie := &Movie{
				ID:          uuid.New(),
				Title:       "Test Movie",
				Description: "Test Description",
				ReleaseDate: time.Now(),
				Runtime:     120,
				Status:      StatusActive,
			}

			movieRepo.On("Get", ctx, movie.ID).Return(movie, nil)
			movieRepo.On("Delete", ctx, movie.ID).Return(nil)
			eventStore.On("Save", ctx, mock.AnythingOfType("*MovieDeleted")).Return(nil)
			eventPub.On("PublishEvent", ctx, mock.AnythingOfType("*MovieDeleted")).Return(nil)

			err := service.DeleteMovie(ctx, movie.ID)
			require.NoError(t, err)
		})
	})
} 