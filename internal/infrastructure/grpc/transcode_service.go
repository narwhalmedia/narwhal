package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	transcodev1 "github.com/narwhalmedia/narwhal/api/proto/transcode/v1"
	"github.com/narwhalmedia/narwhal/internal/domain/transcode"
)

// TranscodeService implements the gRPC transcode service
type TranscodeService struct {
	transcodev1.UnimplementedTranscodeServiceServer
	service transcode.Service
}

// NewTranscodeService creates a new gRPC transcode service
func NewTranscodeService(service transcode.Service) *TranscodeService {
	return &TranscodeService{
		service: service,
	}
}

// StartTranscode starts a new transcoding job
func (s *TranscodeService) StartTranscode(ctx context.Context, req *transcodev1.StartTranscodeRequest) (*transcodev1.StartTranscodeResponse, error) {
	if req.InputPath == "" {
		return nil, status.Error(codes.InvalidArgument, "input_path is required")
	}
	if req.OutputPath == "" {
		return nil, status.Error(codes.InvalidArgument, "output_path is required")
	}
	if req.Profile == nil {
		return nil, status.Error(codes.InvalidArgument, "profile is required")
	}

	// Map profile name to domain profile
	profile := mapProfile(req.Profile.Name)
	
	// Create job with options from profile
	opts := transcode.JobOptions{
		VideoCodec:  req.Profile.Format,
		AudioCodec:  "aac", // Default
		Container:   req.Profile.Container,
		Bitrate:     int(req.Profile.VideoBitrate),
		Width:       int(req.Profile.Width),
		Height:      int(req.Profile.Height),
		FrameRate:   req.Profile.Framerate,
	}
	
	job, err := s.service.CreateJobWithOptions(ctx, req.InputPath, req.OutputPath, profile, opts)
	if err != nil {
		return nil, handleError(err)
	}

	return &transcodev1.StartTranscodeResponse{
		Job: toProtoJob(job),
	}, nil
}

// GetTranscodeJob retrieves a transcoding job by ID
func (s *TranscodeService) GetTranscodeJob(ctx context.Context, req *transcodev1.GetTranscodeJobRequest) (*transcodev1.GetTranscodeJobResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid job ID")
	}

	job, err := s.service.GetJob(ctx, id)
	if err != nil {
		return nil, handleError(err)
	}

	return &transcodev1.GetTranscodeJobResponse{
		Job: toProtoJob(job),
	}, nil
}

// ListTranscodeJobs lists all transcoding jobs
func (s *TranscodeService) ListTranscodeJobs(ctx context.Context, req *transcodev1.ListTranscodeJobsRequest) (*transcodev1.ListTranscodeJobsResponse, error) {
	// TODO: Implement filtering by status and pagination
	jobs, err := s.service.ListJobs(ctx, nil)
	if err != nil {
		return nil, handleError(err)
	}

	protoJobs := make([]*transcodev1.TranscodeJob, len(jobs))
	for i, job := range jobs {
		protoJobs[i] = toProtoJob(job)
	}

	return &transcodev1.ListTranscodeJobsResponse{
		Jobs:          protoJobs,
		NextPageToken: "", // TODO: Implement pagination
	}, nil
}

// CancelTranscodeJob cancels a transcoding job
func (s *TranscodeService) CancelTranscodeJob(ctx context.Context, req *transcodev1.CancelTranscodeJobRequest) (*transcodev1.CancelTranscodeJobResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid job ID")
	}

	if err := s.service.CancelJob(ctx, id); err != nil {
		return nil, handleError(err)
	}

	// Get updated job
	job, err := s.service.GetJob(ctx, id)
	if err != nil {
		return nil, handleError(err)
	}

	return &transcodev1.CancelTranscodeJobResponse{
		Job: toProtoJob(job),
	}, nil
}

// toProtoJob converts a domain job to a proto job
func toProtoJob(job *transcode.Job) *transcodev1.TranscodeJob {
	if job == nil {
		return nil
	}

	pb := &transcodev1.TranscodeJob{
		Id:           job.ID().String(),
		InputPath:    job.InputPath(),
		OutputPath:   job.OutputPath(),
		Status:       toProtoTranscodeStatus(job.Status()),
		Progress:     float32(job.Progress().Percent),
		ErrorMessage: job.Error(),
		CreatedAt:    timestamppb.New(job.CreatedAt()),
		UpdatedAt:    timestamppb.New(job.UpdatedAt()),
	}

	// Add profile
	metadata := job.Metadata()
	pb.Profile = &transcodev1.TranscodeProfile{
		Name:         string(job.Profile()),
		Format:       metadata.VideoCodec,
		Container:    metadata.Container,
		VideoBitrate: int32(metadata.Bitrate),
		AudioBitrate: 128, // Default
		Width:        int32(metadata.Width),
		Height:       int32(metadata.Height),
		Framerate:    float32(metadata.FrameRate),
	}

	return pb
}

// toProtoTranscodeStatus converts a domain status to a proto status
func toProtoTranscodeStatus(s transcode.Status) transcodev1.TranscodeStatus {
	switch s {
	case transcode.StatusPending:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_PENDING
	case transcode.StatusRunning:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_PROCESSING
	case transcode.StatusCompleted:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_COMPLETED
	case transcode.StatusFailed:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_FAILED
	case transcode.StatusCancelled:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_CANCELLED
	default:
		return transcodev1.TranscodeStatus_TRANSCODE_STATUS_UNSPECIFIED
	}
}

// mapProfile maps a profile name to a domain profile
func mapProfile(name string) transcode.Profile {
	switch name {
	case "hls":
		return transcode.ProfileHLS
	case "mp4":
		return transcode.ProfileMP4
	case "webm":
		return transcode.ProfileWebM
	default:
		return transcode.ProfileMP4
	}
}

// handleError converts domain errors to gRPC errors
func handleError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case transcode.ErrJobNotFound:
		return status.Error(codes.NotFound, "job not found")
	case transcode.ErrJobAlreadyStarted:
		return status.Error(codes.FailedPrecondition, "job already started")
	case transcode.ErrJobNotPending:
		return status.Error(codes.FailedPrecondition, "job is not in pending state")
	case transcode.ErrJobAlreadyCompleted:
		return status.Error(codes.FailedPrecondition, "job already completed")
	case transcode.ErrInvalidProfile:
		return status.Error(codes.InvalidArgument, "invalid profile")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}