package interfaces

import (
	"context"
	"time"
)

// Cache defines a generic caching interface.
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in the cache with a TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Clear removes all values from the cache
	Clear(ctx context.Context) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// TTL returns the remaining TTL for a key
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// LayeredCache represents a multi-layer cache.
type LayeredCache interface {
	Cache

	// AddLayer adds a cache layer
	AddLayer(name string, cache Cache, priority int) error

	// RemoveLayer removes a cache layer
	RemoveLayer(name string) error

	// GetFromLayer gets a value from a specific layer
	GetFromLayer(ctx context.Context, layer, key string) (interface{}, error)
}

// CacheInvalidator handles cache invalidation across instances.
type CacheInvalidator interface {
	// InvalidateKey invalidates a specific key across all instances
	InvalidateKey(ctx context.Context, key string) error

	// InvalidatePattern invalidates keys matching a pattern
	InvalidatePattern(ctx context.Context, pattern string) error

	// Subscribe subscribes to invalidation events
	Subscribe(handler func(key string)) error
}
