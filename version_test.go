package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_NewVersion(t *testing.T) {
	v := NewVersion(1, 2, 3)
	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 2, v.Minor)
	assert.Equal(t, 3, v.FileNumber)
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  *Version
		expected string
	}{
		{
			name:     "basic version",
			version:  NewVersion(1, 2, 3),
			expected: "01.02.0003",
		},
		{
			name:     "zero version",
			version:  NewVersion(0, 0, 0),
			expected: "00.00.0000",
		},
		{
			name:     "large numbers",
			version:  NewVersion(99, 99, 9999),
			expected: "99.99.9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_CompareTo(t *testing.T) {
	tests := []struct {
		name     string
		version1 *Version
		version2 *Version
		expected int
	}{
		{
			name:     "equal versions",
			version1: NewVersion(1, 2, 3),
			version2: NewVersion(1, 2, 3),
			expected: 0,
		},
		{
			name:     "higher major version",
			version1: NewVersion(2, 0, 0),
			version2: NewVersion(1, 5, 10),
			expected: 1,
		},
		{
			name:     "lower major version",
			version1: NewVersion(1, 5, 10),
			version2: NewVersion(2, 0, 0),
			expected: -1,
		},
		{
			name:     "equal major, higher minor version",
			version1: NewVersion(1, 3, 0),
			version2: NewVersion(1, 2, 10),
			expected: 1,
		},
		{
			name:     "equal major, lower minor version",
			version1: NewVersion(1, 2, 10),
			version2: NewVersion(1, 3, 0),
			expected: -1,
		},
		{
			name:     "equal major and minor, higher file number",
			version1: NewVersion(1, 2, 4),
			version2: NewVersion(1, 2, 3),
			expected: 1,
		},
		{
			name:     "equal major and minor, lower file number",
			version1: NewVersion(1, 2, 3),
			version2: NewVersion(1, 2, 4),
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version1.CompareTo(tt.version2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersionFromFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expected    *Version
		expectedErr bool
	}{
		{
			name:        "valid up migration",
			filename:    "0001.02.03.test.up.sql",
			expected:    NewVersion(2, 3, 1),
			expectedErr: false,
		},
		{
			name:        "valid down migration",
			filename:    "0002.01.05.another_test.down.sql",
			expected:    NewVersion(1, 5, 2),
			expectedErr: false,
		},
		{
			name:        "invalid format - missing parts",
			filename:    "001.02.test.sql",
			expected:    nil,
			expectedErr: true,
		},
		{
			name:        "invalid format - not a migration file",
			filename:    "some_random_file.txt",
			expected:    nil,
			expectedErr: true,
		},
		{
			name:        "invalid format - wrong number format",
			filename:    "abc.def.ghi.test.up.sql",
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := ParseVersionFromFilename(tt.filename)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, version)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Major, version.Major)
				assert.Equal(t, tt.expected.Minor, version.Minor)
				assert.Equal(t, tt.expected.FileNumber, version.FileNumber)
			}
		})
	}
}

func TestGetCommentFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "filename with comment - up migration",
			filename: "0001.02.03.test_comment.up.sql",
			expected: "test_comment",
		},
		{
			name:     "filename with multipart comment - down migration",
			filename: "0002.01.05.another.test.comment.down.sql",
			expected: "another.test.comment",
		},
		{
			name:     "filename without comment",
			filename: "0003.01.02.up.sql",
			expected: "",
		},
		{
			name:     "minimum valid filename - up",
			filename: "0001.00.00.up.sql",
			expected: "",
		},
		{
			name:     "minimum valid filename - down",
			filename: "0001.00.00.down.sql",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := GetCommentFromFilename(tt.filename)
			assert.Equal(t, tt.expected, comment)
		})
	}
}

func TestIsUpMigration(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "up migration",
			filename: "0001.00.01.test.up.sql",
			expected: true,
		},
		{
			name:     "down migration",
			filename: "0001.00.01.test.down.sql",
			expected: false,
		},
		{
			name:     "not a migration file",
			filename: "something.sql",
			expected: false,
		},
		{
			name:     "up in filename but not suffix",
			filename: "0001.00.01.up_something.sql",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUpMigration(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDownMigration(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "down migration",
			filename: "0001.00.01.test.down.sql",
			expected: true,
		},
		{
			name:     "up migration",
			filename: "0001.00.01.test.up.sql",
			expected: false,
		},
		{
			name:     "not a migration file",
			filename: "something.sql",
			expected: false,
		},
		{
			name:     "down in filename but not suffix",
			filename: "0001.00.01.down_something.sql",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDownMigration(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBaseName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "up migration",
			filename: "0001.00.01.test.up.sql",
			expected: "0001.00.01.test",
		},
		{
			name:     "down migration",
			filename: "0001.00.01.test.down.sql",
			expected: "0001.00.01.test",
		},
		{
			name:     "not a migration file",
			filename: "something.sql",
			expected: "something.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBaseName(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
