package fetcher

import (
	"context"
	"sort"

	"github.com/AlonMell/migrator/internal/domain/types"
	"github.com/AlonMell/migrator/internal/domain/version"
)

type Parser interface {
	GetMigrationFiles(ctx context.Context) ([]*types.FileInfo, error)
}

type Fetcher struct {
	logger        types.Logger
	parser        Parser
	migrationType types.MigrationType
	current       *version.Version
	target        *version.Version
}

// New creates a new Fetcher instance
func New(
	logger types.Logger, parser Parser,
	migrationType types.MigrationType,
	current, target *version.Version,
) *Fetcher {
	return &Fetcher{
		logger:        logger,
		parser:        parser,
		migrationType: migrationType,
		current:       current,
		target:        target,
	}
}

// GetFilesToExecute gets the list of files to execute based on migration type
func (f *Fetcher) FilterMigrationFiles(ctx context.Context) ([]*types.FileInfo, error) {
	files, err := f.parser.GetMigrationFiles(ctx)
	if err != nil {
		return nil, err
	}

	var filteredFiles []*types.FileInfo

	if f.migrationType == types.MigrationUp {
		filteredFiles = f.filterUpMigrationFiles(ctx, files)
	} else {
		filteredFiles = f.filterDownMigrationFiles(ctx, files)
	}

	return filteredFiles, nil
}

// filterUpMigrationFiles filters and sorts up migration files
func (f *Fetcher) filterUpMigrationFiles(
	ctx context.Context, files []*types.FileInfo,
) []*types.FileInfo {
	var result []*types.FileInfo

	for _, file := range files {
		if file.Type != types.MigrationUp {
			continue
		}

		if file.Version.CompareTo(f.current) > 0 &&
			file.Version.CompareTo(f.target) <= 0 {
			result = append(result, file)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return vI.CompareTo(vJ) < 0
	})

	return result
}

// filterDownMigrationFiles filters and sorts down migration files
func (f *Fetcher) filterDownMigrationFiles(
	ctx context.Context, files []*types.FileInfo,
) []*types.FileInfo {
	var result []*types.FileInfo

	for _, file := range files {
		if file.Type != types.MigrationDown {
			continue
		}

		if file.Version.CompareTo(f.current) <= 0 &&
			file.Version.CompareTo(f.target) > 0 {
			result = append(result, file)
		}
	}

	// Sort files by version in descending order for down migration
	sort.Slice(result, func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return vI.CompareTo(vJ) > 0
	})

	return result
}
