package scanner

import (
	"context"
	"fmt"
	"strings"
)

type Scanner interface {
	ID() string
	Capabilities(ctx context.Context, force bool) Capabilities
	Scan(ctx context.Context, options ScanOptions) error
	Save(ctx context.Context, path string) error
	Close(ctx context.Context) error
}

type Capabilities struct {
	Sources     []Source
	Papers      []Paper
	Modes       []Mode
	Resolutions []int
}

type ScanOptions struct {
	Source     Source
	Paper      Paper
	Mode       Mode
	Resolution int // DPI
}

// Source is an exact source value accepted by a scanner backend.
type Source string

type Paper string

const (
	PaperA4     Paper = "a4"
	PaperA5     Paper = "a5"
	PaperLetter Paper = "letter"
)

type Mode string

const (
	ModeColor Mode = "color"
	ModeGray  Mode = "gray"
)

func ParsePaper(value string) (Paper, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "a4", "din-a4":
		return PaperA4, nil
	case "a5", "din-a5":
		return PaperA5, nil
	case "letter":
		return PaperLetter, nil
	default:
		return "", fmt.Errorf("invalid paper format %q (must be %q)", value, PaperA4)
	}
}
