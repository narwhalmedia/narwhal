package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides AES-256-GCM encryption for sensitive data
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the provided key
// The key will be hashed with SHA-256 to ensure it's exactly 32 bytes
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	
	// Hash the key to ensure it's exactly 32 bytes for AES-256
	hash := sha256.Sum256([]byte(key))
	
	return &Encryptor{
		key: hash[:],
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64 encoded ciphertext
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64 encoded ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce := data[:nonceSize]
	ciphertextBytes := data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}