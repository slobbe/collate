package scanner

import "context"

// Info identifies a scanner and provides its user-facing name.
type Info struct {
	ID   string
	Name string
}

// Scanner acquires scans from a physical or network device.
type Scanner interface {
	Info() Info
	Capabilities(ctx context.Context) (Capabilities, error)
	Scan(ctx context.Context, options ScanOptions) (ScanResult, error)
}

// ScanResult owns the files produced by a single scan operation.
type ScanResult interface {
	SavePDF(ctx context.Context, path string) error
	Close() error
}

type Capabilities struct {
	Sources []Source
}

// Source is a scanner-specific input source.
type Source struct {
	ID     string
	Name   string
	Feeder bool
	Duplex bool

	ColorModes  []string
	Resolutions []int // DPI
	Dimensions  struct {
		MinWidth  int // micrometres
		MinHeight int // micrometres
		MaxWidth  int // micrometres
		MaxHeight int // micrometres
	}
}

type ScanOptions struct {
	Source     string
	Mode       string
	Paper      Paper
	Resolution int // DPI
}

type Paper struct {
	ID   string
	Name string

	WidthMicrometres  int
	HeightMicrometres int
}

var (
	PaperA4 = Paper{
		ID:                "a4",
		Name:              "A4",
		WidthMicrometres:  210_000,
		HeightMicrometres: 297_000,
	}

	PaperA5 = Paper{
		ID:                "a5",
		Name:              "A5",
		WidthMicrometres:  148_000,
		HeightMicrometres: 210_000,
	}

	PaperLetter = Paper{
		ID:                "letter",
		Name:              "Letter",
		WidthMicrometres:  215_900,
		HeightMicrometres: 279_400,
	}
)
