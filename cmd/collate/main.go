package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/slobbe/collate/internal/collate"
)

func main() {
	backOrderFlag := flag.String(
		"back-order",
		string(collate.BackOrderReverse),
		"order of pages in back.pdf: reverse or forward",
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags] <front.pdf> <back.pdf> <output.pdf>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) != 3 {
		flag.Usage()
		os.Exit(2)
	}

	backOrder, err := collate.ParseBackOrder(*backOrderFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(2)
	}

	paths := make([]string, len(flag.Args()))
	for i, path := range flag.Args() {
		paths[i], err = normalizePDFPath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: normalize path %q: %v\n", path, err)
			os.Exit(2)
		}
	}

	if err := collate.Collate(paths[0], paths[1], paths[2], backOrder); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func normalizePDFPath(path string) (string, error) {
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
