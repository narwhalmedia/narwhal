package gorm

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	
	"github.com/narwhalmedia/narwhal/internal/config"
)

// NewDB creates a new database connection with proper configuration
func NewDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, func(), error) {
	// Create GORM logger adapter
	gormLog := newGormLogger(logger, cfg.Server.LogLevel == "debug")
	
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: gormLog,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true, // Prepare statements for better performance
	})
	if err != nil {
		return nil, nil, err
	}

	// Get underlying SQL database to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	// Run migrations
	if err := AutoMigrate(db); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup, nil
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	// TODO: Add models when implemented
	return nil
}

// gormLogger wraps zap logger for GORM
type gormLogger struct {
	logger *zap.Logger
	debug  bool
}

func newGormLogger(logger *zap.Logger, debug bool) gormlogger.Interface {
	return &gormLogger{
		logger: logger.Named("gorm"),
		debug:  debug,
	}
}

func (l *gormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Infof(msg, data...)
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Warnf(msg, data...)
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Errorf(msg, data...)
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && err != gorm.ErrRecordNotFound {
		l.logger.Error("sql error",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
		return
	}

	if l.debug {
		l.logger.Debug("sql trace",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	} else if elapsed > 200*time.Millisecond {
		l.logger.Warn("slow sql query",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	}
} 