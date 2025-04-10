package types

import (
	"context"

	"github.com/AlonMell/migrator/pkg/version"
)

// MigrationType indicates the type of migration operation
type MigrationType int

const (
	// MigrationUp represents an up migration
	MigrationUp MigrationType = iota
	// MigrationDown represents a down migration
	MigrationDown
)

// Logger defines the interface for logging in the application
type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type File struct {
	Name    string
	Comment string
	Type    MigrationType
	Version *version.Version
}

func NewFile(comment string, migrationType MigrationType, version *version.Version) *File {
	return &File{
		Comment: comment,
		Type:    migrationType,
		Version: version,
	}
}
