package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	transcodepb "github.com/narwhalmedia/narwhal/api/proto/transcode/v1"
	"github.com/narwhalmedia/narwhal/internal/config"
	grpcinfra "github.com/narwhalmedia/narwhal/internal/infrastructure/grpc"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/grpc/interceptors"
)

func main() {
	// Load configuration
	cfg, err := config.Load("transcode")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize server using wire
	grpcServer, cleanup, err := InitializeTranscodeServer(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize server", zap.Error(err))
	}
	defer cleanup()

	// Start servers
	if err := runServers(cfg, grpcServer, logger); err != nil {
		logger.Fatal("failed to run servers", zap.Error(err))
	}
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.Server.Environment == "development" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func setupGRPCServer(cfg *config.Config, transcodeServer *grpcinfra.TranscodeServiceServer, logger *zap.Logger) *grpc.Server {
	// Create gRPC server with interceptors
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryLoggingInterceptor(logger),
			interceptors.UnaryRecoveryInterceptor(logger),
		),
	}

	grpcServer := grpc.NewServer(opts...)

	// Register service
	transcodepb.RegisterTranscodeServiceServer(grpcServer, transcodeServer)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("transcode.v1.TranscodeService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection for development
	if cfg.Server.Environment == "development" {
		reflection.Register(grpcServer)
	}

	return grpcServer
}

func runServers(cfg *config.Config, grpcServer *grpc.Server, logger *zap.Logger) error {
	// Start gRPC server
	grpcAddr := fmt.Sprintf(":%d", cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcAddr, err)
	}

	// Start gRPC server in background
	go func() {
		logger.Info("starting gRPC server", zap.String("addr", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// Start HTTP gateway if enabled
	if cfg.Server.HTTPPort > 0 {
		if err := runHTTPGateway(cfg, logger); err != nil {
			return fmt.Errorf("failed to start HTTP gateway: %w", err)
		}
	}

	// Wait for shutdown signal
	return waitForShutdown(grpcServer, logger)
}

func runHTTPGateway(cfg *config.Config, logger *zap.Logger) error {
	ctx := context.Background()
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
	)

	// Register gRPC gateway
	grpcEndpoint := fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := transcodepb.RegisterTranscodeServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	httpServer := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("starting HTTP gateway", zap.String("addr", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP gateway failed", zap.Error(err))
		}
	}()

	return nil
}

func waitForShutdown(grpcServer *grpc.Server, logger *zap.Logger) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		logger.Warn("graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		logger.Info("server stopped gracefully")
	}

	return nil
}