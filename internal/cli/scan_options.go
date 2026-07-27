package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/slobbe/collate/internal/collate"
)

func resolveScanner(ctx context.Context, input *bufio.Reader, output io.Writer, infos []collate.ScannerInfo, value string) (collate.ScannerInfo, error) {
	if value == "" {
		return selectScanner(ctx, input, output, infos)
	}

	var matches []collate.ScannerInfo
	for _, info := range infos {
		if matchesValue(value, info.ID, info.Name) {
			matches = append(matches, info)
		}
	}
	switch len(matches) {
	case 0:
		return collate.ScannerInfo{}, fmt.Errorf("scanner %q not found", value)
	case 1:
		fmt.Fprintf(output, "Using scanner: %s\n", scannerName(matches[0]))
		return matches[0], nil
	default:
		return collate.ScannerInfo{}, fmt.Errorf("scanner %q is ambiguous", value)
	}
}

func resolveScanOptions(
	ctx context.Context,
	input *bufio.Reader,
	output io.Writer,
	capabilities collate.Capabilities,
	sourceValue, paperValue, modeValue string,
	resolutionValue int,
) (collate.ScanOptions, error) {
	var source collate.ScanSource
	var err error
	if sourceValue == "" {
		source, err = selectSource(ctx, input, output, capabilities.Sources)
	} else {
		source, err = resolveSource(capabilities.Sources, sourceValue)
	}
	if err != nil {
		return collate.ScanOptions{}, err
	}

	var paper collate.Paper
	if paperValue == "" {
		paper, err = promptPaper(ctx, input, output)
	} else {
		paper, err = parsePaper(paperValue)
	}
	if err != nil {
		return collate.ScanOptions{}, err
	}

	var mode string
	if modeValue == "" {
		mode, err = selectMode(ctx, input, output, source.ColorModes)
	} else {
		mode, err = resolveMode(source.ColorModes, modeValue)
	}
	if err != nil {
		return collate.ScanOptions{}, err
	}

	resolution := resolutionValue
	if resolution == 0 {
		resolution, err = selectResolution(ctx, input, output, source.Resolutions)
	} else if !slices.Contains(source.Resolutions, resolution) {
		err = fmt.Errorf("source %q does not support %d DPI", source.ID, resolution)
	}
	if err != nil {
		return collate.ScanOptions{}, err
	}

	return collate.ScanOptions{Source: source.ID, Paper: paper, Mode: mode, Resolution: resolution}, nil
}

func resolveSource(sources []collate.ScanSource, value string) (collate.ScanSource, error) {
	var matches []collate.ScanSource
	for _, source := range sources {
		if matchesValue(value, source.ID, source.Name) {
			matches = append(matches, source)
		}
	}
	switch len(matches) {
	case 0:
		return collate.ScanSource{}, fmt.Errorf("scan source %q is not supported", value)
	case 1:
		return matches[0], nil
	default:
		return collate.ScanSource{}, fmt.Errorf("scan source %q is ambiguous", value)
	}
}

func resolveMode(modes []string, value string) (string, error) {
	var matches []string
	for _, mode := range modes {
		if matchesValue(value, mode) {
			matches = append(matches, mode)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("color mode %q is not supported", value)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("color mode %q is ambiguous", value)
	}
}

func matchesValue(value string, candidates ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range candidates {
		if strings.ToLower(strings.TrimSpace(candidate)) == value {
			return true
		}
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), value) {
			return true
		}
	}
	return false
}

func parsePaper(value string) (collate.Paper, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "a4", "din-a4":
		return collate.PaperA4, nil
	case "a5", "din-a5":
		return collate.PaperA5, nil
	case "letter":
		return collate.PaperLetter, nil
	default:
		return collate.Paper{}, fmt.Errorf("invalid paper format %q", value)
	}
}
