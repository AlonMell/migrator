package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	ver "github.com/AlonMell/migrator/internal/version"
)

type MigrationType int

const (
	Up MigrationType = iota
	Down
)

func (mt MigrationType) String() string {
	if mt == Up {
		return "up"
	}
	return "down"
}

type FileInfo struct {
	Name          string
	Version       ver.Version
	Comment       string
	MigrationType MigrationType
}

// FileFormat: nnnn.mm.nn[.comment].(up|down).sql
const pattern = `^(\d{4})\.(\d{2})\.(\d{2})(?:\.([^.]+))?\.(up|down)\.sql$`

var re *regexp.Regexp = regexp.MustCompile(pattern)

func ParseFileName(fileName string) (*FileInfo, error) {
	matches := re.FindStringSubmatch(fileName)

	if matches == nil {
		return nil, errors.New("incorrect File Format")
	}

	version, err := ver.ParseStringToVersion(matches[1], matches[2], matches[3])
	if err != nil {
		return nil, err
	}

	info := &FileInfo{
		Name:          fileName,
		Version:       version,
		Comment:       matches[4],
		MigrationType: Up,
	}

	if matches[5] == "down" {
		info.MigrationType = Down
	}

	return info, nil
}

// FilterMigrationFiles returns sorted slice [FileInfo] and filtered by [MigrationType]
func FilterMigrationFiles(
	fileInfos []*FileInfo, current, target ver.Version,
) []*FileInfo {
	migrationType := getMigrationType(current, target)
	if migrationType == Up {
		// current < target
		// range : [current target)
		return filterUp(fileInfos, current, target)
	}
	// current > target
	// range : (target current]
	return filterDown(fileInfos, current, target)
}

func getMigrationType(current, target ver.Version) MigrationType {
	if ver.CompareVersion(current, target) > 0 {
		return Up
	}
	return Down
}

func filterUp(fileInfos []*FileInfo, current, target ver.Version) []*FileInfo {
	var result []*FileInfo

	for _, info := range fileInfos {
		inRange := ver.CompareVersion(info.Version, current) > 0 &&
			ver.CompareVersion(info.Version, target) <= 0

		if info.MigrationType == Up && inRange {
			result = append(result, info)
		}
	}

	less := func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return ver.CompareVersion(vI, vJ) < 0
	}

	sort.Slice(result, less)

	return result
}

func filterDown(fileInfos []*FileInfo, current, target ver.Version) []*FileInfo {
	var result []*FileInfo

	for _, info := range fileInfos {
		inRange := ver.CompareVersion(info.Version, current) <= 0 &&
			ver.CompareVersion(info.Version, target) > 0

		if info.MigrationType == Down && inRange {
			result = append(result, info)
		}
	}

	less := func(i, j int) bool {
		vI, vJ := result[i].Version, result[j].Version
		return ver.CompareVersion(vI, vJ) > 0
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
