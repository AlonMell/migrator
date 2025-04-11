package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/AlonMell/migrator/internal/domain/types"
	"github.com/AlonMell/migrator/internal/domain/version"
	"github.com/AlonMell/migrator/internal/executor"
	"github.com/AlonMell/migrator/internal/fetcher"
	"github.com/AlonMell/migrator/internal/parser"
	"github.com/AlonMell/migrator/internal/reader"
	"github.com/AlonMell/migrator/internal/schema"
)

// Config holds configuration for Migrator
type Config struct {
	DB       *sql.DB
	Logger   *slog.Logger
	Path     string
	Table    string
	MajorVer int
	MinorVer int
}

// Migrate performs the migration process
func Migrate(ctx context.Context, cfg Config) error {
	cfg.Logger.InfoContext(ctx, "Starting migration process")

	schema := schema.New(cfg.Logger, cfg.DB, cfg.Table)

	currentVersion, err := schema.InitializeTable(ctx)
	if err != nil {
		return fmt.Errorf("initializing schema: %w", err)
	}
	targetVersion := version.New(cfg.MajorVer, cfg.MinorVer, 0)

	migrationType := getMigrationType(targetVersion, currentVersion)

	reader := reader.New(cfg.Path)
	parser := parser.New(reader)

	fetcher := fetcher.New(cfg.Logger, parser, migrationType, currentVersion, targetVersion)
	files, err := fetcher.FilterMigrationFiles(ctx)
	if err != nil {
		return fmt.Errorf("getting files to execute: %w", err)
	}
	if len(files) == 0 {
		cfg.Logger.InfoContext(ctx, "No migration files to execute - database is up to date")
		return nil
	}

	executor := executor.New(cfg.Logger, cfg.DB, cfg.Table, reader)
	if err := executor.Execute(ctx, files); err != nil {
		return err
	}

	cfg.Logger.InfoContext(ctx, "Migration completed successfully")
	return nil
}

func getMigrationType(
	target, current *version.Version,
) types.MigrationType {
	if current.CompareTo(target) > 0 {
		return types.MigrationDown
	}
	return types.MigrationUp
}
