package collate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ScanSession coordinates one or more front and optional back scan chunks for one document.
type ScanSession struct {
	scanner Scanner
	options ScanOptions
	chunks  []scanChunk
}

type scanChunk struct {
	front     ScanResult
	back      ScanResult
	backOrder BackOrder
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
		chunks:  []scanChunk{{front: front}},
	}, nil
}

// ScanBack acquires reverse-ordered back pages for the current chunk.
func (s *ScanSession) ScanBack(ctx context.Context) error {
	return s.ScanBackInOrder(ctx, BackOrderReverse)
}

// ScanBackInOrder acquires back pages for the current chunk and records their order.
func (s *ScanSession) ScanBackInOrder(ctx context.Context, backOrder BackOrder) error {
	if _, err := ParseBackOrder(string(backOrder)); err != nil {
		return err
	}
	chunk := &s.chunks[len(s.chunks)-1]
	if chunk.back != nil {
		return fmt.Errorf("back pages have already been scanned")
	}

	back, err := s.scanner.Scan(ctx, s.options)
	if err != nil {
		return err
	}
	chunk.back = back
	chunk.backOrder = backOrder
	return nil
}

// ScanNextChunk acquires the front pages of the next document chunk.
func (s *ScanSession) ScanNextChunk(ctx context.Context) error {
	front, err := s.scanner.Scan(ctx, s.options)
	if err != nil {
		return err
	}
	s.chunks = append(s.chunks, scanChunk{front: front})
	return nil
}

// SaveFront saves the front pages without duplex collation.
func (s *ScanSession) SaveFront(ctx context.Context, outputPath string) error {
	if len(s.chunks) == 0 || s.chunks[0].front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	return s.chunks[0].front.SavePDF(ctx, outputPath)
}

// Collate saves the front and back pages temporarily, then creates a duplex PDF.
func (s *ScanSession) Collate(ctx context.Context, outputPath string, backOrder BackOrder) error {
	if len(s.chunks) == 0 || s.chunks[0].front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	if s.chunks[0].back == nil {
		return fmt.Errorf("back pages have not been scanned")
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-duplex-")
	if err != nil {
		return fmt.Errorf("create temporary scan directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	frontPath := filepath.Join(temporaryDir, "front.pdf")
	if err := s.chunks[0].front.SavePDF(ctx, frontPath); err != nil {
		return fmt.Errorf("save front pages: %w", err)
	}

	backPath := filepath.Join(temporaryDir, "back.pdf")
	if err := s.chunks[0].back.SavePDF(ctx, backPath); err != nil {
		return fmt.Errorf("save back pages: %w", err)
	}

	return Collate(ctx, frontPath, backPath, outputPath, backOrder)
}

// Save collates each chunk as needed and appends all chunks to one PDF.
func (s *ScanSession) Save(ctx context.Context, outputPath string) error {
	if len(s.chunks) == 0 {
		return fmt.Errorf("front pages have not been scanned")
	}
	if len(s.chunks) == 1 {
		if s.chunks[0].back == nil {
			return s.SaveFront(ctx, outputPath)
		}
		return s.Collate(ctx, outputPath, s.chunks[0].backOrder)
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-scan-")
	if err != nil {
		return fmt.Errorf("create temporary scan directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	chunkPaths := make([]string, 0, len(s.chunks))
	for index, chunk := range s.chunks {
		chunkPath := filepath.Join(temporaryDir, fmt.Sprintf("chunk-%d.pdf", index+1))
		if chunk.back == nil {
			if err := chunk.front.SavePDF(ctx, chunkPath); err != nil {
				return fmt.Errorf("save chunk %d: %w", index+1, err)
			}
		} else {
			frontPath := filepath.Join(temporaryDir, fmt.Sprintf("front-%d.pdf", index+1))
			if err := chunk.front.SavePDF(ctx, frontPath); err != nil {
				return fmt.Errorf("save chunk %d front pages: %w", index+1, err)
			}
			backPath := filepath.Join(temporaryDir, fmt.Sprintf("back-%d.pdf", index+1))
			if err := chunk.back.SavePDF(ctx, backPath); err != nil {
				return fmt.Errorf("save chunk %d back pages: %w", index+1, err)
			}
			if err := Collate(ctx, frontPath, backPath, chunkPath, chunk.backOrder); err != nil {
				return fmt.Errorf("collate chunk %d: %w", index+1, err)
			}
		}
		chunkPaths = append(chunkPaths, chunkPath)
	}

	return Merge(ctx, chunkPaths, outputPath)
}

// Close releases resources held by all scan results.
func (s *ScanSession) Close() error {
	var errs []error
	for _, chunk := range s.chunks {
		if chunk.front != nil {
			errs = append(errs, chunk.front.Close())
		}
		if chunk.back != nil {
			errs = append(errs, chunk.back.Close())
		}
	}
	return errors.Join(errs...)
}
