package migrator

import (
	"fmt"
	"os"
	"sort"

	"github.com/AlonMell/migrator"
)

type Fetcher struct {
	migrationType migrator.MigrationType
}

// getFilesToExecute gets the list of files to execute based on migration type
func (f *Fetcher) getFilesToExecute() ([]string, error) {
	allFiles, err := f.findMigrationFiles()
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
func (m *Fetcher) filterDownMigrationFiles(files []string) []string {
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

// findMigrationFiles finds all migration files in the path
func (m *Fetcher) findMigrationFiles() ([]string, error) {
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
