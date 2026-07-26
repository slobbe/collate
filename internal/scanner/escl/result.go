package escl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/slobbe/collate/internal/scanner"
)

type scanResult struct {
	path string
}

func (r *scanResult) SavePDF(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || r.path == "" {
		return fmt.Errorf("no scanned document to save")
	}
	if path == "" {
		return fmt.Errorf("output path is required")
	}

	source, err := os.Open(r.path)
	if err != nil {
		return fmt.Errorf("open scanned document: %w", err)
	}
	defer source.Close()

	temporary, err := os.CreateTemp(filepath.Dir(path), ".collate-scan-*.pdf")
	if err != nil {
		return fmt.Errorf("create temporary output file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := io.Copy(temporary, source); err != nil {
		temporary.Close()
		return fmt.Errorf("copy scanned document: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save scanned document: %w", err)
	}

	return nil
}

func (r *scanResult) Close() error {
	if r == nil || r.path == "" {
		return nil
	}

	err := os.Remove(r.path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err == nil {
		r.path = ""
	}
	return err
}

var _ scanner.ScanResult = (*scanResult)(nil)
