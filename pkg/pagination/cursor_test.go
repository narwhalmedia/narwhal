package pagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorEncoder(t *testing.T) {
	// Create encoder with a test key
	key := []byte("test-key-for-pagination-12345678") // 32 bytes
	encoder, err := NewCursorEncoder(key)
	require.NoError(t, err)

	t.Run("encode and decode offset cursor", func(t *testing.T) {
		// Create cursor
		original := CreateOffsetCursor(100)

		// Encode
		encoded, err := encoder.EncodeCursor(original)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		// Decode
		decoded, err := encoder.DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.Offset, decoded.Offset)
		assert.WithinDuration(t, original.Timestamp, decoded.Timestamp, time.Second)
	})

	t.Run("encode and decode key cursor", func(t *testing.T) {
		// Create cursor with sort info
		original := CreateKeyCursor(50, "created_at", "2024-01-01T00:00:00Z")

		// Encode
		encoded, err := encoder.EncodeCursor(original)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		// Decode
		decoded, err := encoder.DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.Offset, decoded.Offset)
		assert.Equal(t, original.SortField, decoded.SortField)
		assert.Equal(t, original.SortValue, decoded.SortValue)
	})

	t.Run("invalid key length", func(t *testing.T) {
		_, err := NewCursorEncoder([]byte("short-key"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("invalid encoded cursor", func(t *testing.T) {
		_, err := encoder.DecodeCursor("invalid-base64")
		require.Error(t, err)
	})

	t.Run("cursor expiration", func(t *testing.T) {
		cursor := &Cursor{
			Offset:    10,
			Timestamp: time.Now().Add(-25 * time.Hour),
		}

		assert.True(t, cursor.IsExpired(24*time.Hour))
		assert.False(t, cursor.IsExpired(48*time.Hour))
	})
}

func TestPaginationHelpers(t *testing.T) {
	key := []byte("test-key-for-pagination-12345678")
	encoder, err := NewCursorEncoder(key)
	require.NoError(t, err)

	t.Run("calculate offset from token", func(t *testing.T) {
		// Create and encode a cursor
		cursor := CreateOffsetCursor(50)
		token, err := encoder.EncodeCursor(cursor)
		require.NoError(t, err)

		// Calculate offset
		offset, err := CalculateOffset(encoder, token, 0)
		require.NoError(t, err)
		assert.Equal(t, 50, offset)
	})

	t.Run("calculate offset with empty token", func(t *testing.T) {
		offset, err := CalculateOffset(encoder, "", 10)
		require.NoError(t, err)
		assert.Equal(t, 10, offset)
	})

	t.Run("calculate offset with expired token", func(t *testing.T) {
		// Create expired cursor
		cursor := &Cursor{
			Offset:    50,
			Timestamp: time.Now().Add(-25 * time.Hour),
		}
		token, err := encoder.EncodeCursor(cursor)
		require.NoError(t, err)

		// Should fail due to expiration
		_, err = CalculateOffset(encoder, token, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("generate next page token", func(t *testing.T) {
		token, err := GenerateNextPageToken(encoder, 0, 10, 100)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Decode and verify
		cursor, err := encoder.DecodeCursor(token)
		require.NoError(t, err)
		assert.Equal(t, 10, cursor.Offset)
	})

	t.Run("generate next page token at end", func(t *testing.T) {
		token, err := GenerateNextPageToken(encoder, 90, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, token) // No more pages
	})

	t.Run("generate previous page token", func(t *testing.T) {
		token, err := GeneratePrevPageToken(encoder, 20, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Decode and verify
		cursor, err := encoder.DecodeCursor(token)
		require.NoError(t, err)
		assert.Equal(t, 10, cursor.Offset)
	})

	t.Run("generate previous page token at start", func(t *testing.T) {
		token, err := GeneratePrevPageToken(encoder, 0, 10)
		require.NoError(t, err)
		assert.Empty(t, token) // No previous page
	})
}
