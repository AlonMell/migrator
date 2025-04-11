package parser

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlonMell/migrator/internal/domain/types"
	"github.com/AlonMell/migrator/internal/domain/version"
)

type Reader interface {
	GetFileNames(context.Context) ([]string, error)
}

type Parser struct {
	r Reader
}

func New(r Reader) *Parser {
	return &Parser{
		r: r,
	}
}

func (p *Parser) GetMigrationFiles(ctx context.Context) ([]*types.FileInfo, error) {
	fileNames, err := p.r.GetFileNames(ctx)
	if err != nil {
		return nil, err
	}

	var files []*types.FileInfo

	for _, fileName := range fileNames {
		file, err := getMigrationFile(fileName)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

func getMigrationFile(fileName string) (*types.FileInfo, error) {
	version, err := getVersionFromFilename(fileName)
	if err != nil {
		return nil, err
	}

	comment, err := getCommentFromFilename(fileName)
	if err != nil {
		return nil, err
	}

	migrationType, err := getMigrationType(fileName)
	if err != nil {
		return nil, err
	}

	return &types.FileInfo{
		Name:    fileName,
		Version: version,
		Comment: comment,
		Type:    migrationType,
	}, nil
}

// getVersionFromFilename extracts version information from a migration filename
// Expected formats:
// - nnnn.mm.nn[.comment].(up|down).sql
// where nnnn is the file number, mm is the major version, and nn is the minor version
func getVersionFromFilename(fileName string) (*version.Version, error) {
	re := regexp.MustCompile(`^(\d{4})\.(\d{2})\.(\d{2})`)
	matches := re.FindStringSubmatch(fileName)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid filename format: %s", fileName)
	}

	fileNumber, _ := strconv.Atoi(matches[1])
	major, _ := strconv.Atoi(matches[2])
	minor, _ := strconv.Atoi(matches[3])

	return version.New(major, minor, fileNumber), nil
}

// getCommentFromFilename extracts the comment part from a filename
// Expected format: nnnn.mm.nn[.comment].(up|down).sql
func getCommentFromFilename(filename string) (string, error) {
	base := strings.TrimSuffix(filename, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")

	parts := strings.Split(base, ".")

	if len(parts) > 3 {
		return strings.Join(parts[3:], "."), nil
	}

	return "", fmt.Errorf("invalid filename format: %s", filename)
}

func getMigrationType(fileName string) (types.MigrationType, error) {
	var zero types.MigrationType

	if isUpMigration(fileName) {
		return types.MigrationUp, nil
	} else if isDownMigration(fileName) {
		return types.MigrationDown, nil
	}

	return zero, fmt.Errorf("migration file is not a up/down migration")
}

func isUpMigration(filename string) bool {
	return strings.HasSuffix(filename, ".up.sql")
}

func isDownMigration(filename string) bool {
	return strings.HasSuffix(filename, ".down.sql")
}
