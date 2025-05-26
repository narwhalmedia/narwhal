package gorm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/narwhalmedia/narwhal/internal/application/media"
	"gorm.io/gorm"
)

// UnitOfWork implements the Unit of Work pattern for GORM
type UnitOfWork struct {
	db *gorm.DB
}

// NewUnitOfWork creates a new GORM-based unit of work
func NewUnitOfWork(db *gorm.DB) media.UnitOfWork {
	return &UnitOfWork{db: db}
}

// Begin starts a new transaction
func (u *UnitOfWork) Begin(ctx context.Context) (media.Transaction, error) {
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("begin transaction: %w", tx.Error)
	}

	return &gormTransaction{
		tx:  tx,
		ctx: ctx,
	}, nil
}

// gormTransaction implements the Transaction interface for GORM
type gormTransaction struct {
	tx  *gorm.DB
	ctx context.Context
}

// Commit commits the transaction
func (t *gormTransaction) Commit() error {
	if err := t.tx.Commit().Error; err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (t *gormTransaction) Rollback() error {
	if err := t.tx.Rollback().Error; err != nil {
		// Check if already rolled back
		if err == sql.ErrTxDone {
			return nil
		}
		return fmt.Errorf("rollback transaction: %w", err)
	}
	return nil
}

// Context returns the transaction context
func (t *gormTransaction) Context() context.Context {
	return t.ctx
}

// TransactionalRepositories provides repositories that use the same transaction
type TransactionalRepositories struct {
	tx *gorm.DB
}

// NewTransactionalRepositories creates repositories that share a transaction
func NewTransactionalRepositories(tx *gorm.DB) *TransactionalRepositories {
	return &TransactionalRepositories{tx: tx}
}

// SeriesRepository returns a series repository using the transaction
func (r *TransactionalRepositories) SeriesRepository() *SeriesRepository {
	return &SeriesRepository{db: r.tx}
}

// MovieRepository returns a movie repository using the transaction
func (r *TransactionalRepositories) MovieRepository() *MovieRepository {
	return &MovieRepository{db: r.tx}
}

// WithTransaction executes a function within a transaction
func WithTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}