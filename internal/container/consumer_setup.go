package container

import (
	"context"

	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/handlers"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
	"github.com/narwhalmedia/narwhal/internal/sagas"
)

// SetupEventConsumers sets up event consumers and handlers
func SetupEventConsumers(
	ctx context.Context,
	container *MediaServiceContainer,
	downloadService sagas.DownloadService,  // Will be nil until download service is implemented
	transcodeService sagas.TranscodeService, // Will be nil until transcode service is implemented
) error {
	// Create saga store
	sagaStore := nats.NewInMemorySagaStore()

	// Create saga orchestrator
	orchestrator := nats.NewSagaOrchestrator(
		container.NATSClient,
		nats.NewPublisher(container.NATSClient, container.Logger),
		sagaStore,
		container.Logger,
	)

	// Register sagas (only if services are available)
	if downloadService != nil && transcodeService != nil {
		mediaSaga := sagas.NewMediaProcessingSaga(
			container.MediaService,
			downloadService,
			transcodeService,
			container.Logger,
		)
		orchestrator.RegisterSaga(mediaSaga.GetDefinition())
	}

	// Create consumer group
	consumerConfig := nats.ConsumerConfig{
		Name:       "media-service",
		MaxRetries: 3,
		AckWait:    30,
		MaxDeliver: 5,
	}
	
	consumerGroup := nats.NewConsumerGroup(
		container.NATSClient,
		consumerConfig,
		container.Logger,
	)

	// Register event handlers
	mediaEventHandler := handlers.NewMediaEventHandler(orchestrator, container.Logger)
	consumerGroup.RegisterHandler(mediaEventHandler)

	// Start consumer group in background
	go func() {
		if err := consumerGroup.Start(ctx); err != nil {
			container.Logger.Error("consumer group failed", zap.Error(err))
		}
	}()

	container.Logger.Info("event consumers started")
	return nil
}