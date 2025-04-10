package executor

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlonMell/grovelog/util"
	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

type Reader interface {
	ReadFile(ctx context.Context) ([]byte, error)
}

// Executor implements the Interface for executing migration files
type Executor struct {
	db     *sql.DB
	table  string
	logger types.Logger
	reader Reader
}

// New creates a new Executor instance
func New(db *sql.DB, table string, logger types.Logger, reader Reader) *Executor {
	return &Executor{
		db:     db,
		table:  table,
		logger: logger,
		reader: reader,
	}
}

func (e *Executor) InitializeTable(ctx context.Context) error {
	// Check if migration table exists
	exists, err := e.tableExists(ctx)
	if err != nil {
		return fmt.Errorf("checking migration table: %w", err)
	}

	// Create table if it doesn't exist
	if !exists {
		e.logger.InfoContext(ctx, "Migration table doesn't exist, creating it")
		if err := e.createMigrationTable(ctx); err != nil {
			return fmt.Errorf("creating migration table: %w", err)
		}
	} else {
		current, err := e.fetchCurrentVersion(ctx)
		if err != nil {
			return fmt.Errorf("fetching current version: %w", err)
		}

		e.logger.InfoContext(ctx, "Current database version", "version", current)
	}

	return nil
}

// ExecuteFile executes a migration file
func (e *Executor) Execute(ctx context.Context, filename string) error {
	e.logger.InfoContext(ctx, "Executing file", "filename", filename)

	version, err := e.parser.GetVersionFromFilename(filename)
	if err != nil {
		return fmt.Errorf("parsing version from filename: %w", err)
	}

	filePath := filepath.Join(e.path, filename)
	content, err := e.parser.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

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

	if _, err = tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	// Record migration in history table if not already recorded in the SQL
	if !strings.Contains(string(content), fmt.Sprintf("INSERT INTO %s", e.table)) {
		comment := e.parser.GetCommentFromFilename(filename)
		if err = e.recordMigration(ctx, tx, version, comment, e.parser.IsUpMigration(filename)); err != nil {
			return fmt.Errorf("recording migration: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	e.logger.InfoContext(ctx, "Successfully executed file", "filename", filename)
	return nil
}

// tableExists checks if the migration table exists
func (e *Executor) tableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`

	var exists bool
	err := e.db.QueryRowContext(ctx, query, e.table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("querying table existence: %w", err)
	}

	return exists, nil
}

// createMigrationTable creates the migration table
func (e *Executor) createMigrationTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			date_applied TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			major_version VARCHAR(2),
			minor_version VARCHAR(2),
			file_number VARCHAR(4),
			comment TEXT,
			migration_type VARCHAR(4)
		)
	`, e.table)

	_, err := e.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}

	return nil
}

// fetchCurrentVersion fetches the current version from the database
func (e *Executor) fetchCurrentVersion(ctx context.Context) (*version.Version, error) {
	query := fmt.Sprintf(`
		WITH latest_migrations AS (
			SELECT
				major_version,
				minor_version,
				file_number,
				date_applied,
				ROW_NUMBER() OVER (
					PARTITION BY major_version, minor_version, file_number
					ORDER BY date_applied DESC
				) as rn
			FROM %s
		)
		SELECT major_version, minor_version, file_number
		FROM latest_migrations
		WHERE rn = 1
		ORDER BY date_applied DESC
		LIMIT 1
	`, e.table)

	var major, minor, fileNum string

	err := e.db.QueryRowContext(ctx, query).Scan(&major, &minor, &fileNum)
	if err != nil {
		if err == sql.ErrNoRows {
			// No migrations applied yet
			return version.New(0, 0, 0), nil
		}
		return nil, fmt.Errorf("querying current version: %w", err)
	}

	majorInt, _ := strconv.Atoi(major)
	minorInt, _ := strconv.Atoi(minor)
	fileNumInt, _ := strconv.Atoi(fileNum)

	return version.New(majorInt, minorInt, fileNumInt), nil
}

// recordMigration records a migration in the history table
func (e *Executor) recordMigration(ctx context.Context, tx *sql.Tx, version *version.Version, comment string, isUp bool) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (major_version, minor_version, file_number, comment, migration_type)
		VALUES ($1, $2, $3, $4, $5)
	`, e.table)

	migrationType := "up"
	if !isUp {
		migrationType = "down"
	}

	_, err := tx.ExecContext(
		ctx,
		query,
		fmt.Sprintf("%02d", version.Major),
		fmt.Sprintf("%02d", version.Minor),
		fmt.Sprintf("%04d", version.FileNumber),
		comment,
		migrationType,
	)

	if err != nil {
		return fmt.Errorf("inserting migration record: %w", err)
	}

	return nil
}
