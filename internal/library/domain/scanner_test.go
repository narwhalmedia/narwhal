package domain_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), library.ID, result.LibraryID)
	assert.Equal(suite.T(), "completed", result.Status)
	assert.Equal(suite.T(), 3, result.FilesFound)
	assert.Equal(suite.T(), 0, result.Errors)
	assert.True(suite.T(), result.Duration >= 0)
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 3, result.FilesFound)
	assert.Equal(suite.T(), "completed", result.Status)
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0, result.FilesFound)
	assert.Equal(suite.T(), "completed", result.Status)
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "failed", result.Status)
	assert.Equal(suite.T(), 0, result.FilesFound)
	assert.True(suite.T(), result.Errors > 0)
}

func (suite *ScannerTestSuite) TestScanPath_ContextCancellation() {
	// Arrange
	library := &domain.Library{
		ID:   uuid.New(),
		Type: "movie",
		Path: suite.tempDir,
	}
	
	// Create many files to ensure scan takes some time
	for i := 0; i < 100; i++ {
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	// Status might be "completed" if scan finished before cancellation
	assert.Contains(suite.T(), []string{"completed", "cancelled"}, result.Status)
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
		assert.True(suite.T(), suite.scanner.IsVideoFile("test"+ext), "Extension %s should be valid", ext)
	}
	
	// Test invalid extensions
	invalidExtensions := []string{
		".txt", ".doc", ".pdf", ".jpg", ".png", ".mp3", ".zip",
	}
	
	for _, ext := range invalidExtensions {
		assert.False(suite.T(), suite.scanner.IsVideoFile("test"+ext), "Extension %s should be invalid", ext)
	}
}

func (suite *ScannerTestSuite) TestIsScanning() {
	// Arrange
	libraryID := uuid.New().String()
	
	// Initially should not be scanning
	assert.False(suite.T(), suite.scanner.IsScanning(libraryID))
	
	// Set scanning
	suite.scanner.SetScanning(libraryID, true)
	assert.True(suite.T(), suite.scanner.IsScanning(libraryID))
	
	// Unset scanning
	suite.scanner.SetScanning(libraryID, false)
	assert.False(suite.T(), suite.scanner.IsScanning(libraryID))
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
	
	assert.Equal(suite.T(), "completed", result1.Status)
	assert.Equal(suite.T(), "completed", result2.Status)
	assert.Equal(suite.T(), 1, result1.FilesFound)
	assert.Equal(suite.T(), 1, result2.FilesFound)
}

func TestScannerTestSuite(t *testing.T) {
	suite.Run(t, new(ScannerTestSuite))
}