package version

import (
	"fmt"
)

// Version represents a migration version with major, minor version and file number
type Version struct {
	Major      int
	Minor      int
	FileNumber int
}

// New creates a new Version instance
func New(major, minor, fileNumber int) *Version {
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
