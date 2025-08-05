package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "test-encryption-key",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncryptor(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, enc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, enc)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	enc, err := NewEncryptor("test-key-for-encryption")
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "API key",
			plaintext: "sk-1234567890abcdef",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "unicode",
			plaintext: "Hello 世界 🌍",
		},
		{
			name:      "long text",
			plaintext: "This is a very long API key that should still be encrypted and decrypted correctly without any issues whatsoever",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := enc.Encrypt(tt.plaintext)
			assert.NoError(t, err)

			if tt.plaintext == "" {
				assert.Empty(t, encrypted)
			} else {
				assert.NotEmpty(t, encrypted)
				assert.NotEqual(t, tt.plaintext, encrypted)
			}

			// Decrypt
			decrypted, err := enc.Decrypt(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	enc, err := NewEncryptor("test-key")
	require.NoError(t, err)

	plaintext := "test-api-key"

	// Encrypt the same plaintext multiple times
	encrypted1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Each encryption should produce different ciphertext due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same plaintext
	decrypted1, err := enc.Decrypt(encrypted1)
	require.NoError(t, err)

	decrypted2, err := enc.Decrypt(encrypted2)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDecryptInvalidData(t *testing.T) {
	enc, err := NewEncryptor("test-key")
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    bool
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!@#$",
			wantErr:    true,
		},
		{
			name:       "too short ciphertext",
			ciphertext: "dGVzdA==", // "test" in base64
			wantErr:    true,
		},
		{
			name:       "tampered ciphertext",
			ciphertext: "dGVzdHRlc3R0ZXN0dGVzdHRlc3R0ZXN0dGVzdHRlc3Q=",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := enc.Decrypt(tt.ciphertext)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, decrypted)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDifferentKeys(t *testing.T) {
	enc1, err := NewEncryptor("key1")
	require.NoError(t, err)

	enc2, err := NewEncryptor("key2")
	require.NoError(t, err)

	plaintext := "secret-api-key"

	// Encrypt with first encryptor
	encrypted, err := enc1.Encrypt(plaintext)
	require.NoError(t, err)

	// Try to decrypt with second encryptor (different key)
	decrypted, err := enc2.Decrypt(encrypted)
	assert.Error(t, err)
	assert.Empty(t, decrypted)

	// Decrypt with correct encryptor should work
	decrypted, err = enc1.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
