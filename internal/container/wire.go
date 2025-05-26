//go:build wireinject
// +build wireinject

package container

import (
	"github.com/google/wire"
	"github.com/narwhalmedia/narwhal/internal/config"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
	grpcservice "github.com/narwhalmedia/narwhal/internal/infrastructure/grpc"
	gormrepo "github.com/narwhalmedia/narwhal/internal/infrastructure/persistence/gorm"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MediaServiceContainer holds all dependencies for the media service
type MediaServiceContainer struct {
	Config       *config.Config
	Logger       *zap.Logger
	DB           *gorm.DB
	NATSClient   *nats.Client
	MediaService media.Service
	GRPCService  *grpcservice.MediaService
}

// InitializeMediaService creates a new media service with all dependencies
func InitializeMediaService(cfg *config.Config, logger *zap.Logger) (*MediaServiceContainer, func(), error) {
	wire.Build(
		// Database
		gormrepo.NewDB,
		
		// Repositories
		gormrepo.NewMovieRepository,
		wire.Bind(new(media.MovieRepository), new(*gormrepo.MovieRepository)),
		gormrepo.NewSeriesRepository,
		wire.Bind(new(media.SeriesRepository), new(*gormrepo.SeriesRepository)),
		
		// Event Store
		gormrepo.NewEventStore,
		wire.Bind(new(media.EventStore), new(*gormrepo.EventStore)),
		
		// NATS Client
		nats.NewClient,
		
		// Event Publisher
		nats.NewPublisher,
		wire.Bind(new(media.EventPublisher), new(*nats.Publisher)),
		
		// Domain Service
		media.NewService,
		
		// gRPC Service
		grpcservice.NewMediaService,
		
		// Container
		wire.Struct(new(MediaServiceContainer), "*"),
	)
	
	return nil, nil, nil
}

