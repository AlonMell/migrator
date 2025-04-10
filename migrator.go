package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/AlonMell/grovelog"
	"github.com/AlonMell/migrator/internal/executor"
	"github.com/AlonMell/migrator/internal/fetcher"
	"github.com/AlonMell/migrator/internal/parser"
	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

// Migrator handles database migrations
type Migrator struct {
	executor executor.Interface
	fetcher  fetcher.Interface
	logger   types.Logger
	path     string
}

// Config holds configuration for Migrator
type Config struct {
	DB       *sql.DB
	Logger   *slog.Logger
	Path     string
	Table    string
	MajorVer int
	MinorVer int
}

// DefaultLogger returns default logger for logging in migrator
func DefaultLogger() *slog.Logger {
	opts := grovelog.NewOptions(slog.LevelInfo, "", grovelog.Color)
	return grovelog.NewLogger(os.Stdout, opts)
}

// New creates a new Migrator instance with the given configuration
func New(config Config) *Migrator {
	log := config.Logger
	if log == nil {
		log = DefaultLogger()
	}

	parser := parser.New()

	current := version.New(0, 0, 0)
	target := version.New(config.MajorVer, config.MinorVer, 0)

	var migrationType types.MigrationType
	if target.CompareTo(current) >= 0 {
		migrationType = types.MigrationUp
	} else {
		migrationType = types.MigrationDown
	}

	executor := executor.New(config.DB, config.Table, log, parser, config.Path)
	fetcher := fetcher.New(log, parser, migrationType, current, target)

	return &Migrator{
		executor: executor,
		fetcher:  fetcher,
		logger:   log,
		path:     config.Path,
	}
}

// Migrate performs the migration process
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.InfoContext(ctx, "Starting migration process")

	if err := m.executor.InitializeTable(ctx); err != nil {
		return fmt.Errorf("initializing migration: %w", err)
	}

	// Get files to execute
	files, err := m.fetcher.GetFilesToExecute(ctx, m.path)
	if err != nil {
		return fmt.Errorf("getting files to execute: %w", err)
	}

	if len(files) == 0 {
		m.logger.InfoContext(ctx, "No migration files to execute - database is up to date")
		return nil
	}

	// Execute each file
	for _, file := range files {
		if err := m.executor.ExecuteFile(ctx, file); err != nil {
			return fmt.Errorf("executing file %s: %w", file, err)
		}
	}

	m.logger.InfoContext(ctx, "Migration completed successfully")
	return nil
}
