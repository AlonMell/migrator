package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlonMell/migrator/pkg/types"
	"github.com/AlonMell/migrator/pkg/version"
)

// Parser implements the Interface for parsing migration files
type Parser struct{}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{}
}

func (p *Parser) GetMigrationFile(fileName string) (*types.File, error) {
	migrationType, err := p.getMigrationType(fileName)
	if err != nil {
		return nil, err
	}

	version, err := p.getVersionFromFilename(fileName)
	if err != nil {
		return nil, err
	}

	comment, err := p.getCommentFromFilename(fileName)
	if err != nil {
		return nil, err
	}

	return &types.File{
		Comment: comment,
		Version: version,
		Type:    migrationType,
	}, nil
}

func (p *Parser) getMigrationType(fileName string) (types.MigrationType, error) {
	var zero types.MigrationType

	if p.isUpMigration(fileName) {
		return types.MigrationUp, nil
	} else if p.isDownMigration(fileName) {
		return types.MigrationDown, nil
	}

	return zero, fmt.Errorf("migration file is not a up/down migration")
}

func (p *Parser) isUpMigration(filename string) bool {
	return strings.HasSuffix(filename, ".up.sql")
}

func (p *Parser) isDownMigration(filename string) bool {
	return strings.HasSuffix(filename, ".down.sql")
}

// getCommentFromFilename extracts the comment part from a filename
// Expected format: nnnn.mm.nn[.comment].(up|down).sql
func (p *Parser) getCommentFromFilename(filename string) (string, error) {
	// Remove up/down.sql part
	base := strings.TrimSuffix(filename, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")

	parts := strings.Split(base, ".")

	if len(parts) > 3 {
		return strings.Join(parts[3:], "."), nil
	}

	return "", fmt.Errorf("invalid filename format: %s", filename)
}

// getVersionFromFilename extracts version information from a migration filename
// Expected formats:
// - nnnn.mm.nn[.comment].(up|down).sql
// where nnnn is the file number, mm is the major version, and nn is the minor version
func (p *Parser) getVersionFromFilename(fileName string) (*version.Version, error) {
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
