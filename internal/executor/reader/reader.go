package reader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AlonMell/migrator/pkg/types"
)

type Reader struct {
	files   []*types.File
	path    string
	current int
}

func New(files []*types.File, path string) *Reader {
	return &Reader{
		files: files,
		path:  path,
	}
}

// ReadFile reads the content of a file
func (r *Reader) ReadFile() ([]byte, error) {
	filePath := filepath.Join(r.path, r.files[r.current].Name)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	r.current++
	return content, nil
}
