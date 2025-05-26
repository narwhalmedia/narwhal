package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	downloadv1 "github.com/narwhalmedia/narwhal/api/proto/download/v1"
	"github.com/narwhalmedia/narwhal/internal/domain/download"
)

// DownloadService implements the download.v1.DownloadServiceServer interface
type DownloadService struct {
	downloadv1.UnimplementedDownloadServiceServer
	service download.Service
}

// NewDownloadService creates a new download gRPC service
func NewDownloadService(service download.Service) *DownloadService {
	return &DownloadService{
		service: service,
	}
}

// StartDownload starts a new download
func (s *DownloadService) StartDownload(ctx context.Context, req *downloadv1.StartDownloadRequest) (*downloadv1.StartDownloadResponse, error) {
	// Validate request
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	// Default to HTTP download type
	downloadType := download.TypeHTTP

	// Create download
	dl, err := s.service.CreateDownload(ctx, req.Url, downloadType, req.TargetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Start download immediately
	if err := s.service.StartDownload(ctx, dl.ID()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &downloadv1.StartDownloadResponse{
		Download: toProtoDownload(dl),
	}, nil
}

// GetDownload gets download information
func (s *DownloadService) GetDownload(ctx context.Context, req *downloadv1.GetDownloadRequest) (*downloadv1.GetDownloadResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid download ID")
	}

	dl, err := s.service.GetDownload(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &downloadv1.GetDownloadResponse{
		Download: toProtoDownload(dl),
	}, nil
}

// ListDownloads lists downloads
func (s *DownloadService) ListDownloads(ctx context.Context, req *downloadv1.ListDownloadsRequest) (*downloadv1.ListDownloadsResponse, error) {
	var statusFilter *download.Status
	if req.Status != downloadv1.DownloadStatus_DOWNLOAD_STATUS_UNSPECIFIED {
		status := mapProtoStatus(req.Status)
		statusFilter = &status
	}

	downloads, err := s.service.ListDownloads(ctx, statusFilter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoDownloads := make([]*downloadv1.Download, len(downloads))
	for i, dl := range downloads {
		protoDownloads[i] = toProtoDownload(dl)
	}

	return &downloadv1.ListDownloadsResponse{
		Downloads: protoDownloads,
	}, nil
}

// CancelDownload cancels a download
func (s *DownloadService) CancelDownload(ctx context.Context, req *downloadv1.CancelDownloadRequest) (*downloadv1.CancelDownloadResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid download ID")
	}

	if err := s.service.CancelDownload(ctx, id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get updated download
	dl, err := s.service.GetDownload(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &downloadv1.CancelDownloadResponse{
		Download: toProtoDownload(dl),
	}, nil
}

// TODO: Implement these methods when proto definitions are added
// PauseDownload, ResumeDownload, RetryDownload

// WatchDownload streams download progress updates
func (s *DownloadService) WatchDownload(req *downloadv1.WatchDownloadRequest, stream downloadv1.DownloadService_WatchDownloadServer) error {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid download ID")
	}

	ctx := stream.Context()

	// Send initial state
	dl, err := s.service.GetDownload(ctx, id)
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}

	if err := stream.Send(&downloadv1.WatchDownloadResponse{
		Download: toProtoDownload(dl),
	}); err != nil {
		return err
	}

	// TODO: Implement real-time progress streaming
	// This would require subscribing to progress events from the service

	return nil
}

// Helper functions

func toProtoDownload(dl *download.Download) *downloadv1.Download {
	progress := dl.Progress()
	// metadata := dl.Metadata() // TODO: Use when proto is updated

	proto := &downloadv1.Download{
		Id:              dl.ID().String(),
		Url:             dl.URL(),
		// Type:            toProtoDownloadType(dl.Type()), // TODO: Add when DownloadType is in proto
		TargetPath:      dl.TargetPath(),
		Status:          toProtoDownloadStatus(dl.Status()),
		BytesDownloaded: progress.BytesDownloaded,
		TotalBytes:      progress.TotalBytes,
		Progress:        float32(progress.PercentComplete() / 100.0),
		// Speed:           progress.Speed, // TODO: Add to proto
		// Eta:             int64(progress.ETA.Seconds()), // TODO: Add to proto
		// FileName:        metadata.FileName, // TODO: Add to proto
		// ContentType:     metadata.ContentType, // TODO: Add to proto
		ErrorMessage:    dl.Error(),
		CreatedAt:       timestamppb.New(dl.CreatedAt()),
		UpdatedAt:       timestamppb.New(dl.UpdatedAt()),
	}

	// TODO: Add these fields to proto
	// if dl.StartedAt() != nil {
	// 	proto.StartedAt = timestamppb.New(*dl.StartedAt())
	// }
	//
	// if dl.CompletedAt() != nil {
	// 	proto.CompletedAt = timestamppb.New(*dl.CompletedAt())
	// }
	//
	// // Add torrent-specific info
	// if dl.Type() == download.TypeTorrent {
	// 	proto.Seeders = int32(progress.Seeders)
	// 	proto.Leechers = int32(progress.Leechers)
	// 	proto.InfoHash = metadata.InfoHash
	// }

	return proto
}

// TODO: Implement when DownloadType is added to proto
// func toProtoDownloadType(t download.Type) downloadv1.DownloadType

func toProtoDownloadStatus(s download.Status) downloadv1.DownloadStatus {
	switch s {
	case download.StatusPending:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_PENDING
	case download.StatusDownloading:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_DOWNLOADING
	case download.StatusPaused:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_PENDING // Map paused to pending for now
	case download.StatusCompleted:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_COMPLETED
	case download.StatusFailed:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_FAILED
	case download.StatusCancelled:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_CANCELLED
	case download.StatusVerifying:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_DOWNLOADING // Map verifying to downloading
	default:
		return downloadv1.DownloadStatus_DOWNLOAD_STATUS_UNSPECIFIED
	}
}

// TODO: Implement when DownloadType is added to proto
// func mapDownloadType(t downloadv1.DownloadType) download.Type {
//	return download.TypeHTTP
// }

func mapProtoStatus(s downloadv1.DownloadStatus) download.Status {
	switch s {
	case downloadv1.DownloadStatus_DOWNLOAD_STATUS_PENDING:
		return download.StatusPending
	case downloadv1.DownloadStatus_DOWNLOAD_STATUS_DOWNLOADING:
		return download.StatusDownloading
	case downloadv1.DownloadStatus_DOWNLOAD_STATUS_COMPLETED:
		return download.StatusCompleted
	case downloadv1.DownloadStatus_DOWNLOAD_STATUS_FAILED:
		return download.StatusFailed
	case downloadv1.DownloadStatus_DOWNLOAD_STATUS_CANCELLED:
		return download.StatusCancelled
	default:
		return download.StatusPending
	}
}