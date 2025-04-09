package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/AlonMell/grovelog"
)

// MigrationType indicates the type of migration operation
type MigrationType int

const (
	// MigrationUp represents an up migration
	MigrationUp MigrationType = iota
	// MigrationDown represents a down migration
	MigrationDown
)

type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// Migrator handles database migrations
type Migrator struct {
	db      *sql.DB
	logger  *slog.Logger
	target  *Version
	current *Version
	path    string
	table   string
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
	return &Migrator{
		db:      config.DB,
		path:    config.Path,
		table:   config.Table,
		logger:  config.Logger,
		target:  NewVersion(config.MajorVer, config.MinorVer, 0),
		current: NewVersion(0, 0, 0),
	}
}

// Migrate performs the migration process
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.InfoContext(ctx, "Starting migration process")

	if err := m.initialize(ctx); err != nil {
		return err
	}

	migrationType := m.determineMigrationType()

	filesToExecute, err := m.getFilesToExecute(migrationType)
	if err != nil {
		return fmt.Errorf("getting files to execute: %w", err)
	}

	if len(filesToExecute) == 0 {
		m.logger.InfoContext(ctx, "No migration files to execute - database is up to date")
		return nil
	}

	for _, file := range filesToExecute {
		if err := m.executeFile(ctx, file); err != nil {
			return fmt.Errorf("executing file %s: %w", file, err)
		}
	}

	m.logger.InfoContext(ctx, "Migration completed successfully")
	return nil
}

// initialize checks if migration table exists and creates it if necessary
func (m *Migrator) initialize(ctx context.Context) error {
	exists, err := m.tableExists(ctx)
	if err != nil {
		return fmt.Errorf("checking migration table: %w", err)
	}

	if !exists {
		m.logger.InfoContext(ctx, "Migration table doesn't exist, creating it")
		if err := m.createMigrationTable(ctx); err != nil {
			return fmt.Errorf("creating migration table: %w", err)
		}
		return nil
	}

	if err := m.fetchCurrentVersion(ctx); err != nil {
		return fmt.Errorf("fetching current version: %w", err)
	}

	m.logger.InfoContext(ctx, "Initialize migration table", "version", m.current)
	return nil
}

// determineMigrationType determines whether to migrate up or down
func (m *Migrator) determineMigrationType() MigrationType {
	switch m.target.CompareTo(m.current) {
	case 1:
		return MigrationUp
	case -1:
		return MigrationDown
	default:
		return MigrationUp
	}
}
