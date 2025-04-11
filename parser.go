package migrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
)

// FileFormat: nnnn.mm.nn[.comment].(up|down).sql
const pattern = `^(\d{4})\.(\d{2})\.(\d{2})(?:\.([^.]+))?\.(up|down)\.sql$`

var re *regexp.Regexp = regexp.MustCompile(pattern)

func ParseFileName(fileName string) (*FileInfo, error) {
	matches := re.FindStringSubmatch(fileName)

	if matches == nil {
		return nil, errors.New("incorrect File Format")
	}

	var info FileInfo

	info.Name = fileName

	fileNumber, _ := strconv.Atoi(matches[1])
	major, _ := strconv.Atoi(matches[2])
	minor, _ := strconv.Atoi(matches[3])

	info.Version = Version{
		Major:      major,
		Minor:      minor,
		FileNumber: fileNumber,
	}

	info.Comment = matches[4]

	info.MigrationType = Up
	if matches[5] == "down" {
		info.MigrationType = Down
	}

	return &info, nil
}

// FilterMigrationFiles returns sorted slice [FileInfo] and filtered by [MigrationType]
func FilterMigrationFiles(
	fileInfos []*FileInfo, migrationType MigrationType,
	current Version, target Version,
) []*FileInfo {
	if migrationType == Up {
		// current < target
		// range : [current target)
		return filterUp(fileInfos, current, target)
	}
	// current > target
	// range : (target current]
	return filterDown(fileInfos, current, target)
}

func filterUp(fileInfos []*FileInfo, current, target Version) []*FileInfo {
	var result []*FileInfo

	for _, info := range fileInfos {
		inRange := CompareVersion(info.Version, current) > 0 &&
			CompareVersion(info.Version, target) <= 0

		if info.MigrationType == Up && inRange {
			result = append(result, info)
		}
	}

	less := func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return CompareVersion(vI, vJ) < 0
	}

	sort.Slice(result, less)

	return result
}

func filterDown(fileInfos []*FileInfo, current Version, target Version) []*FileInfo {
	var result []*FileInfo

	for _, info := range fileInfos {
		inRange := CompareVersion(info.Version, current) <= 0 &&
			CompareVersion(info.Version, target) > 0

		if info.MigrationType == Down && inRange {
			result = append(result, info)
		}
	}

	less := func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return CompareVersion(vI, vJ) > 0
	}

	sort.Slice(result, less)

	return result
}

func ReadFile(path string) ([]byte, error) {
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

func GetFileNames(ctx context.Context, dirPath string) ([]string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(files))

	for _, file := range files {
		names = append(names, file.Name())
	}

	return names, nil
}
