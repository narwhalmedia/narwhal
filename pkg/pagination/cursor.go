package pagination

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Cursor represents pagination cursor data
type Cursor struct {
	Offset    int       `json:"offset"`
	Timestamp time.Time `json:"timestamp"`
	SortField string    `json:"sort_field,omitempty"`
	SortValue string    `json:"sort_value,omitempty"`
}

// CursorEncoder handles cursor encryption/decryption
type CursorEncoder struct {
	cipher cipher.Block
}

// NewCursorEncoder creates a new cursor encoder with the given key
func NewCursorEncoder(key []byte) (*CursorEncoder, error) {
	// Ensure key is exactly 32 bytes for AES-256
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &CursorEncoder{
		cipher: block,
	}, nil
}

// EncodeCursor encrypts and encodes a cursor to a base64 string
func (e *CursorEncoder) EncodeCursor(cursor *Cursor) (string, error) {
	// Marshal cursor to JSON
	plaintext, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %w", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(e.cipher)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Encode to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecodeCursor decrypts and decodes a cursor from a base64 string
func (e *CursorEncoder) DecodeCursor(encoded string) (*Cursor, error) {
	// Decode from base64
	ciphertext, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(e.cipher)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Unmarshal cursor
	var cursor Cursor
	if err := json.Unmarshal(plaintext, &cursor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %w", err)
	}

	return &cursor, nil
}

// CreateOffsetCursor creates a simple offset-based cursor
func CreateOffsetCursor(offset int) *Cursor {
	return &Cursor{
		Offset:    offset,
		Timestamp: time.Now(),
	}
}

// CreateKeyCursor creates a cursor for keyset pagination
func CreateKeyCursor(offset int, sortField, sortValue string) *Cursor {
	return &Cursor{
		Offset:    offset,
		Timestamp: time.Now(),
		SortField: sortField,
		SortValue: sortValue,
	}
}

// IsExpired checks if the cursor is older than the given duration
func (c *Cursor) IsExpired(maxAge time.Duration) bool {
	return time.Since(c.Timestamp) > maxAge
}

// PaginationParams contains common pagination parameters
type PaginationParams struct {
	PageSize  int32
	PageToken string
}

// Response contains pagination response metadata
type Response struct {
	NextPageToken string
	PrevPageToken string
	TotalItems    int32
	HasMore       bool
}

// CalculateOffset calculates the offset from a page token
func CalculateOffset(encoder *CursorEncoder, pageToken string, defaultOffset int) (int, error) {
	if pageToken == "" {
		return defaultOffset, nil
	}

	cursor, err := encoder.DecodeCursor(pageToken)
	if err != nil {
		return 0, fmt.Errorf("invalid page token: %w", err)
	}

	// Check if cursor is expired (24 hours)
	if cursor.IsExpired(24 * time.Hour) {
		return 0, fmt.Errorf("page token expired")
	}

	return cursor.Offset, nil
}

// GenerateNextPageToken generates the next page token
func GenerateNextPageToken(encoder *CursorEncoder, currentOffset, pageSize, totalItems int) (string, error) {
	nextOffset := currentOffset + pageSize
	if nextOffset >= totalItems {
		return "", nil // No more pages
	}

	cursor := CreateOffsetCursor(nextOffset)
	return encoder.EncodeCursor(cursor)
}

// GeneratePrevPageToken generates the previous page token
func GeneratePrevPageToken(encoder *CursorEncoder, currentOffset, pageSize int) (string, error) {
	if currentOffset <= 0 {
		return "", nil // No previous page
	}

	prevOffset := currentOffset - pageSize
	if prevOffset < 0 {
		prevOffset = 0
	}

	cursor := CreateOffsetCursor(prevOffset)
	return encoder.EncodeCursor(cursor)
}
