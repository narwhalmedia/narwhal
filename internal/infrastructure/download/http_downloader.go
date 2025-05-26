package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/download"
)

// HTTPDownloader implements HTTP download with resume support
type HTTPDownloader struct {
	client *http.Client
	logger *zap.Logger
}

// NewHTTPDownloader creates a new HTTP downloader
func NewHTTPDownloader(logger *zap.Logger) *HTTPDownloader {
	return &HTTPDownloader{
		client: &http.Client{
			Timeout: 0, // No timeout for downloads
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		logger: logger.Named("http-downloader"),
	}
}

// Download starts downloading from the source
func (d *HTTPDownloader) Download(ctx context.Context, source string, destination io.Writer, progress chan<- download.Progress) error {
	return d.downloadWithRange(ctx, source, destination, 0, progress)
}

// Resume resumes a download from the given offset
func (d *HTTPDownloader) Resume(ctx context.Context, source string, destination io.WriteSeeker, offset int64, progress chan<- download.Progress) error {
	// Seek to the offset
	if _, err := destination.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	return d.downloadWithRange(ctx, source, destination, offset, progress)
}

// GetMetadata fetches metadata about the download
func (d *HTTPDownloader) GetMetadata(ctx context.Context, source string) (*download.Metadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, source, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	metadata := &download.Metadata{
		ContentType: resp.Header.Get("Content-Type"),
		Headers:     make(map[string]string),
	}

	// Extract relevant headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			metadata.Headers[key] = values[0]
		}
	}

	// Try to get filename from Content-Disposition
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		// Simple extraction, could be improved with proper parsing
		if filename := extractFilename(cd); filename != "" {
			metadata.FileName = filename
		}
	}

	return metadata, nil
}

// downloadWithRange performs the actual download with optional range support
func (d *HTTPDownloader) downloadWithRange(ctx context.Context, source string, destination io.Writer, offset int64, progress chan<- download.Progress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add range header for resume
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	// Set user agent
	req.Header.Set("User-Agent", "Narwhal/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	expectedStatus := http.StatusOK
	if offset > 0 {
		expectedStatus = http.StatusPartialContent
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get total size
	totalSize := int64(0)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			totalSize = size + offset
		}
	} else if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		// Parse Content-Range header (e.g., "bytes 200-1023/1024")
		if size := parseContentRange(contentRange); size > 0 {
			totalSize = size
		}
	}

	// Create progress reporter
	progressReporter := &progressWriter{
		writer:     destination,
		progress:   progress,
		offset:     offset,
		totalSize:  totalSize,
		lastReport: time.Now(),
		logger:     d.logger,
	}

	// Start download with progress tracking
	bytesWritten, err := io.CopyBuffer(progressReporter, resp.Body, make([]byte, 32*1024))
	if err != nil && err != context.Canceled {
		return fmt.Errorf("download failed after %d bytes: %w", bytesWritten, err)
	}

	// Send final progress
	if progress != nil {
		finalProgress := download.Progress{
			BytesDownloaded: offset + bytesWritten,
			TotalBytes:      totalSize,
			Speed:           0,
		}
		select {
		case progress <- finalProgress:
		case <-ctx.Done():
		}
	}

	return nil
}

// progressWriter wraps an io.Writer to report progress
type progressWriter struct {
	writer          io.Writer
	progress        chan<- download.Progress
	offset          int64
	totalSize       int64
	bytesWritten    int64
	lastReport      time.Time
	lastBytes       int64
	reportInterval  time.Duration
	logger          *zap.Logger
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.bytesWritten += int64(n)

	// Report progress periodically
	if pw.progress != nil && time.Since(pw.lastReport) >= time.Second {
		elapsed := time.Since(pw.lastReport).Seconds()
		bytesInInterval := pw.bytesWritten - pw.lastBytes
		speed := int64(float64(bytesInInterval) / elapsed)

		prog := download.Progress{
			BytesDownloaded: pw.offset + pw.bytesWritten,
			TotalBytes:      pw.totalSize,
			Speed:           speed,
		}

		// Calculate ETA
		if speed > 0 && pw.totalSize > 0 {
			remainingBytes := pw.totalSize - (pw.offset + pw.bytesWritten)
			prog.ETA = time.Duration(remainingBytes/speed) * time.Second
		}

		select {
		case pw.progress <- prog:
		default:
			// Don't block if progress channel is full
		}

		pw.lastReport = time.Now()
		pw.lastBytes = pw.bytesWritten
	}

	return n, nil
}

// extractFilename attempts to extract filename from Content-Disposition header
func extractFilename(contentDisposition string) string {
	// Simple implementation - could be improved with proper parsing
	const filenamePrefix = "filename="
	if idx := indexOf(contentDisposition, filenamePrefix); idx != -1 {
		start := idx + len(filenamePrefix)
		end := indexOf(contentDisposition[start:], ";")
		if end == -1 {
			end = len(contentDisposition[start:])
		}
		filename := contentDisposition[start : start+end]
		// Remove quotes if present
		filename = trimQuotes(filename)
		return filename
	}
	return ""
}

// parseContentRange parses Content-Range header to get total size
func parseContentRange(contentRange string) int64 {
	// Parse "bytes 200-1023/1024" format
	if idx := indexOf(contentRange, "/"); idx != -1 {
		sizeStr := contentRange[idx+1:]
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			return size
		}
	}
	return 0
}

// Helper functions
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}