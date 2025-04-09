package migrator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AlonMell/grovelog/util"
)

type Executor struct {
}

// executeFile executes a migration file
func (m *Migrator) executeFile(ctx context.Context, filename string) error {
	m.logger.InfoContext(ctx, "Executing file", "filename", filename)

	version, err := ParseVersionFromFilename(filename)
	if err != nil {
		return fmt.Errorf("parsing version from filename: %w", err)
	}

	filePath := filepath.Join(m.path, filename)
	content, err := m.readFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			m.logger.ErrorContext(ctx, "Rolling back transaction", util.Err(err))
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	// Record migration in history table if not already recorded in the SQL
	if !strings.Contains(string(content), fmt.Sprintf("INSERT INTO %s", m.table)) {
		comment := GetCommentFromFilename(filename)
		if err = m.recordMigration(ctx, tx, version, comment, IsUpMigration(filename)); err != nil {
			return fmt.Errorf("recording migration: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	m.logger.InfoContext(ctx, "Successfully executed file", "filename", filename)
	return nil
}

// tableExists checks if the migration table exists
func (m *Migrator) tableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`

	var exists bool
	err := m.db.QueryRowContext(ctx, query, m.table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("querying table existence: %w", err)
	}

	return exists, nil
}

// createMigrationTable creates the migration table
func (m *Migrator) createMigrationTable(ctx context.Context) error {
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
	`, m.table)

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}

	return nil
}
