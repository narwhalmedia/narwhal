package download

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/narwhalmedia/narwhal/internal/events"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/narwhalmedia/narwhal/api/proto/download/v1"
)

// Service implements the DownloadService
type Service struct {
	pb.UnimplementedDownloadServiceServer
	logger       *zap.Logger
	downloads    map[string]*pb.Download
	mu           sync.RWMutex
	eventManager *events.EventManager
}

// NewService creates a new download service
func NewService(logger *zap.Logger, eventManager *events.EventManager) *Service {
	return &Service{
		logger:       logger,
		downloads:    make(map[string]*pb.Download),
		eventManager: eventManager,
	}
}

// StartDownload starts a new download
func (s *Service) StartDownload(ctx context.Context, req *pb.StartDownloadRequest) (*pb.Download, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new download
	download := &pb.Download{
		Id:        fmt.Sprintf("dl_%d", time.Now().UnixNano()),
		Url:       req.Url,
		Status:    pb.DownloadStatus_DOWNLOAD_STATUS_PENDING,
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}

	// Store the download
	s.downloads[download.Id] = download

	// Start the download in a goroutine
	go s.downloadFile(ctx, download)

	return download, nil
}

// GetDownload gets a download by ID
func (s *Service) GetDownload(ctx context.Context, req *pb.GetDownloadRequest) (*pb.Download, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	download, ok := s.downloads[req.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "download not found")
	}

	return download, nil
}

// ListDownloads lists all downloads
func (s *Service) ListDownloads(ctx context.Context, req *pb.ListDownloadsRequest) (*pb.ListDownloadsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var downloads []*pb.Download
	for _, download := range s.downloads {
		if req.MediaId != "" && download.MediaId != req.MediaId {
			continue
		}
		downloads = append(downloads, download)
	}

	return &pb.ListDownloadsResponse{
		Downloads: downloads,
	}, nil
}

// CancelDownload cancels a download
func (s *Service) CancelDownload(ctx context.Context, req *pb.CancelDownloadRequest) (*pb.Download, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	download, ok := s.downloads[req.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "download not found")
	}

	if download.Status == pb.DownloadStatus_DOWNLOAD_STATUS_COMPLETED {
		return nil, status.Error(codes.FailedPrecondition, "download already completed")
	}

	if download.Status == pb.DownloadStatus_DOWNLOAD_STATUS_CANCELLED {
		return nil, status.Error(codes.FailedPrecondition, "download already cancelled")
	}

	download.Status = pb.DownloadStatus_DOWNLOAD_STATUS_CANCELLED
	download.UpdatedAt = timestamppb.Now()

	return download, nil
}

// WatchDownload streams download progress updates
func (s *Service) WatchDownload(req *pb.WatchDownloadRequest, stream pb.DownloadService_WatchDownloadServer) error {
	s.mu.RLock()
	download, ok := s.downloads[req.Id]
	s.mu.RUnlock()

	if !ok {
		return status.Error(codes.NotFound, "download not found")
	}

	// Create a channel to receive progress updates
	progressChan := make(chan *pb.Download, 1)
	defer close(progressChan)

	// Subscribe to progress events
	err := s.eventManager.Subscribe(stream.Context(), events.EventTypeDownloadProgress, func(ctx context.Context, event *events.Event) error {
		var progress pb.DownloadProgress
		if err := event.UnmarshalData(&progress); err != nil {
			return err
		}

		if progress.MediaId == download.Id {
			progressChan <- download
		}
		return nil
	})
	if err != nil {
		return status.Error(codes.Internal, "failed to subscribe to progress updates")
	}

	// Stream updates until the download is complete or cancelled
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case download := <-progressChan:
			if err := stream.Send(&pb.WatchDownloadResponse{Download: download}); err != nil {
				return status.Error(codes.Internal, "failed to send progress update")
			}
			if download.Status == pb.DownloadStatus_DOWNLOAD_STATUS_COMPLETED ||
				download.Status == pb.DownloadStatus_DOWNLOAD_STATUS_CANCELLED {
				return nil
			}
		}
	}
}

// downloadFile handles the actual file download
func (s *Service) downloadFile(ctx context.Context, download *pb.Download) {
	// Update status to downloading
	s.mu.Lock()
	download.Status = pb.DownloadStatus_DOWNLOAD_STATUS_DOWNLOADING
	download.UpdatedAt = timestamppb.Now()
	s.mu.Unlock()

	// TODO: Implement actual file download logic here
	// For now, we'll just simulate progress updates

	totalBytes := int64(1000000) // 1MB
	bytesDownloaded := int64(0)
	chunkSize := int64(100000) // 100KB chunks

	for bytesDownloaded < totalBytes {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			download.Status = pb.DownloadStatus_DOWNLOAD_STATUS_CANCELLED
			download.UpdatedAt = timestamppb.Now()
			s.mu.Unlock()
			return
		default:
			// Simulate download progress
			bytesDownloaded += chunkSize
			if bytesDownloaded > totalBytes {
				bytesDownloaded = totalBytes
			}

			progress := float32(bytesDownloaded) / float32(totalBytes) * 100

			// Update download progress
			s.mu.Lock()
			download.BytesDownloaded = bytesDownloaded
			download.TotalBytes = totalBytes
			download.Progress = progress
			download.UpdatedAt = timestamppb.Now()
			s.mu.Unlock()

			// Publish progress event
			err := s.eventManager.Publish(ctx, events.EventTypeDownloadProgress, &pb.DownloadProgress{
				MediaId:        download.Id,
				Progress:       progress,
				BytesDownloaded: bytesDownloaded,
				TotalBytes:     totalBytes,
				UpdatedAt:      timestamppb.Now(),
			})
			if err != nil {
				s.logger.Error("failed to publish progress event",
					zap.String("download_id", download.Id),
					zap.Error(err))
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	// Update status to completed
	s.mu.Lock()
	download.Status = pb.DownloadStatus_DOWNLOAD_STATUS_COMPLETED
	download.Progress = 100
	download.UpdatedAt = timestamppb.Now()
	s.mu.Unlock()

	// Publish completed event
	err := s.eventManager.Publish(ctx, events.EventTypeMediaDownloaded, &pb.MediaDownloaded{
		MediaId:      download.Id,
		FilePath:     download.FilePath,
		FileSize:     download.TotalBytes,
		DownloadedAt: timestamppb.Now(),
	})
	if err != nil {
		s.logger.Error("failed to publish completed event",
			zap.String("download_id", download.Id),
			zap.Error(err))
	}
} 