package domain

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// MediaFile represents a discovered media file
type MediaFile struct {
	Path     string
	Size     int64
	Modified time.Time
}

// Scanner handles directory scanning for media files
type Scanner struct {
	logger       interfaces.Logger
	scanningMu   sync.RWMutex
	scanningLibs map[string]bool
}

// NewScanner creates a new scanner
func NewScanner(logger interfaces.Logger) *Scanner {
	return &Scanner{
		logger:       logger,
		scanningLibs: make(map[string]bool),
	}
}

// IsScanning checks if a library is currently being scanned
func (s *Scanner) IsScanning(libraryID string) bool {
	s.scanningMu.RLock()
	defer s.scanningMu.RUnlock()
	return s.scanningLibs[libraryID]
}

// SetScanning sets the scanning state for a library
func (s *Scanner) SetScanning(libraryID string, scanning bool) {
	s.scanningMu.Lock()
	defer s.scanningMu.Unlock()
	if scanning {
		s.scanningLibs[libraryID] = true
	} else {
		delete(s.scanningLibs, libraryID)
	}
}

// ScanDirectory scans a directory for media files
func (s *Scanner) ScanDirectory(path string, mediaType string) ([]*MediaFile, error) {
	var files []*MediaFile
	extensions := getMediaExtensions(models.MediaType(mediaType))

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Warn("Error accessing path",
				interfaces.String("path", filePath),
				interfaces.Error(err))
			return nil // Continue scanning
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file has a valid media extension
		ext := strings.ToLower(filepath.Ext(filePath))
		if contains(extensions, ext) {
			files = append(files, &MediaFile{
				Path:     filePath,
				Size:     info.Size(),
				Modified: info.ModTime(),
			})
		}

		return nil
	})

	return files, err
}

// getMediaExtensions returns valid file extensions for a media type
func getMediaExtensions(mediaType models.MediaType) []string {
	switch mediaType {
	case models.MediaTypeMovie, models.MediaTypeSeries, models.MediaTypeTV:
		return []string{
			".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm",
			".m4v", ".mpg", ".mpeg", ".3gp", ".ogv", ".ts", ".vob",
		}
	case models.MediaTypeMusic:
		return []string{
			".mp3", ".flac", ".aac", ".ogg", ".wma", ".m4a", ".opus",
			".wav", ".ape", ".alac", ".dsd", ".dsf",
		}
	default:
		return []string{}
	}
}

// contains checks if a slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ExtractTitle extracts a clean title from a file path
func ExtractTitle(path string) string {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	title := strings.TrimSuffix(filename, ext)
	
	// Regular expressions for common patterns
	yearPattern := regexp.MustCompile(`\s*[\(\[]?\d{4}[\)\]]?\s*`)
	qualityPattern := regexp.MustCompile(`\s*[\(\[]?(1080p|720p|480p|2160p|4K|BluRay|BRRip|WEBRip|HDTV|DVDRip|WEB-DL|x264|x265|h264|h265|HEVC)[\)\]]?.*`)
	releaseGroupPattern := regexp.MustCompile(`-[A-Za-z0-9]+$`)
	
	// Clean up common separators
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	
	// Remove year
	title = yearPattern.ReplaceAllString(title, " ")
	
	// Remove quality and everything after
	title = qualityPattern.ReplaceAllString(title, "")
	
	// Remove release group
	title = releaseGroupPattern.ReplaceAllString(title, "")
	
	// Clean up multiple spaces
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	
	// Trim whitespace
	title = strings.TrimSpace(title)
	
	// Capitalize words
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	
	return strings.Join(words, " ")
}

// ScanPath scans a library path and returns scan results
func (s *Scanner) ScanPath(ctx context.Context, library *Library) (*ScanResult, error) {
	libraryID := library.ID.String()
	
	// Check if already scanning
	if s.IsScanning(libraryID) {
		return &ScanResult{
			LibraryID: library.ID,
			Status:    "already_scanning",
			Errors:    1,
			ErrorMessage: "Library is already being scanned",
		}, nil
	}
	
	// Set scanning state
	s.SetScanning(libraryID, true)
	defer s.SetScanning(libraryID, false)
	
	result := &ScanResult{
		ID:        uuid.New(),
		LibraryID: library.ID,
		StartedAt: time.Now(),
		Status:    "scanning",
	}
	
	// Check if path exists
	if _, err := os.Stat(library.Path); os.IsNotExist(err) {
		result.Status = "failed"
		result.Errors = 1
		result.ErrorMessage = "Library path does not exist"
		result.CompletedAt = &[]time.Time{time.Now()}[0]
		result.Duration = time.Since(result.StartedAt).Milliseconds()
		return result, nil
	}
	
	// Scan for files
	files, err := s.ScanDirectory(library.Path, library.Type)
	if err != nil {
		result.Status = "failed"
		result.Errors = 1
		result.ErrorMessage = err.Error()
		result.CompletedAt = &[]time.Time{time.Now()}[0]
		result.Duration = time.Since(result.StartedAt).Milliseconds()
		return result, nil
	}
	
	// Count files
	result.FilesFound = len(files)
	result.FilesScanned = len(files)
	
	// Complete scan
	completedAt := time.Now()
	result.CompletedAt = &completedAt
	result.Status = "completed"
	result.Duration = time.Since(result.StartedAt).Milliseconds()
	
	s.logger.Info("Library scan completed",
		interfaces.String("library_id", libraryID),
		interfaces.Int("files_found", result.FilesFound))
	
	return result, nil
}

// IsVideoFile checks if a file has a video extension
func (s *Scanner) IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExtensions := []string{
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm",
		".m4v", ".mpg", ".mpeg", ".3gp", ".ogv", ".ts", ".vob",
	}
	return contains(videoExtensions, ext)
}