//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/narwhalmedia/narwhal/internal/config"
	"github.com/narwhalmedia/narwhal/internal/domain/transcode"
	"github.com/narwhalmedia/narwhal/internal/events"
	grpcinfra "github.com/narwhalmedia/narwhal/internal/infrastructure/grpc"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/persistence/gorm"
	transcodeinfra "github.com/narwhalmedia/narwhal/internal/infrastructure/transcode"
)

func InitializeTranscodeServer(cfg *config.Config, logger *zap.Logger) (*grpc.Server, func(), error) {
	wire.Build(
		// Infrastructure
		gorm.NewDB,
		gorm.NewTranscodeJobRepository,
		wire.Bind(new(transcode.Repository), new(*gorm.TranscodeJobRepository)),
		nats.NewClient,
		wire.Bind(new(events.EventBus), new(*nats.Client)),
		
		// Transcode infrastructure
		transcodeinfra.NewFFmpegTranscoder,
		wire.Bind(new(transcode.Transcoder), new(*transcodeinfra.FFmpegTranscoder)),
		provideStorage,
		
		// Domain
		transcode.NewService,
		
		// gRPC
		grpcinfra.NewTranscodeServiceServer,
		provideGRPCServer,
	)
	
	return nil, nil, nil
}

func provideStorage(cfg *config.Config, logger *zap.Logger) (transcode.StorageBackend, error) {
	if cfg.Storage.Type == "s3" {
		return transcodeinfra.NewS3Storage(
			cfg.Storage.S3Config.Bucket,
			"transcode", // prefix
			cfg.Storage.S3Config.Region,
			logger,
		)
	}
	
	return transcodeinfra.NewLocalStorage(cfg.Storage.LocalPath, logger)
}

func provideGRPCServer(cfg *config.Config, transcodeServer *grpcinfra.TranscodeServiceServer, logger *zap.Logger) *grpc.Server {
	return setupGRPCServer(cfg, transcodeServer, logger)
}