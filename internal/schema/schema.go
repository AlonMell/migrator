package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

// Schema manages the migration table schema
type Schema struct {
	db     *sql.DB
	table  string
	logger types.Logger
}

// New creates a new Schema instance
func New(db *sql.DB, table string, logger types.Logger) *Schema {
	return &Schema{
		db:     db,
		table:  table,
		logger: logger,
	}
}

func (s *Schema) InitializeTable(ctx context.Context) (*version.Version, error) {
	exists, err := s.tableExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking migration table: %w", err)
	}

	if !exists {
		s.logger.InfoContext(ctx, "Migration table doesn't exist, creating it")
		if err := s.createMigrationTable(ctx); err != nil {
			return nil, fmt.Errorf("creating migration table: %w", err)
		}
	}

	current, err := s.fetchCurrentVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching current version: %w", err)
	}

	s.logger.InfoContext(ctx, "Current database version", "version", current)

	return current, nil
}

// tableExists checks if the migration table exists
func (s *Schema) tableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	var exists bool

	err := s.db.QueryRowContext(ctx, query, s.table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("querying table existence: %w", err)
	}

	return exists, nil
}

// createMigrationTable creates the migration table
func (s *Schema) createMigrationTable(ctx context.Context) error {
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
	`, s.table)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}

	return nil
}

// fetchCurrentVersion fetches the current version from the database
func (s *Schema) fetchCurrentVersion(ctx context.Context) (*version.Version, error) {
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
	`, s.table)

	var major, minor, fileNum string

	err := s.db.QueryRowContext(ctx, query).Scan(&major, &minor, &fileNum)
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
