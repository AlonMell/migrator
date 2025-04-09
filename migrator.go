package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

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

// readFile reads the content of a file
func (m *Migrator) readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return content, nil
}

// recordMigration records a migration in the history table
func (m *Migrator) recordMigration(ctx context.Context, tx *sql.Tx, version *Version, comment string, isUp bool) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (major_version, minor_version, file_number, comment, migration_type)
		VALUES ($1, $2, $3, $4, $5)
	`, m.table)

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

// fetchCurrentVersion fetches the current version from the database
func (m *Migrator) fetchCurrentVersion(ctx context.Context) error {
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
	`, m.table)

	var major, minor, fileNum string

	err := m.db.QueryRowContext(ctx, query).Scan(&major, &minor, &fileNum)
	if err != nil {
		if err == sql.ErrNoRows {
			// No migrations applied yet
			m.current = NewVersion(0, 0, 0)
			return nil
		}
		return fmt.Errorf("querying current version: %w", err)
	}

	majorInt, _ := strconv.Atoi(major)
	minorInt, _ := strconv.Atoi(minor)
	fileNumInt, _ := strconv.Atoi(fileNum)

	m.current = NewVersion(majorInt, minorInt, fileNumInt)
	return nil
}
