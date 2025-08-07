package constants

import "time"

const (
	// Database and conversion constants.
	SecondsToMinutes = 60

	// Pagination constants.
	DefaultPageSize = 50
	MaxPageSize     = 200

	// Cache constants.
	CacheTTL = 5 * time.Minute
)
