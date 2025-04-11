package executor

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AlonMell/grovelog/util"
	"github.com/AlonMell/migrator/pkg/types"
)

type Reader interface {
	ReadFile(context.Context, *types.FileInfo) ([]byte, error)
}

// Executor implements the Interface for executing migration files
type Executor struct {
	logger types.Logger
	db     *sql.DB
	reader Reader
	table  string
}

// New creates a new Executor instance
func New(logger types.Logger, db *sql.DB, table string, reader Reader) *Executor {
	return &Executor{
		logger: logger,
		db:     db,
		table:  table,
		reader: reader,
	}
}

// ExecuteFile executes a migration file
func (e *Executor) Execute(ctx context.Context, files []*types.FileInfo) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "Rolling back transaction", util.Err(err))
			_ = tx.Rollback()
		}
	}()

	for _, file := range files {
		content, err := e.reader.ReadFile(ctx, file)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		if err := e.execute(ctx, tx, content, file); err != nil {
			return fmt.Errorf("executing file: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	e.logger.InfoContext(ctx, "Successfully executed files")

	return nil
}

func (e *Executor) execute(
	ctx context.Context, tx *sql.Tx, content []byte, file *types.FileInfo,
) error {
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	if err := e.recordMigration(ctx, tx, file); err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	return nil
}

// recordMigration records a migration in the history table
func (e *Executor) recordMigration(ctx context.Context, tx *sql.Tx, file *types.FileInfo) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (major_version, minor_version, file_number, comment, migration_type)
		VALUES ($1, $2, $3, $4, $5)
	`, e.table)

	migrationType := "up"
	if file.Type == types.MigrationDown {
		migrationType = "down"
	}

	_, err := tx.ExecContext(
		ctx,
		query,
		fmt.Sprintf("%02d", file.Version.Major),
		fmt.Sprintf("%02d", file.Version.Minor),
		fmt.Sprintf("%04d", file.Version.FileNumber),
		file.Comment,
		migrationType,
	)

	if err != nil {
		return fmt.Errorf("inserting migration record: %w", err)
	}

	return nil
}
