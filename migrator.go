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

func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.InitMigrationTable(ctx); err != nil {
		return err
	}

	current, err := m.FetchCurrentVersion(ctx)
	if err != nil {
		return err
	}

	fileNames, err := parser.GetFileNames(ctx, m.path)
	if err != nil {
		return err
	}

	files := make([]*parser.FileInfo, 0, len(fileNames))
	for _, name := range fileNames {
		info, err := parser.ParseFileName(name)
		if err != nil {
			return err
		}
		files = append(files, info)
	}

	files = parser.FilterMigrationFiles(files, current, m.target)

	if err := m.Execute(ctx, files); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) InitMigrationTable(ctx context.Context) error {
	if exists, err := m.tableExists(ctx); err != nil {
		return fmt.Errorf("checking migration table: %w", err)
	} else if exists {
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
		if err := m.execute(ctx, tx, content, file); err != nil {
			return fmt.Errorf("executing file: %w", err)
		}
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
	query := fmt.Sprintf(`
		INSERT INTO %s (major_version, minor_version, file_number, comment, migration_type)
		VALUES ($1, $2, $3, $4, $5)
	`, m.table)

	_, err := tx.ExecContext(
		ctx,
		query,
		fmt.Sprintf("%02d", file.Version.Major),
		fmt.Sprintf("%02d", file.Version.Minor),
		fmt.Sprintf("%04d", file.Version.FileNumber),
		file.Comment,
		file.MigrationType.String(),
	)

	if err != nil {
		return fmt.Errorf("inserting migration record: %w", err)
	}

	return nil
}
