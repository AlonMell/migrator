package reader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AlonMell/migrator/internal/domain/types"
)

type Reader struct {
	path string
}

func New(path string) *Reader {
	return &Reader{
		path: path,
	}
}

func (r *Reader) GetFileNames(ctx context.Context) ([]string, error) {
	files, err := os.ReadDir(r.path)
	if err != nil {
		return nil, err
	}

	var names []string

	for _, file := range files {
		names = append(names, file.Name())
	}

	return names, nil
}

// ReadFile reads the content of a file
func (r *Reader) ReadFile(ctx context.Context, fileInfo *types.FileInfo) ([]byte, error) {
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
