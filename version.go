package migrator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a migration version with major, minor version and file number
type Version struct {
	Major      int
	Minor      int
	FileNumber int
}

// NewVersion creates a new Version instance
func NewVersion(major, minor, fileNumber int) *Version {
	return &Version{
		Major:      major,
		Minor:      minor,
		FileNumber: fileNumber,
	}
}

// String returns a formatted string representation of the version
func (v *Version) String() string {
	return fmt.Sprintf("%02d.%02d.%04d", v.Major, v.Minor, v.FileNumber)
}

// CompareTo compares this version with another version
// Returns:
//
//	-1 if this version is less than the other version
//	 0 if this version is equal to the other version
//	 1 if this version is greater than the other version
func (v *Version) CompareTo(other *Version) int {
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}

	// Major versions are equal, compare minor versions
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}

	// Minor versions are equal, compare file numbers
	if v.FileNumber < other.FileNumber {
		return -1
	}
	if v.FileNumber > other.FileNumber {
		return 1
	}

	// Versions are equal
	return 0
}

// ParseVersionFromFilename extracts version information from a migration filename
// Expected formats:
// - nnnn.mm.nn[.comment].(up|down).sql
// where nnnn is the file number, mm is the major version, and nn is the minor version
func ParseVersionFromFilename(filename string) (*Version, error) {
	re := regexp.MustCompile(`^(\d{4})\.(\d{2})\.(\d{2})`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid filename format: %s", filename)
	}

	fileNumber, _ := strconv.Atoi(matches[1])
	major, _ := strconv.Atoi(matches[2])
	minor, _ := strconv.Atoi(matches[3])

	return NewVersion(major, minor, fileNumber), nil
}

// GetCommentFromFilename extracts the comment part from a filename
// Expected format: nnnn.mm.nn[.comment].(up|down).sql
func GetCommentFromFilename(filename string) string {
	// Remove up/down.sql part
	base := strings.TrimSuffix(filename, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")

	parts := strings.Split(base, ".")

	// Skip the first three parts (file number, major, minor)
	if len(parts) > 3 {
		return strings.Join(parts[3:], ".")
	}

	return ""
}

// IsUpMigration checks if the filename is an up migration
func IsUpMigration(filename string) bool {
	return strings.HasSuffix(filename, ".up.sql")
}

// IsDownMigration checks if the filename is a down migration
func IsDownMigration(filename string) bool {
	return strings.HasSuffix(filename, ".down.sql")
}

// GetBaseName gets the base name of a migration file without the up/down suffix
func GetBaseName(filename string) string {
	if IsUpMigration(filename) {
		return strings.TrimSuffix(filename, ".up.sql")
	} else if IsDownMigration(filename) {
		return strings.TrimSuffix(filename, ".down.sql")
	}
	return filename
}
