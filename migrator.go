package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AlonMell/grovelog"
	"github.com/AlonMell/grovelog/util"
	"github.com/AlonMell/migrator/internal/parser"
	ver "github.com/AlonMell/migrator/internal/version"
)

type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type Migrator struct {
	db     *sql.DB
	logger Logger
	target ver.Version
	table  string
	path   string
}

func NewDefaultLogger() *slog.Logger {
	opts := grovelog.NewOptions(slog.LevelDebug, "", grovelog.Color)
	return grovelog.NewLogger(os.Stdout, opts)
}

func New(
	db *sql.DB, logger Logger,
	major, minor int,
	table, path string,
) *Migrator {
	if logger == nil {
		logger = NewDefaultLogger()
	}
	return &Migrator{
		db:     db,
		logger: logger,
		target: ver.Version{Major: major, Minor: minor},
		table:  table,
		path:   path,
	}
}

// Migrate performs database migration from the current version to the target version
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.InfoContext(ctx, "Starting database migration")

	if err := m.InitMigrationTable(ctx); err != nil {
		m.logger.ErrorContext(ctx, "Initialization migration table completed with error", util.Err(err))
		return err
	}
	m.logger.DebugContext(ctx, "Initialization migration table completed successfully")

	m.logger.DebugContext(ctx, "Fetching current database version")
	current, err := m.FetchCurrentVersion(ctx)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to fetch current version", util.Err(err))
		return err
	}
	m.logger.InfoContext(ctx, "Current database version", "version", current)

	m.logger.DebugContext(ctx, "Getting migration files from path", "path", m.path)
	fileNames, err := parser.GetFileNames(ctx, m.path)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to get migration files", util.Err(err))
		return err
	}
	m.logger.DebugContext(ctx, "Found migration files", "count", len(fileNames))

	m.logger.DebugContext(ctx, "Parsing migration filenames")
	files := make([]*parser.FileInfo, 0, len(fileNames))
	for _, name := range fileNames {
		m.logger.DebugContext(ctx, "Parsing filename", "file", name)
		info, err := parser.ParseFileName(name)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to parse filename", "file", name, util.Err(err))
			return err
		}
		files = append(files, info)
	}
	m.logger.DebugContext(ctx, "Successfully parsed all filenames")

	m.logger.DebugContext(ctx, "Filtering migration files", "current", current, "target", m.target)
	files = parser.FilterMigrationFiles(files, current, m.target)
	m.logger.DebugContext(ctx, "Migration files filtered", "files_to_apply", len(files))

	if len(files) == 0 {
		m.logger.InfoContext(ctx, "No migrations to apply, database is up to date")
		return nil
	}

	m.logger.InfoContext(ctx, "Executing migration files")
	if err := m.Execute(ctx, files); err != nil {
		m.logger.ErrorContext(ctx, "Migration execution failed", util.Err(err))
		return err
	}
	m.logger.InfoContext(ctx, "Migration completed successfully", "from", current, "to", m.target)

	return nil
}

func (m *Migrator) InitMigrationTable(ctx context.Context) error {
	if exists, err := m.tableExists(ctx); err != nil {
		return fmt.Errorf("checking migration table: %w", err)
	} else if exists {
		m.logger.InfoContext(ctx, "Migration table already exists")
		return nil
	}

	m.logger.InfoContext(ctx, "Migration table doesn't exist, creating it")
	if err := m.createMigrationTable(ctx); err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}

	return nil
}

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

func (m *Migrator) FetchCurrentVersion(ctx context.Context) (ver.Version, error) {
	query := fmt.Sprintf(`
			SELECT major_version, minor_version, file_number
			FROM %s
			ORDER BY date_applied DESC
			LIMIT 1
		`, m.table)

	var major, minor, fileNumber string

	err := m.db.QueryRowContext(ctx, query).Scan(&major, &minor, &fileNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ver.Zero, nil
		}
		return ver.Zero, fmt.Errorf("querying current version: %w", err)
	}

	return ver.ParseStringToVersion(major, minor, fileNumber)
}

func (m *Migrator) Execute(ctx context.Context, files []*parser.FileInfo) error {
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

	for _, file := range files {
		filepath := filepath.Join(m.path, file.Name)
		content, err := parser.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		m.logger.DebugContext(ctx, "Execute migration SQL script", "file", file.Name)
		if err := m.execute(ctx, tx, content, file); err != nil {
			return fmt.Errorf("executing file: %w", err)
		}
		m.logger.DebugContext(ctx, "Executing complete successfully", "file", file.Name)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	m.logger.InfoContext(ctx, "Successfully executed files")

	return nil
}

func (m *Migrator) execute(
	ctx context.Context, tx *sql.Tx, content []byte, file *parser.FileInfo,
) error {
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	if err := m.recordMigration(ctx, tx, file); err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	return nil
}

// recordMigration records a migration in the history table
func (m *Migrator) recordMigration(ctx context.Context, tx *sql.Tx, file *parser.FileInfo) error {
	majorVersion := fmt.Sprintf("%02d", file.Version.Major)
	minorVersion := fmt.Sprintf("%02d", file.Version.Minor)
	fileNumber := fmt.Sprintf("%04d", file.Version.FileNumber)
	migrationType := file.MigrationType.String()

	m.logger.DebugContext(ctx, "Recording migration",
		"major_version", majorVersion,
		"minor_version", minorVersion,
		"file_number", fileNumber,
		"comment", file.Comment,
		"migration_type", migrationType,
		"table", m.table)

	query := fmt.Sprintf(`
		INSERT INTO %s (major_version, minor_version, file_number, comment, migration_type)
		VALUES ($1, $2, $3, $4, $5)
	`, m.table)

	res, err := tx.ExecContext(
		ctx,
		query,
		majorVersion,
		minorVersion,
		fileNumber,
		file.Comment,
		migrationType,
	)
	if err != nil {
		return fmt.Errorf("inserting migration record: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		m.logger.WarnContext(ctx, "Failed to get rows affected", util.Err(err))
	} else {
		m.logger.DebugContext(ctx, "Migration record inserted", "rows_affected", rowsAffected)
	}

	return nil
}
