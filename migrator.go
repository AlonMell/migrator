package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/AlonMell/migrator/internal/executor"
	"github.com/AlonMell/migrator/internal/fetcher"
	"github.com/AlonMell/migrator/internal/parser"
	"github.com/AlonMell/migrator/internal/reader"
	"github.com/AlonMell/migrator/internal/schema"
	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

// Migrator handles database migrations
type Migrator struct {
	db             *sql.DB
	logger         types.Logger
	path           string
	table          string
	currentVersion *version.Version
	targetVersion  *version.Version
	migrationType  types.MigrationType
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

func New(cfg Config) *Migrator {
	return &Migrator{
		db:            cfg.DB,
		logger:        cfg.Logger,
		path:          cfg.Path,
		table:         cfg.Table,
		targetVersion: version.New(cfg.MajorVer, cfg.MinorVer, 0),
	}
}

// Migrate performs the migration process
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.InfoContext(ctx, "Starting migration process")

	schema := schema.New(m.db, m.table, m.logger)
	currentVersion, err := schema.InitializeTable(ctx)
	if err != nil {
		return fmt.Errorf("initializing schema: %w", err)
	}
	m.currentVersion = currentVersion

	m.migrationType = m.getMigrationType()

	parser := parser.New()
	fetcher := fetcher.New(m.logger, parser, m.migrationType, m.currentVersion, m.targetVersion)
	files, err := fetcher.GetFilesToExecute(ctx, m.path)
	if err != nil {
		return fmt.Errorf("getting files to execute: %w", err)
	}

	if len(files) == 0 {
		m.logger.InfoContext(ctx, "No migration files to execute - database is up to date")
		return nil
	}

	reader := reader.New(files, m.path)
	executor := executor.New(m.db, m.table, m.logger, reader)
	executor.Execute(ctx, files)

	m.logger.InfoContext(ctx, "Migration completed successfully")
	return nil
}

func (m *Migrator) getMigrationType() types.MigrationType {
	if m.currentVersion.CompareTo(m.targetVersion) > 0 {
		return types.MigrationDown
	}
	return types.MigrationUp
}
