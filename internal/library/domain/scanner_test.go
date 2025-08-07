package domain_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/logger"
)

type ScannerTestSuite struct {
	suite.Suite

	ctx     context.Context
	scanner *domain.Scanner
	tempDir string
}

func (suite *ScannerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.scanner = domain.NewScanner(logger.NewNoopLogger())

	// Create temporary directory for test files
	var err error
	suite.tempDir, err = os.MkdirTemp("", "scanner_test_*")
	suite.Require().NoError(err)
}

func (suite *ScannerTestSuite) TearDownTest() {
	// Clean up temporary directory
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *ScannerTestSuite) createTestFile(name string, content string) string {
	path := filepath.Join(suite.tempDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	suite.Require().NoError(err)
	return path
}

func (suite *ScannerTestSuite) createTestDir(name string) string {
	path := filepath.Join(suite.tempDir, name)
	err := os.MkdirAll(path, 0755)
	suite.Require().NoError(err)
	return path
}

func (suite *ScannerTestSuite) TestScanPath_MovieLibrary() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: suite.tempDir,
	}

	// Create test movie files
	suite.createTestFile("movie1.mp4", "fake video content")
	suite.createTestFile("movie2.mkv", "fake video content")
	suite.createTestFile("movie3.avi", "fake video content")
	suite.createTestFile("readme.txt", "not a video")

	// Act
	result, err := suite.scanner.ScanPath(suite.ctx, library)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Equal(library.ID, result.LibraryID)
	suite.Equal("completed", result.Status)
	suite.Equal(3, result.FilesFound)
	suite.Equal(0, result.Errors)
	suite.GreaterOrEqual(result.Duration, 0)
}

func (suite *ScannerTestSuite) TestScanPath_TVLibrary() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "tv",
		Path: suite.tempDir,
	}

	// Create test TV show structure
	suite.createTestDir("Test Show")
	suite.createTestDir(filepath.Join("Test Show", "Season 1"))
	suite.createTestDir(filepath.Join("Test Show", "Season 2"))

	// Create episode files
	suite.createTestFile(filepath.Join("Test Show", "Season 1", "S01E01.mp4"), "fake video")
	suite.createTestFile(filepath.Join("Test Show", "Season 1", "S01E02.mp4"), "fake video")
	suite.createTestFile(filepath.Join("Test Show", "Season 2", "S02E01.mp4"), "fake video")

	// Act
	result, err := suite.scanner.ScanPath(suite.ctx, library)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Equal(3, result.FilesFound)
	suite.Equal("completed", result.Status)
}

func (suite *ScannerTestSuite) TestScanPath_EmptyDirectory() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: suite.tempDir,
	}

	// Act
	result, err := suite.scanner.ScanPath(suite.ctx, library)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Equal(0, result.FilesFound)
	suite.Equal("completed", result.Status)
}

func (suite *ScannerTestSuite) TestScanPath_NonExistentPath() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: "/non/existent/path",
	}

	// Act
	result, err := suite.scanner.ScanPath(suite.ctx, library)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Equal("failed", result.Status)
	suite.Equal(0, result.FilesFound)
	suite.Positive(result.Errors)
}

func (suite *ScannerTestSuite) TestScanPath_ContextCancellation() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: suite.tempDir,
	}

	// Create many files to ensure scan takes some time
	for i := range 100 {
		suite.createTestFile(fmt.Sprintf("movie%d.mp4", i), "fake video content")
	}

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(suite.ctx)

	// Start scan in goroutine
	resultChan := make(chan *domain.ScanResult)
	errChan := make(chan error)

	go func() {
		result, err := suite.scanner.ScanPath(ctx, library)
		resultChan <- result
		errChan <- err
	}()

	// Cancel context after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Act
	result := <-resultChan
	err := <-errChan

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(result)
	// Status might be "completed" if scan finished before cancellation
	suite.Contains([]string{"completed", "cancelled"}, result.Status)
}

func (suite *ScannerTestSuite) TestIsVideoFile() {
	// Test valid video extensions
	validExtensions := []string{
		".mp4", ".MP4",
		".mkv", ".MKV",
		".avi", ".AVI",
		".mov", ".MOV",
		".wmv", ".WMV",
		".flv", ".FLV",
		".webm", ".WEBM",
		".m4v", ".M4V",
		".mpg", ".MPG",
		".mpeg", ".MPEG",
	}

	for _, ext := range validExtensions {
		suite.True(suite.scanner.IsVideoFile("test"+ext), "Extension %s should be valid", ext)
	}

	// Test invalid extensions
	invalidExtensions := []string{
		".txt", ".doc", ".pdf", ".jpg", ".png", ".mp3", ".zip",
	}

	for _, ext := range invalidExtensions {
		suite.False(suite.scanner.IsVideoFile("test"+ext), "Extension %s should be invalid", ext)
	}
}

func (suite *ScannerTestSuite) TestIsScanning() {
	// Arrange
	libraryID := uuid.New().String()

	// Initially should not be scanning
	suite.False(suite.scanner.IsScanning(libraryID))

	// Set scanning
	suite.scanner.SetScanning(libraryID, true)
	suite.True(suite.scanner.IsScanning(libraryID))

	// Unset scanning
	suite.scanner.SetScanning(libraryID, false)
	suite.False(suite.scanner.IsScanning(libraryID))
}

func (suite *ScannerTestSuite) TestConcurrentScanning() {
	// Test that multiple libraries can be scanned concurrently
	library1 := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: suite.tempDir,
	}

	library2Dir, err := os.MkdirTemp("", "scanner_test2_*")
	suite.Require().NoError(err)
	defer os.RemoveAll(library2Dir)

	library2 := &domain.Library{
		ID:   uuid.New(),
		Type: "tv",
		Path: library2Dir,
	}

	// Create test files in both directories
	suite.createTestFile("movie1.mp4", "fake video")
	os.WriteFile(filepath.Join(library2Dir, "show.mp4"), []byte("fake video"), 0644)

	// Act - scan both libraries concurrently
	result1Chan := make(chan *domain.ScanResult)
	result2Chan := make(chan *domain.ScanResult)

	go func() {
		result, _ := suite.scanner.ScanPath(suite.ctx, library1)
		result1Chan <- result
	}()

	go func() {
		result, _ := suite.scanner.ScanPath(suite.ctx, library2)
		result2Chan <- result
	}()

	// Assert
	result1 := <-result1Chan
	result2 := <-result2Chan

	suite.Equal("completed", result1.Status)
	suite.Equal("completed", result2.Status)
	suite.Equal(1, result1.FilesFound)
	suite.Equal(1, result2.FilesFound)
}

func TestScannerTestSuite(t *testing.T) {
	suite.Run(t, new(ScannerTestSuite))
}
