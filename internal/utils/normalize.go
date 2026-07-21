package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizePDFPath returns a clean absolute PDF path, expanding ~ and ~/.
func NormalizePDFPath(path string) (string, error) {
	normalized, err := normalizePath(path)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(normalized), ".pdf") {
		return "", fmt.Errorf("expected a .pdf file")
	}
	return normalized, nil
}

func normalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path must not be empty")
	}

	if path[0] == '~' {
		switch {
		case path == "~":
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			path = home
		case len(path) > 1 && os.IsPathSeparator(path[1]):
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			path = filepath.Join(home, path[2:])
		default:
			return "", fmt.Errorf("unsupported home path; use ~ or ~/path")
		}
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return filepath.Clean(absolute), nil
}
