package reader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AlonMell/migrator/pkg/types"
)

type Reader struct {
	path string
}

func New(files []*types.File, path string) *Reader {
	return &Reader{
		path: path,
	}
}

// ReadFile reads the content of a file
func (r *Reader) ReadFile(ctx context.Context, fileInfo *types.File) ([]byte, error) {
	path := filepath.Join(r.path, fileInfo.Name)
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
