package utils

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCacheMiss = errors.New("cache miss")
	ErrExpired   = errors.New("cache entry expired")
)

// CacheEntry represents a cached value with expiration.
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// InMemoryCache is a simple in-memory cache implementation.
type InMemoryCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		entries: make(map[string]*CacheEntry),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache.
func (c *InMemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, ErrCacheMiss
	}

	if time.Now().After(entry.Expiration) {
		return nil, ErrExpired
	}

	return entry.Value, nil
}

// Set stores a value in the cache with a TTL.
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from the cache.
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

// Clear removes all values from the cache.
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	return nil
}

// Exists checks if a key exists in the cache.
func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.Expiration) {
		return false, nil
	}

	return true, nil
}

// TTL returns the remaining TTL for a key.
func (c *InMemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return 0, ErrCacheMiss
	}

	ttl := time.Until(entry.Expiration)
	if ttl < 0 {
		return 0, ErrExpired
	}

	return ttl, nil
}

// cleanup periodically removes expired entries.
func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.Expiration) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
