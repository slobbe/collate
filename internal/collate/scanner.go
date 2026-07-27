package collate

import "context"

// ScannerInfo identifies a scanner available to Collate.
type ScannerInfo struct {
	ID   string
	Name string
}

// Scanner acquires scans from a physical or network device.
type Scanner interface {
	Info() ScannerInfo
	Capabilities(ctx context.Context) (Capabilities, error)
	Scan(ctx context.Context, options ScanOptions) (ScanResult, error)
}

// ScannerProvider discovers scanners supported by the active adapters.
type ScannerProvider interface {
	Discover(ctx context.Context) ([]Scanner, error)
}

// ScanResult owns files produced by a single scan operation.
type ScanResult interface {
	SavePDF(ctx context.Context, path string) error
	Close() error
}

// Capabilities describes the supported settings for a scanner.
type Capabilities struct {
	Sources []ScanSource
}

// ScanSource is a scanner input source and its supported settings.
type ScanSource struct {
	ID     string
	Name   string
	Feeder bool
	Duplex bool

	ColorModes  []string
	Resolutions []int
	Dimensions  ScanDimensions
}

// ScanDimensions describes the source's supported scan area in micrometres.
type ScanDimensions struct {
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
}

// ScanOptions describes one scan operation.
type ScanOptions struct {
	Source     string
	Mode       string
	Paper      Paper
	Resolution int
}

// Paper describes a paper format in micrometres.
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
