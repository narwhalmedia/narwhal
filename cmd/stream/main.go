package main

import (
	"context"
	"flag"
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
	"google.golang.org/protobuf/types/known/timestamppb"

	streamv1 "github.com/narwhalmedia/narwhal/api/proto/stream/v1"
)

var (
	grpcPort = flag.Int("grpc-port", 9092, "The gRPC server port")
	httpPort = flag.Int("http-port", 8082, "The HTTP server port")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	streamv1.RegisterStreamServiceServer(grpcServer, &streamServer{})

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}
	go func() {
		logger.Info("starting gRPC server", zap.Int("port", *grpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("failed to serve gRPC", zap.Error(err))
		}
	}()

	// Create HTTP server with gRPC-Gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := streamv1.RegisterStreamServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", *grpcPort), opts); err != nil {
		logger.Fatal("failed to register gateway", zap.Error(err))
	}

	// Start HTTP server
	logger.Info("starting HTTP server", zap.Int("port", *httpPort))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux); err != nil {
		logger.Fatal("failed to serve HTTP", zap.Error(err))
	}
}

// streamServer implements the StreamService
type streamServer struct {
	streamv1.UnimplementedStreamServiceServer
}

func (s *streamServer) GetStreamURL(ctx context.Context, req *streamv1.GetStreamURLRequest) (*streamv1.GetStreamURLResponse, error) {
	// TODO: Implement actual stream URL generation
	return &streamv1.GetStreamURLResponse{
		Url: fmt.Sprintf("http://localhost:8082/stream/%s/%s", req.MediaId, req.Quality),
		Format: "hls",
		Quality: &streamv1.StreamQuality{
			Name: req.Quality,
			Width: 1920,
			Height: 1080,
			Bitrate: 5000,
			Codec: "h264",
		},
	}, nil
}

func (s *streamServer) GetStreamInfo(ctx context.Context, req *streamv1.GetStreamInfoRequest) (*streamv1.GetStreamInfoResponse, error) {
	// TODO: Implement actual stream info lookup
	return &streamv1.GetStreamInfoResponse{
		Info: &streamv1.StreamInfo{
			MediaId: req.MediaId,
			Title: "Example Stream",
			Format: "hls",
			Duration: 3600,
			Qualities: []*streamv1.StreamQuality{
				{
					Name: "1080p",
					Width: 1920,
					Height: 1080,
					Bitrate: 5000,
					Codec: "h264",
				},
				{
					Name: "720p",
					Width: 1280,
					Height: 720,
					Bitrate: 2500,
					Codec: "h264",
				},
			},
		},
	}, nil
}

func (s *streamServer) GetStreamManifest(ctx context.Context, req *streamv1.GetStreamManifestRequest) (*streamv1.GetStreamManifestResponse, error) {
	// TODO: Implement actual manifest generation
	return &streamv1.GetStreamManifestResponse{
		Manifest: "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXT-X-MEDIA-SEQUENCE:0",
		Format: "hls",
		Quality: &streamv1.StreamQuality{
			Name: req.Quality,
			Width: 1920,
			Height: 1080,
			Bitrate: 5000,
			Codec: "h264",
		},
	}, nil
}

func (s *streamServer) GetStreamSegment(ctx context.Context, req *streamv1.GetStreamSegmentRequest) (*streamv1.GetStreamSegmentResponse, error) {
	// TODO: Implement actual segment retrieval
	return &streamv1.GetStreamSegmentResponse{
		Data: []byte("example segment data"),
		Format: "hls",
		Quality: &streamv1.StreamQuality{
			Name: req.Quality,
			Width: 1920,
			Height: 1080,
			Bitrate: 5000,
			Codec: "h264",
		},
	}, nil
} 