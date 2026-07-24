// Package scanner discovers scanners available to collate.
package scanner

import (
	"context"
	"fmt"
	"strings"
)

// Info describes a discovered scanner.
type Info struct {
	Device      string
	Description string
}

// Scanner is a scanner discovered on the current system.
type Scanner interface {
	Info() Info
	Scan(ctx context.Context, options ScanOptions) error
	Save(ctx context.Context, outputPath string) error
	Close() error
}

// Paper identifies a physical paper format.
type Paper string

const PaperA4 Paper = "a4"

// ParsePaper validates a supported paper format.
func ParsePaper(value string) (Paper, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case "a4", "din-a4":
		return PaperA4, nil
	default:
		return "", fmt.Errorf("invalid paper format %q (must be %q)", value, PaperA4)
	}
}

// ScanOptions controls how a scanner acquires pages.
type ScanOptions struct {
	Source string
	Batch  bool
	Paper  Paper
}
