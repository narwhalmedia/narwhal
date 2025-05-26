package download

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
)

// FileValidator implements file validation functionality
type FileValidator struct {
	logger *zap.Logger
}

// NewFileValidator creates a new file validator
func NewFileValidator(logger *zap.Logger) *FileValidator {
	return &FileValidator{
		logger: logger.Named("file-validator"),
	}
}

// ValidateChecksum validates a file against a checksum
func (v *FileValidator) ValidateChecksum(filepath string, expectedChecksum string, checksumType string) error {
	actualChecksum, err := v.CalculateChecksum(filepath, checksumType)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Normalize checksums for comparison
	expected := strings.ToLower(strings.TrimSpace(expectedChecksum))
	actual := strings.ToLower(strings.TrimSpace(actualChecksum))

	if expected != actual {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	v.logger.Info("checksum validation passed",
		zap.String("file", filepath),
		zap.String("type", checksumType),
		zap.String("checksum", actual),
	)

	return nil
}

// CalculateChecksum calculates the checksum of a file
func (v *FileValidator) CalculateChecksum(filepath string, checksumType string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var hasher hash.Hash
	switch strings.ToLower(checksumType) {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	case "sha512":
		hasher = sha512.New()
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checksumType)
	}

	// Calculate checksum with progress tracking
	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := hasher.Write(buf[:n]); err != nil {
			return "", fmt.Errorf("failed to update hash: %w", err)
		}
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	
	v.logger.Debug("calculated checksum",
		zap.String("file", filepath),
		zap.String("type", checksumType),
		zap.String("checksum", checksum),
	)

	return checksum, nil
}

// ValidateFile performs basic file validation
func (v *FileValidator) ValidateFile(filepath string) error {
	stat, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	if stat.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// Check if file is readable
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	file.Close()

	return nil
}

// GetFileInfo returns information about a file
func (v *FileValidator) GetFileInfo(filepath string) (FileInfo, error) {
	stat, err := os.Stat(filepath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to stat file: %w", err)
	}

	info := FileInfo{
		Path:    filepath,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
	}

	// Calculate common checksums
	if md5sum, err := v.CalculateChecksum(filepath, "md5"); err == nil {
		info.MD5 = md5sum
	}

	if sha256sum, err := v.CalculateChecksum(filepath, "sha256"); err == nil {
		info.SHA256 = sha256sum
	}

	return info, nil
}

// FileInfo contains file information
type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	MD5     string
	SHA256  string
}

// ParFileValidator validates files using PAR2 files
type ParFileValidator struct {
	validator *FileValidator
	logger    *zap.Logger
}

// NewParFileValidator creates a new PAR file validator
func NewParFileValidator(logger *zap.Logger) *ParFileValidator {
	return &ParFileValidator{
		validator: NewFileValidator(logger),
		logger:    logger.Named("par-validator"),
	}
}

// ValidateWithPar validates files using PAR2 files
func (p *ParFileValidator) ValidateWithPar(files []string, parFile string) error {
	// This would require par2cmdline or similar tool
	// For now, this is a placeholder
	p.logger.Warn("PAR2 validation not yet implemented")
	return nil
}

// RepairWithPar attempts to repair files using PAR2 files
func (p *ParFileValidator) RepairWithPar(files []string, parFile string) error {
	// This would require par2cmdline or similar tool
	// For now, this is a placeholder
	p.logger.Warn("PAR2 repair not yet implemented")
	return nil
}