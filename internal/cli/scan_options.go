package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/slobbe/collate/internal/scanner"
)

func resolveScanner(ctx context.Context, input *bufio.Reader, output io.Writer, infos []scanner.Info, value string) (scanner.Info, error) {
	if value == "" {
		return selectScanner(ctx, input, output, infos)
	}

	var matches []scanner.Info
	for _, info := range infos {
		if matchesValue(value, info.ID, info.Name) {
			matches = append(matches, info)
		}
	}
	switch len(matches) {
	case 0:
		return scanner.Info{}, fmt.Errorf("scanner %q not found", value)
	case 1:
		fmt.Fprintf(output, "Using scanner: %s\n", scannerName(matches[0]))
		return matches[0], nil
	default:
		return scanner.Info{}, fmt.Errorf("scanner %q is ambiguous", value)
	}
}

func resolveScanOptions(
	ctx context.Context,
	input *bufio.Reader,
	output io.Writer,
	capabilities scanner.Capabilities,
	sourceValue, paperValue, modeValue string,
	resolutionValue int,
) (scanner.ScanOptions, error) {
	var source scanner.Source
	var err error
	if sourceValue == "" {
		source, err = selectSource(ctx, input, output, capabilities.Sources)
	} else {
		source, err = resolveSource(capabilities.Sources, sourceValue)
	}
	if err != nil {
		return scanner.ScanOptions{}, err
	}

	var paper scanner.Paper
	if paperValue == "" {
		paper, err = promptPaper(ctx, input, output)
	} else {
		paper, err = parsePaper(paperValue)
	}
	if err != nil {
		return scanner.ScanOptions{}, err
	}

	var mode string
	if modeValue == "" {
		mode, err = selectMode(ctx, input, output, source.ColorModes)
	} else {
		mode, err = resolveMode(source.ColorModes, modeValue)
	}
	if err != nil {
		return scanner.ScanOptions{}, err
	}

	resolution := resolutionValue
	if resolution == 0 {
		resolution, err = selectResolution(ctx, input, output, source.Resolutions)
	} else if !slices.Contains(source.Resolutions, resolution) {
		err = fmt.Errorf("source %q does not support %d DPI", source.ID, resolution)
	}
	if err != nil {
		return scanner.ScanOptions{}, err
	}

	return scanner.ScanOptions{Source: source.ID, Paper: paper, Mode: mode, Resolution: resolution}, nil
}

func resolveSource(sources []scanner.Source, value string) (scanner.Source, error) {
	var matches []scanner.Source
	for _, source := range sources {
		if matchesValue(value, source.ID, source.Name) {
			matches = append(matches, source)
		}
	}
	switch len(matches) {
	case 0:
		return scanner.Source{}, fmt.Errorf("scan source %q is not supported", value)
	case 1:
		return matches[0], nil
	default:
		return scanner.Source{}, fmt.Errorf("scan source %q is ambiguous", value)
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

func parsePaper(value string) (scanner.Paper, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "a4", "din-a4":
		return scanner.PaperA4, nil
	case "a5", "din-a5":
		return scanner.PaperA5, nil
	case "letter":
		return scanner.PaperLetter, nil
	default:
		return scanner.Paper{}, fmt.Errorf("invalid paper format %q", value)
	}
}
