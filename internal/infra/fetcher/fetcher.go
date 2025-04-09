package migrator

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/AlonMell/migrator"
	"github.com/AlonMell/migrator/internal/version"
)

type Parser interface {
	IsUpMigration(string) bool
	IsDownMigration(string) bool
	ParseVersionFromFilename(string) (*version.Version, error)
}

type Fetcher struct {
	migrationType migrator.MigrationType
	logger        migrator.Logger
	parser        Parser
	current       *version.Version
	target        *version.Version
}

func NewFetcher(migrationType migrator.MigrationType, parser Parser, current, target *version.Version) *Fetcher {
	return &Fetcher{
		migrationType: migrationType,
		parser:        parser,
		current:       current,
		target:        target,
	}
}

// GetFileNamesToExecute gets the list of files to execute based on migration type
func (f *Fetcher) GetFileNamesToExecute(path string) ([]string, error) {
	allFiles, err := f.findMigrationFiles(path)
	if err != nil {
		return nil, fmt.Errorf("finding migration files: %w", err)
	}

	var filteredFiles []string

	if f.migrationType == migrator.MigrationUp {
		filteredFiles = f.filterUpMigrationFiles(allFiles)
	} else {
		filteredFiles = f.filterDownMigrationFiles(allFiles)
	}

	return filteredFiles, nil
}

// filterUpMigrationFiles filters and sorts up migration files
func (f *Fetcher) filterUpMigrationFiles(files []string) []string {
	var result []string

	for _, file := range files {
		if !f.parser.IsUpMigration(file) {
			continue
		}

		version, err := f.parser.ParseVersionFromFilename(file)
		if err != nil {
			f.logger.WarnContext(context.Background(), "Skipping file with invalid name", "name", file)
			continue
		}

		// Include files with version higher than current and up to target
		if version.CompareTo(f.current) > 0 && version.CompareTo(f.target) <= 0 {
			result = append(result, file)
		}
	}

	// Sort files by version
	sort.Slice(result, func(i, j int) bool {
		vI, _ := f.parser.ParseVersionFromFilename(result[i])
		vJ, _ := f.parser.ParseVersionFromFilename(result[j])
		return vI.CompareTo(vJ) < 0
	})

	return result
}

// filterDownMigrationFiles filters and sorts down migration files
func (f *Fetcher) filterDownMigrationFiles(files []string) []string {
	var result []string

	for _, file := range files {
		if !f.parser.IsDownMigration(file) {
			continue
		}

		version, err := f.parser.ParseVersionFromFilename(file)
		if err != nil {
			f.logger.WarnContext(context.Background(), "Skipping file with invalid name", "name", file)
			continue
		}

		// Include files with version less than or equal to current and higher than target
		if version.CompareTo(f.current) <= 0 && version.CompareTo(f.target) > 0 {
			result = append(result, file)
		}
	}

	// Sort files by version in descending order for down migration
	sort.Slice(result, func(i, j int) bool {
		vI, _ := f.parser.ParseVersionFromFilename(result[i])
		vJ, _ := f.parser.ParseVersionFromFilename(result[j])
		return vI.CompareTo(vJ) > 0
	})

	return result
}

// findMigrationFiles finds all migration files in the path
func (f *Fetcher) findMigrationFiles(path string) ([]string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var result []string
	for _, file := range files {
		has_suffix := f.parser.IsUpMigration(file.Name()) || f.parser.IsDownMigration(file.Name())
		if !file.IsDir() && has_suffix {
			result = append(result, file.Name())
		}
	}

	return result, nil
}
