package collate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ScanSession coordinates front and optional back scans for one document.
type ScanSession struct {
	scanner Scanner
	options ScanOptions
	front   ScanResult
	back    ScanResult
}

// StartScan acquires the front pages of a document.
func StartScan(ctx context.Context, device Scanner, options ScanOptions) (*ScanSession, error) {
	if device == nil {
		return nil, fmt.Errorf("scanner is required")
	}

	front, err := device.Scan(ctx, options)
	if err != nil {
		return nil, err
	}

	return &ScanSession{
		scanner: device,
		options: options,
		front:   front,
	}, nil
}

// ScanBack acquires the back pages of the document.
func (s *ScanSession) ScanBack(ctx context.Context) error {
	if s.back != nil {
		return fmt.Errorf("back pages have already been scanned")
	}

	back, err := s.scanner.Scan(ctx, s.options)
	if err != nil {
		return err
	}
	s.back = back
	return nil
}

// SaveFront saves the front pages without duplex collation.
func (s *ScanSession) SaveFront(ctx context.Context, outputPath string) error {
	if s.front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	return s.front.SavePDF(ctx, outputPath)
}

// Collate saves the front and back pages temporarily, then creates a duplex PDF.
func (s *ScanSession) Collate(ctx context.Context, outputPath string, backOrder BackOrder) error {
	if s.front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	if s.back == nil {
		return fmt.Errorf("back pages have not been scanned")
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-duplex-")
	if err != nil {
		return fmt.Errorf("create temporary scan directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	frontPath := filepath.Join(temporaryDir, "front.pdf")
	if err := s.front.SavePDF(ctx, frontPath); err != nil {
		return fmt.Errorf("save front pages: %w", err)
	}

	backPath := filepath.Join(temporaryDir, "back.pdf")
	if err := s.back.SavePDF(ctx, backPath); err != nil {
		return fmt.Errorf("save back pages: %w", err)
	}

	return Collate(ctx, frontPath, backPath, outputPath, backOrder)
}

// Close releases resources held by the front and back scan results.
func (s *ScanSession) Close() error {
	var errs []error
	if s.front != nil {
		errs = append(errs, s.front.Close())
	}
	if s.back != nil {
		errs = append(errs, s.back.Close())
	}
	return errors.Join(errs...)
}
