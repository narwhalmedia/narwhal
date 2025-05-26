package media

import (
	"context"
)

// UnitOfWork defines the interface for managing transactions across repositories
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	// Commit commits the transaction
	Commit() error
	// Rollback rolls back the transaction
	Rollback() error
	// Context returns the transaction context
	Context() context.Context
}