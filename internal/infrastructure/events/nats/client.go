package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/config"
)

// Client wraps NATS and JetStream connections
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *zap.Logger
	config *config.Config
}

// NewClient creates a new NATS client with JetStream
func NewClient(cfg *config.Config, logger *zap.Logger) (*Client, func(), error) {
	opts := []nats.Option{
		nats.Name(cfg.NATS.ClientID),
		nats.MaxReconnects(cfg.NATS.MaxReconnect),
		nats.ReconnectWait(cfg.NATS.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Error("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("NATS async error", 
				zap.Error(err),
				zap.String("subject", sub.Subject),
			)
		}),
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATS.URL, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &Client{
		nc:     nc,
		js:     js,
		logger: logger.Named("nats"),
		config: cfg,
	}

	// Initialize streams
	if err := client.initializeStreams(context.Background()); err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	cleanup := func() {
		if err := nc.Drain(); err != nil {
			logger.Error("failed to drain NATS connection", zap.Error(err))
		}
		nc.Close()
	}

	logger.Info("NATS client initialized", 
		zap.String("url", cfg.NATS.URL),
		zap.String("client_id", cfg.NATS.ClientID),
	)

	return client, cleanup, nil
}

// initializeStreams creates the necessary JetStream streams
func (c *Client) initializeStreams(ctx context.Context) error {
	// Media events stream
	mediaStream := jetstream.StreamConfig{
		Name:        "MEDIA_EVENTS",
		Description: "Stream for media domain events",
		Subjects: []string{
			"media.>",
		},
		Retention:    jetstream.LimitsPolicy,
		MaxAge:       30 * 24 * time.Hour, // 30 days
		MaxConsumers: -1,
		Replicas:     1,
		Storage:      jetstream.FileStorage,
		Discard:      jetstream.DiscardOld,
		MaxMsgs:      -1,
		MaxBytes:     -1,
	}

	if _, err := c.js.CreateOrUpdateStream(ctx, mediaStream); err != nil {
		return fmt.Errorf("failed to create media stream: %w", err)
	}

	// Download events stream
	downloadStream := jetstream.StreamConfig{
		Name:        "DOWNLOAD_EVENTS",
		Description: "Stream for download domain events",
		Subjects: []string{
			"download.>",
		},
		Retention:    jetstream.LimitsPolicy,
		MaxAge:       7 * 24 * time.Hour, // 7 days
		MaxConsumers: -1,
		Replicas:     1,
		Storage:      jetstream.FileStorage,
		Discard:      jetstream.DiscardOld,
		MaxMsgs:      -1,
		MaxBytes:     -1,
	}

	if _, err := c.js.CreateOrUpdateStream(ctx, downloadStream); err != nil {
		return fmt.Errorf("failed to create download stream: %w", err)
	}

	// Transcode events stream
	transcodeStream := jetstream.StreamConfig{
		Name:        "TRANSCODE_EVENTS",
		Description: "Stream for transcode domain events",
		Subjects: []string{
			"transcode.>",
		},
		Retention:    jetstream.LimitsPolicy,
		MaxAge:       7 * 24 * time.Hour, // 7 days
		MaxConsumers: -1,
		Replicas:     1,
		Storage:      jetstream.FileStorage,
		Discard:      jetstream.DiscardOld,
		MaxMsgs:      -1,
		MaxBytes:     -1,
	}

	if _, err := c.js.CreateOrUpdateStream(ctx, transcodeStream); err != nil {
		return fmt.Errorf("failed to create transcode stream: %w", err)
	}

	// Saga events stream (for orchestration)
	sagaStream := jetstream.StreamConfig{
		Name:        "SAGA_EVENTS",
		Description: "Stream for saga orchestration events",
		Subjects: []string{
			"saga.>",
		},
		Retention:    jetstream.LimitsPolicy,
		MaxAge:       7 * 24 * time.Hour, // 7 days
		MaxConsumers: -1,
		Replicas:     1,
		Storage:      jetstream.FileStorage,
		Discard:      jetstream.DiscardOld,
		MaxMsgs:      -1,
		MaxBytes:     -1,
	}

	if _, err := c.js.CreateOrUpdateStream(ctx, sagaStream); err != nil {
		return fmt.Errorf("failed to create saga stream: %w", err)
	}

	// Dead letter queue stream
	dlqStream := jetstream.StreamConfig{
		Name:        "DLQ",
		Description: "Dead letter queue for failed messages",
		Subjects: []string{
			"dlq.>",
		},
		Retention:    jetstream.LimitsPolicy,
		MaxAge:       30 * 24 * time.Hour, // 30 days
		MaxConsumers: -1,
		Replicas:     1,
		Storage:      jetstream.FileStorage,
		Discard:      jetstream.DiscardOld,
		MaxMsgs:      -1,
		MaxBytes:     -1,
	}

	if _, err := c.js.CreateOrUpdateStream(ctx, dlqStream); err != nil {
		return fmt.Errorf("failed to create DLQ stream: %w", err)
	}

	c.logger.Info("JetStream streams initialized")
	return nil
}

// Connection returns the underlying NATS connection
func (c *Client) Connection() *nats.Conn {
	return c.nc
}

// JetStream returns the JetStream context
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	return c.nc.IsConnected()
}

// Health checks the health of the NATS connection
func (c *Client) Health() error {
	if !c.IsConnected() {
		return fmt.Errorf("NATS client is not connected")
	}
	
	// Try to get account info as a health check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	info, err := c.js.AccountInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get JetStream account info: %w", err)
	}
	
	c.logger.Debug("NATS health check passed",
		zap.Int("streams", info.Streams),
		zap.Int("consumers", info.Consumers),
	)
	
	return nil
}

// Publish publishes an event to the specified subject (implements EventBus)
func (c *Client) Publish(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = c.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	c.logger.Debug("published event",
		zap.String("subject", subject),
		zap.Any("event", event))
	return nil
}

// Subscribe subscribes to events on the specified subject (implements EventBus)
func (c *Client) Subscribe(ctx context.Context, subject string, handler func([]byte) error) error {
	consumer, err := c.js.CreateOrUpdateConsumer(ctx, "TRANSCODE_EVENTS", jetstream.ConsumerConfig{
		Durable:       fmt.Sprintf("transcode-%s", subject),
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	msgs, err := consumer.Messages()
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				msgs.Stop()
				return
			default:
				msg, err := msgs.Next()
				if err != nil {
					c.logger.Error("failed to get message",
						zap.String("subject", subject),
						zap.Error(err))
					continue
				}
				if err := handler(msg.Data()); err != nil {
					c.logger.Error("failed to handle message",
						zap.String("subject", subject),
						zap.Error(err))
					msg.Nak()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

// Close closes the NATS connection
func (c *Client) Close() error {
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}