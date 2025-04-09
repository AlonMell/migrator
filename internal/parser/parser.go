package migrator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Parser struct {
	filename string
}

func NewParser(filename string) *Parser {
	return &Parser{filename: filename}
}

// ParseVersionFromFilename extracts version information from a migration filename
// Expected formats:
// - nnnn.mm.nn[.comment].(up|down).sql
// where nnnn is the file number, mm is the major version, and nn is the minor version
func (p *Parser) ParseVersionFromFilename() (*Version, error) {
	re := regexp.MustCompile(`^(\d{4})\.(\d{2})\.(\d{2})`)
	matches := re.FindStringSubmatch(p.filename)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid filename format: %s", p.filename)
	}

	fileNumber, _ := strconv.Atoi(matches[1])
	major, _ := strconv.Atoi(matches[2])
	minor, _ := strconv.Atoi(matches[3])

	return NewVersion(major, minor, fileNumber), nil
}

// GetCommentFromFilename extracts the comment part from a filename
// Expected format: nnnn.mm.nn[.comment].(up|down).sql
func (p *Parser) GetCommentFromFilename() string {
	// Remove up/down.sql part
	base := strings.TrimSuffix(p.filename, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")

	parts := strings.Split(base, ".")

	// Skip the first three parts (file number, major, minor)
	if len(parts) > 3 {
		return strings.Join(parts[3:], ".")
	}

	return ""
}

// IsUpMigration checks if the filename is an up migration
func (p *Parser) IsUpMigration() bool {
	return strings.HasSuffix(p.filename, ".up.sql")
}

// IsDownMigration checks if the filename is a down migration
func (p *Parser) IsDownMigration() bool {
	return strings.HasSuffix(p.filename, ".down.sql")
}

// GetBaseName gets the base name of a migration file without the up/down suffix
func (p *Parser) GetBaseName() string {
	if p.IsUpMigration() {
		return strings.TrimSuffix(p.filename, ".up.sql")
	} else if p.IsDownMigration() {
		return strings.TrimSuffix(p.filename, ".down.sql")
	}
	return p.filename
}
