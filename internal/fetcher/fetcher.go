package fetcher

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

type Parser interface {
	GetMigrationFile(fileName string) (*types.File, error)
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
	logger types.Logger, parser Parser, migrationType types.MigrationType,
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
func (f *Fetcher) GetFilesToExecute(
	ctx context.Context, path string,
) ([]*types.File, error) {
	allFiles, err := f.findMigrationFiles(path)
	if err != nil {
		return nil, fmt.Errorf("finding migration files: %w", err)
	}

	var filteredFiles []*types.File

	if f.migrationType == types.MigrationUp {
		filteredFiles = f.filterUpMigrationFiles(ctx, allFiles)
	} else {
		filteredFiles = f.filterDownMigrationFiles(ctx, allFiles)
	}

	return filteredFiles, nil
}

// filterUpMigrationFiles filters and sorts up migration files
func (f *Fetcher) filterUpMigrationFiles(
	ctx context.Context, files []*types.File,
) []*types.File {
	var result []*types.File

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
	ctx context.Context, files []*types.File,
) []*types.File {
	var result []*types.File

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

// findMigrationFiles finds all migration files in the path
func (f *Fetcher) findMigrationFiles(path string) ([]*types.File, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var result []*types.File

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		migrationFile, err := f.parser.GetMigrationFile(file.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, migrationFile)
	}

	return result, nil
}
