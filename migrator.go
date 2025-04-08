package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/AlonMell/grovelog"
	"github.com/AlonMell/grovelog/util"
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

// getFilesToExecute gets the list of files to execute based on migration type
func (m *Migrator) getFilesToExecute(migrationType MigrationType) ([]string, error) {
	allFiles, err := m.findMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("finding migration files: %w", err)
	}

	var filteredFiles []string

	if migrationType == MigrationUp {
		filteredFiles = m.filterUpMigrationFiles(allFiles)
	} else {
		filteredFiles = m.filterDownMigrationFiles(allFiles)
	}

	return filteredFiles, nil
}

// filterUpMigrationFiles filters and sorts up migration files
func (m *Migrator) filterUpMigrationFiles(files []string) []string {
	var result []string

	for _, file := range files {
		if !IsUpMigration(file) {
			continue
		}

		version, err := ParseVersionFromFilename(file)
		if err != nil {
			m.logger.Warn("Skipping file with invalid name", "name", file)
			continue
		}

		// Include files with version higher than current and up to target
		if version.CompareTo(m.current) > 0 && version.CompareTo(m.target) <= 0 {
			result = append(result, file)
		}
	}

	// Sort files by version
	sort.Slice(result, func(i, j int) bool {
		vI, _ := ParseVersionFromFilename(result[i])
		vJ, _ := ParseVersionFromFilename(result[j])
		return vI.CompareTo(vJ) < 0
	})

	return result
}

// filterDownMigrationFiles filters and sorts down migration files
func (m *Migrator) filterDownMigrationFiles(files []string) []string {
	var result []string

	for _, file := range files {
		if !IsDownMigration(file) {
			continue
		}

		version, err := ParseVersionFromFilename(file)
		if err != nil {
			m.logger.Warn("Skipping file with invalid name", "name", file)
			continue
		}

		// Include files with version less than or equal to current and higher than target
		if version.CompareTo(m.current) <= 0 && version.CompareTo(m.target) > 0 {
			result = append(result, file)
		}
	}

	// Sort files by version in descending order for down migration
	sort.Slice(result, func(i, j int) bool {
		vI, _ := ParseVersionFromFilename(result[i])
		vJ, _ := ParseVersionFromFilename(result[j])
		return vI.CompareTo(vJ) > 0
	})

	return result
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

// findMigrationFiles finds all migration files in the path
func (m *Migrator) findMigrationFiles() ([]string, error) {
	files, err := os.ReadDir(m.path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var result []string
	for _, file := range files {
		if !file.IsDir() && (IsUpMigration(file.Name()) || IsDownMigration(file.Name())) {
			result = append(result, file.Name())
		}
	}

	return result, nil
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
