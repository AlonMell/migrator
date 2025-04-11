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

func (m MigrationType) String() string {
	switch m {
	case MigrationUp:
		return "up"
	case MigrationDown:
		return "down"
	default:
		return "unknown"
	}
}

// Logger defines the interface for logging in the application
type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type FileInfo struct {
	Name    string
	Version *version.Version
	Comment string
	Type    MigrationType
}

func NewFile(
	name string, comment string,
	migrationType MigrationType, version *version.Version,
) *FileInfo {
	return &FileInfo{
		Name:    name,
		Version: version,
		Comment: comment,
		Type:    migrationType,
	}
}
