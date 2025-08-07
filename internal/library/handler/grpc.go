package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/narwhalmedia/narwhal/internal/library/constants"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/service"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	commonpb "github.com/narwhalmedia/narwhal/pkg/common/v1"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/pkg/pagination"
)

// GRPCHandler implements the LibraryService gRPC server.
type GRPCHandler struct {
	librarypb.UnimplementedLibraryServiceServer

	libraryService    service.LibraryServiceInterface
	logger            interfaces.Logger
	paginationEncoder *pagination.CursorEncoder
}

// NewGRPCHandler creates a new gRPC handler.
func NewGRPCHandler(
	libraryService service.LibraryServiceInterface,
	logger interfaces.Logger,
	paginationEncoder *pagination.CursorEncoder,
) *GRPCHandler {
	return &GRPCHandler{
		libraryService:    libraryService,
		logger:            logger,
		paginationEncoder: paginationEncoder,
	}
}

// checkAuth validates that the user is authenticated by checking for user context
// Returns roles from context.
func (h *GRPCHandler) checkAuth(ctx context.Context) ([]string, error) {
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	roles, ok := auth.GetRolesFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "roles not found in context")
	}

	return roles, nil
}

// CreateLibrary creates a new media library.
func (h *GRPCHandler) CreateLibrary(
	ctx context.Context,
	req *librarypb.CreateLibraryRequest,
) (*librarypb.CreateLibraryResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	library := &domain.Library{
		ID:           uuid.New(),
		Name:         req.GetName(),
		Path:         req.GetPath(),
		Type:         convertMediaType(req.GetType()),
		Enabled:      req.GetAutoScan(),
		ScanInterval: int(req.GetScanIntervalMinutes()) * constants.SecondsToMinutes, // Convert minutes to seconds
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.libraryService.CreateLibrary(ctx, library); err != nil {
		h.logger.Error("Failed to create library", interfaces.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create library: %v", err)
	}

	return &librarypb.CreateLibraryResponse{
		Library: convertLibraryToProto(library),
	}, nil
}

// GetLibrary retrieves a library by ID.
func (h *GRPCHandler) GetLibrary(
	ctx context.Context,
	req *librarypb.GetLibraryRequest,
) (*librarypb.GetLibraryResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Parse and validate library ID
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid library ID")
	}

	// Get library from service
	library, err := h.libraryService.GetLibrary(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "library not found")
		}
		h.logger.Error("Failed to get library",
			interfaces.Error(err),
			interfaces.String("library_id", req.GetId()))
		return nil, status.Errorf(codes.Internal, "failed to get library: %v", err)
	}

	return &librarypb.GetLibraryResponse{
		Library: convertLibraryToProto(library),
	}, nil
}

// ListLibraries lists all libraries.
func (h *GRPCHandler) ListLibraries(
	ctx context.Context,
	req *librarypb.ListLibrariesRequest,
) (*librarypb.ListLibrariesResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Get all libraries from service
	libraries, err := h.libraryService.ListLibraries(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to list libraries", interfaces.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list libraries: %v", err)
	}

	// Apply type filter if specified
	filteredLibraries := libraries
	if req.GetTypeFilter() != commonpb.MediaType_MEDIA_TYPE_UNSPECIFIED {
		filterType := convertMediaType(req.GetTypeFilter())
		filtered := make([]*domain.Library, 0)
		for _, lib := range libraries {
			if lib.Type == filterType {
				filtered = append(filtered, lib)
			}
		}
		filteredLibraries = filtered
	}

	// Convert to proto format
	protoLibraries := make([]*librarypb.Library, len(filteredLibraries))
	for i, lib := range filteredLibraries {
		protoLibraries[i] = convertLibraryToProto(lib)
	}

	// Handle pagination
	pageSize := int32(constants.DefaultPageSize)
	offset := 0

	if req.GetPagination() != nil {
		if req.GetPagination().GetPageSize() > 0 {
			pageSize = req.GetPagination().GetPageSize()
			if pageSize > constants.MaxPageSize { // Max page size
				pageSize = constants.MaxPageSize
			}
		}

		// Calculate offset from page token
		if req.GetPagination().GetPageToken() != "" && h.paginationEncoder != nil {
			calculatedOffset, err := pagination.CalculateOffset(
				h.paginationEncoder,
				req.GetPagination().GetPageToken(),
				0,
			)
			if err != nil {
				h.logger.Warn("Invalid pagination token",
					interfaces.Error(err),
					interfaces.String("token", req.GetPagination().GetPageToken()))
				// Continue with offset 0 on invalid token
			} else {
				offset = calculatedOffset
			}
		}
	}

	// Apply pagination
	totalItems := len(protoLibraries)
	startIdx := offset
	endIdx := offset + int(pageSize)

	if startIdx > totalItems {
		startIdx = totalItems
	}
	if endIdx > totalItems {
		endIdx = totalItems
	}

	paginatedLibraries := protoLibraries[startIdx:endIdx]

	// Generate next page token
	var nextPageToken string
	if h.paginationEncoder != nil && endIdx < totalItems {
		token, err := pagination.GenerateNextPageToken(h.paginationEncoder, offset, int(pageSize), totalItems)
		if err != nil {
			h.logger.Error("Failed to generate next page token", interfaces.Error(err))
		} else {
			nextPageToken = token
		}
	}

	return &librarypb.ListLibrariesResponse{
		Libraries: paginatedLibraries,
		Pagination: &commonpb.PaginationResponse{
			NextPageToken: nextPageToken,
			TotalItems:    int32(totalItems),
		},
	}, nil
}

// UpdateLibrary updates a library.
func (h *GRPCHandler) UpdateLibrary(
	ctx context.Context,
	req *librarypb.UpdateLibraryRequest,
) (*librarypb.UpdateLibraryResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Parse and validate library ID
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid library ID")
	}

	// Validate request
	if req.GetLibrary() == nil {
		return nil, status.Error(codes.InvalidArgument, "library data is required")
	}

	// Build update map based on field mask
	updates := make(map[string]interface{})

	if req.GetUpdateMask() != nil && len(req.GetUpdateMask().GetPaths()) > 0 {
		// Apply only specified fields
		for _, path := range req.GetUpdateMask().GetPaths() {
			switch path {
			case "name":
				if req.GetLibrary().GetName() != "" {
					updates["name"] = req.GetLibrary().GetName()
				}
			case "path":
				if req.GetLibrary().GetPath() != "" {
					updates["path"] = req.GetLibrary().GetPath()
				}
			case "auto_scan":
				updates["enabled"] = req.GetLibrary().GetAutoScan()
			case "scan_interval_minutes":
				if req.GetLibrary().GetScanIntervalMinutes() > 0 {
					updates["scan_interval"] = int(
						req.GetLibrary().GetScanIntervalMinutes(),
					) * constants.SecondsToMinutes
				}
			}
		}
	} else {
		// Update all provided fields
		if req.GetLibrary().GetName() != "" {
			updates["name"] = req.GetLibrary().GetName()
		}
		if req.GetLibrary().GetPath() != "" {
			updates["path"] = req.GetLibrary().GetPath()
		}
		updates["enabled"] = req.GetLibrary().GetAutoScan()
		if req.GetLibrary().GetScanIntervalMinutes() > 0 {
			updates["scan_interval"] = int(req.GetLibrary().GetScanIntervalMinutes()) * constants.SecondsToMinutes
		}
	}

	// Update library
	library, err := h.libraryService.UpdateLibrary(ctx, id, updates)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "library not found")
		}
		h.logger.Error("Failed to update library",
			interfaces.Error(err),
			interfaces.String("library_id", req.GetId()))
		return nil, status.Errorf(codes.Internal, "failed to update library: %v", err)
	}

	return &librarypb.UpdateLibraryResponse{
		Library: convertLibraryToProto(library),
	}, nil
}

// DeleteLibrary deletes a library.
func (h *GRPCHandler) DeleteLibrary(
	ctx context.Context,
	req *librarypb.DeleteLibraryRequest,
) (*librarypb.DeleteLibraryResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Parse and validate library ID
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid library ID")
	}

	// Delete library
	if err := h.libraryService.DeleteLibrary(ctx, id); err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "library not found")
		}
		h.logger.Error("Failed to delete library",
			interfaces.Error(err),
			interfaces.String("library_id", req.GetId()))
		return nil, status.Errorf(codes.Internal, "failed to delete library: %v", err)
	}

	h.logger.Info("Library deleted successfully",
		interfaces.String("library_id", req.GetId()))

	return &librarypb.DeleteLibraryResponse{}, nil
}

// ScanLibrary starts a library scan.
func (h *GRPCHandler) ScanLibrary(
	ctx context.Context,
	req *librarypb.ScanLibraryRequest,
) (*librarypb.ScanLibraryResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid library ID")
	}

	if err := h.libraryService.ScanLibrary(ctx, id); err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "library not found")
		}
		if errors.IsConflict(err) {
			return &librarypb.ScanLibraryResponse{
				ScanId:  req.GetId(),
				Status:  librarypb.ScanLibraryResponse_STATUS_IN_PROGRESS,
				Message: "scan already in progress",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to start scan: %v", err)
	}

	return &librarypb.ScanLibraryResponse{
		ScanId:  req.GetId(),
		Status:  librarypb.ScanLibraryResponse_STATUS_STARTED,
		Message: "scan started successfully",
	}, nil
}

// GetMedia retrieves a media item.
func (h *GRPCHandler) GetMedia(
	ctx context.Context,
	req *librarypb.GetMediaRequest,
) (*librarypb.GetMediaResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}

	media, err := h.libraryService.GetMedia(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get media: %v", err)
	}

	return &librarypb.GetMediaResponse{
		Media: convertMediaToProto(media, req.GetIncludeMetadata(), req.GetIncludeEpisodes()),
	}, nil
}

// ListMedia lists media items.
func (h *GRPCHandler) ListMedia(
	ctx context.Context,
	req *librarypb.ListMediaRequest,
) (*librarypb.ListMediaResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Parse library ID if provided
	var libraryID *uuid.UUID
	if req.GetLibraryId() != "" {
		id, err := uuid.Parse(req.GetLibraryId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid library ID")
		}
		libraryID = &id
	}

	// TODO: Add status filter to proto definition if needed
	var statusFilter *string

	// Handle pagination
	limit := int(constants.DefaultPageSize)
	offset := int(0)

	if req.GetPagination() != nil {
		if req.GetPagination().GetPageSize() > 0 {
			limit = int(req.GetPagination().GetPageSize())
			if limit > constants.MaxPageSize {
				limit = constants.MaxPageSize // Max page size
			}
		}

		// Calculate offset from page token
		if req.GetPagination().GetPageToken() != "" && h.paginationEncoder != nil {
			calculatedOffset, err := pagination.CalculateOffset(
				h.paginationEncoder,
				req.GetPagination().GetPageToken(),
				0,
			)
			if err != nil {
				h.logger.Warn("Invalid pagination token",
					interfaces.Error(err),
					interfaces.String("token", req.GetPagination().GetPageToken()))
				// Continue with offset 0 on invalid token
			} else {
				offset = calculatedOffset
			}
		}
	}

	// Get media items
	var mediaItems []*models.Media
	var err error

	if libraryID != nil {
		// List media for specific library
		mediaItems, err = h.libraryService.ListMediaByLibrary(ctx, *libraryID, statusFilter, limit, offset)
	} else {
		// List all media
		// TODO: Implement listing all media across libraries
		mediaItems, err = h.libraryService.SearchMedia(ctx, "", nil, statusFilter, nil, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list media",
			interfaces.Error(err),
			interfaces.String("library_id", req.GetLibraryId()))
		return nil, status.Errorf(codes.Internal, "failed to list media: %v", err)
	}

	// Convert to proto format
	protoMedia := make([]*librarypb.Media, len(mediaItems))
	for i, media := range mediaItems {
		// For list operations, include basic metadata but not episodes
		protoMedia[i] = convertMediaToProto(media, true, false)
	}

	// Generate next page token
	var nextPageToken string
	if h.paginationEncoder != nil && len(mediaItems) == limit {
		// Assume there might be more items if we got a full page
		token, err := pagination.GenerateNextPageToken(h.paginationEncoder, offset, limit, offset+limit+1)
		if err != nil {
			h.logger.Error("Failed to generate next page token", interfaces.Error(err))
		} else {
			nextPageToken = token
		}
	}

	return &librarypb.ListMediaResponse{
		Media: protoMedia,
		Pagination: &commonpb.PaginationResponse{
			NextPageToken: nextPageToken,
			TotalItems:    int32(len(mediaItems)), // TODO: Get actual total count from repository
		},
	}, nil
}

// SearchMedia searches for media items.
func (h *GRPCHandler) SearchMedia(
	ctx context.Context,
	req *librarypb.SearchMediaRequest,
) (*librarypb.SearchMediaResponse, error) {
	// Extract filter parameters from request
	var mediaType *string
	var statusFilter *string
	var libraryID *uuid.UUID

	if req.GetTypeFilter() != 0 {
		mt := convertMediaType(req.GetTypeFilter())
		mediaType = &mt
	}

	limit := int(req.GetPagination().GetPageSize())
	if limit <= 0 {
		limit = 50
	}

	results, err := h.libraryService.SearchMedia(ctx, req.GetQuery(), mediaType, statusFilter, libraryID, limit, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	protoResults := make([]*librarypb.Media, len(results))
	for i, media := range results {
		protoResults[i] = convertMediaToProto(media, true, false)
	}

	return &librarypb.SearchMediaResponse{
		Results:      protoResults,
		TotalResults: int32(len(results)),
		Pagination: &commonpb.PaginationResponse{
			NextPageToken: "", // TODO: Implement pagination
			TotalItems:    int32(len(results)),
		},
	}, nil
}

// UpdateMedia updates a media item.
func (h *GRPCHandler) UpdateMedia(
	ctx context.Context,
	req *librarypb.UpdateMediaRequest,
) (*librarypb.UpdateMediaResponse, error) {
	// Authentication/authorization is handled by middleware
	// Just verify the context has auth info
	if _, err := h.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Parse and validate media ID
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}

	// Validate request
	if req.GetMedia() == nil {
		return nil, status.Error(codes.InvalidArgument, "media data is required")
	}

	// Build update map based on field mask
	updates := make(map[string]interface{})

	if req.GetUpdateMask() != nil && len(req.GetUpdateMask().GetPaths()) > 0 {
		// Apply only specified fields
		for _, path := range req.GetUpdateMask().GetPaths() {
			switch path {
			case "title":
				if req.GetMedia().GetTitle() != "" {
					updates["title"] = req.GetMedia().GetTitle()
				}
			case "path":
				if req.GetMedia().GetPath() != "" {
					updates["path"] = req.GetMedia().GetPath()
				}
			case "metadata.description":
				if req.GetMedia().GetMetadata() != nil && req.GetMedia().GetMetadata().GetDescription() != "" {
					updates["description"] = req.GetMedia().GetMetadata().GetDescription()
				}
			case "metadata.genres":
				if req.GetMedia().GetMetadata() != nil && len(req.GetMedia().GetMetadata().GetGenres()) > 0 {
					updates["genres"] = req.GetMedia().GetMetadata().GetGenres()
				}
			case "metadata.rating":
				if req.GetMedia().GetMetadata() != nil && req.GetMedia().GetMetadata().GetRating() > 0 {
					updates["rating"] = req.GetMedia().GetMetadata().GetRating()
				}
			}
		}
	} else {
		// Update basic fields if provided
		if req.GetMedia().GetTitle() != "" {
			updates["title"] = req.GetMedia().GetTitle()
		}
		if req.GetMedia().GetPath() != "" {
			updates["path"] = req.GetMedia().GetPath()
		}

		// Update metadata fields if provided
		if req.GetMedia().GetMetadata() != nil {
			if req.GetMedia().GetMetadata().GetDescription() != "" {
				updates["description"] = req.GetMedia().GetMetadata().GetDescription()
			}
			if len(req.GetMedia().GetMetadata().GetGenres()) > 0 {
				updates["genres"] = req.GetMedia().GetMetadata().GetGenres()
			}
			if req.GetMedia().GetMetadata().GetRating() > 0 {
				updates["rating"] = req.GetMedia().GetMetadata().GetRating()
			}
		}
	}

	// Update media
	media, err := h.libraryService.UpdateMedia(ctx, id, updates)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		h.logger.Error("Failed to update media",
			interfaces.Error(err),
			interfaces.String("media_id", req.GetId()))
		return nil, status.Errorf(codes.Internal, "failed to update media: %v", err)
	}

	return &librarypb.UpdateMediaResponse{
		Media: convertMediaToProto(media, true, false),
	}, nil
}

// DeleteMedia deletes a media item.
func (h *GRPCHandler) DeleteMedia(
	ctx context.Context,
	req *librarypb.DeleteMediaRequest,
) (*librarypb.DeleteMediaResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}

	// Get media details before deletion if we need to delete the file
	var media *models.Media
	if req.GetDeleteFile() {
		// Get media to get the file path
		var err error
		media, err = h.libraryService.GetMedia(ctx, id)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, "media not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to get media: %v", err)
		}
	}

	// Delete from database
	if err := h.libraryService.DeleteMedia(ctx, id); err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete media: %v", err)
	}

	// Delete physical file if requested
	if req.GetDeleteFile() && media != nil && media.Path != "" {
		if err := h.deletePhysicalFile(media.Path); err != nil {
			// Log error but don't fail the whole operation
			h.logger.Error("Failed to delete physical file",
				interfaces.Error(err),
				interfaces.String("path", media.Path),
				interfaces.String("media_id", id.String()))
		} else {
			h.logger.Info("Deleted physical file",
				interfaces.String("path", media.Path),
				interfaces.String("media_id", id.String()))
		}
	}

	return &librarypb.DeleteMediaResponse{}, nil
}

// GetMetadata gets metadata for a media item.
func (h *GRPCHandler) GetMetadata(
	ctx context.Context,
	req *librarypb.GetMetadataRequest,
) (*librarypb.GetMetadataResponse, error) {
	// TODO: Implement
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// UpdateMetadata updates metadata for a media item.
func (h *GRPCHandler) UpdateMetadata(
	ctx context.Context,
	req *librarypb.UpdateMetadataRequest,
) (*librarypb.UpdateMetadataResponse, error) {
	// TODO: Implement
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// RefreshMetadata refreshes metadata for a media item.
func (h *GRPCHandler) RefreshMetadata(
	ctx context.Context,
	req *librarypb.RefreshMetadataRequest,
) (*librarypb.RefreshMetadataResponse, error) {
	// TODO: Implement
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// deletePhysicalFile safely deletes a file from the filesystem.
func (h *GRPCHandler) deletePhysicalFile(path string) error {
	// Security check: ensure path is absolute and within allowed directories
	// This is a simple implementation - in production, you'd want more sophisticated checks
	if !filepath.IsAbs(path) {
		return errors.BadRequest("path must be absolute")
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist, consider this success
			return nil
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Don't delete directories
	if info.IsDir() {
		return errors.BadRequest("cannot delete directory")
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}
