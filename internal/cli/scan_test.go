package cli

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/pdf"
	"github.com/slobbe/collate/internal/scanner"
)

type scanTestResult struct {
	documentPath string
	saveErr      error
	savedPaths   []string
	closed       bool
}

func (r *scanTestResult) SavePDF(_ context.Context, path string) error {
	r.savedPaths = append(r.savedPaths, path)
	if r.saveErr != nil {
		return r.saveErr
	}
	if r.documentPath == "" {
		return nil
	}

	contents, err := os.ReadFile(r.documentPath)
	if err != nil {
		return err
	}
	return os.WriteFile(path, contents, 0o600)
}

func (r *scanTestResult) Close() error {
	r.closed = true
	return nil
}

type scanTestScanner struct {
	info        scanner.Info
	scanErr     error
	scanOptions []scanner.ScanOptions
	results     []*scanTestResult
	nextResult  int
}

func (s *scanTestScanner) Info() scanner.Info {
	return s.info
}

func (s *scanTestScanner) Capabilities(context.Context) (scanner.Capabilities, error) {
	return scanner.Capabilities{}, nil
}

func (s *scanTestScanner) Scan(_ context.Context, options scanner.ScanOptions) (scanner.ScanResult, error) {
	s.scanOptions = append(s.scanOptions, options)
	if s.scanErr != nil {
		return nil, s.scanErr
	}
	if s.nextResult == len(s.results) {
		s.results = append(s.results, &scanTestResult{})
	}
	result := s.results[s.nextResult]
	s.nextResult++
	return result, nil
}

var testCapabilities = scanner.Capabilities{
	Sources: []scanner.Source{
		{
			ID:          "Platen",
			Name:        "Flatbed",
			ColorModes:  []string{"RGB24", "Grayscale8"},
			Resolutions: []int{300, 600},
		},
		{
			ID:          "ADF Simplex",
			Name:        "ADF (single-sided)",
			Feeder:      true,
			ColorModes:  []string{"RGB24", "Grayscale8"},
			Resolutions: []int{300, 600},
		},
	},
}

func TestRunScanUsesOnlyScannerAndDefaultOptions(t *testing.T) {
	front := &scanTestResult{}
	selected := &scanTestScanner{info: scanner.Info{ID: "scanner-1", Name: "Scanner One"}, results: []*scanTestResult{front}}

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\nn\n\n",
		startFor(selected),
		func(context.Context) ([]scanner.Info, error) {
			return []scanner.Info{selected.info}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := selected.scanOptions, []scanner.ScanOptions{{Source: "Platen", Paper: scanner.PaperA4, Mode: "RGB24", Resolution: 300}}; !equalOptions(got, want) {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if got, want := filepath.Base(front.savedPaths[0]), "scan_20260726_123456.pdf"; got != want {
		t.Fatalf("output name = %q, want %q", got, want)
	}
	if !front.closed {
		t.Fatal("front result was not closed")
	}
	if !strings.Contains(stdout, "Using scanner: Scanner One") || !strings.Contains(stdout, "Load the front pages.") || !strings.Contains(stdout, "Output path [scan_20260726_123456.pdf]:") {
		t.Fatalf("stdout = %q, want guided scan prompts", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanLetsUserSelectScannerAndOptions(t *testing.T) {
	selected := &scanTestScanner{info: scanner.Info{ID: "scanner-2", Name: "Scanner Two"}}
	code, stdout, stderr := runScanCommand(
		nil,
		"2\n2\nletter\n2\n2\n\nn\ncustom.pdf\n",
		startFor(selected),
		func(context.Context) ([]scanner.Info, error) {
			return []scanner.Info{{ID: "scanner-1", Name: "Scanner One"}, selected.info}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	want := scanner.ScanOptions{Source: "ADF Simplex", Paper: scanner.PaperLetter, Mode: "Grayscale8", Resolution: 600}
	if got := selected.scanOptions[0]; got != want {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if !strings.Contains(stdout, "Available scanners:\n  1. Scanner One\n  2. Scanner Two") {
		t.Fatalf("stdout = %q, want scanner selection", stdout)
	}
	if got := filepath.Base(selected.results[0].savedPaths[0]); got != "custom.pdf" {
		t.Fatalf("output name = %q, want custom.pdf", got)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanCollatesBackPagesBeforeAskingForOutput(t *testing.T) {
	dir := t.TempDir()
	front := &scanTestResult{documentPath: createTestPDF(t, dir, "front.pdf")}
	back := &scanTestResult{documentPath: createTestPDF(t, dir, "back.pdf")}
	selected := &scanTestScanner{info: scanner.Info{ID: "scanner-1"}, results: []*scanTestResult{front, back}}
	outputPath := filepath.Join(dir, "document.pdf")

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\ny\n\n\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]scanner.Info, error) { return []scanner.Info{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	output, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.PageCount(), 2; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
	if outputPrompt, backPrompt := strings.Index(stdout, "Output path ["), strings.Index(stdout, "Load the back pages."); outputPrompt < backPrompt {
		t.Fatalf("output was requested before back scan: %q", stdout)
	}
	if !front.closed || !back.closed {
		t.Fatal("scan results were not closed")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanReportsNoScanners(t *testing.T) {
	code, _, stderr := runScanCommand(
		nil,
		"",
		func(context.Context, string, scanner.ScanOptions) (*collate.ScanSession, error) {
			t.Fatal("scan start was called")
			return nil, nil
		},
		func(context.Context) ([]scanner.Info, error) { return nil, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 1 {
		t.Fatalf("RunScan() exit code = %d, want 1", code)
	}
	if stderr != "error: no scanners found\n" {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestRunScanListsAvailableScanners(t *testing.T) {
	code, stdout, stderr := runScanCommand(
		[]string{"--device-list"},
		"",
		func(context.Context, string, scanner.ScanOptions) (*collate.ScanSession, error) {
			t.Fatal("scan start was called")
			return nil, nil
		},
		func(context.Context) ([]scanner.Info, error) {
			return []scanner.Info{{Name: "Scanner One", ID: "scanner-1"}, {Name: "Scanner Two", ID: "scanner-2"}}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	const want = "Available scanners:\n- Scanner One\n  device: scanner-1\n- Scanner Two\n  device: scanner-2\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanReportsScanFailure(t *testing.T) {
	selected := &scanTestScanner{info: scanner.Info{ID: "scanner-1"}, scanErr: errors.New("device failure")}
	code, _, stderr := runScanCommand(
		nil,
		"\n\n\n\n\n",
		startFor(selected),
		func(context.Context) ([]scanner.Info, error) { return []scanner.Info{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 1 {
		t.Fatalf("RunScan() exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "error: device failure") {
		t.Fatalf("stderr = %q, want scan error", stderr)
	}
}

func createTestPDF(t *testing.T, dir, name string) string {
	t.Helper()

	imagePath := filepath.Join(dir, name+".png")
	imageFile, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(imageFile, image.NewGray(image.Rect(0, 0, 100, 200))); err != nil {
		imageFile.Close()
		t.Fatal(err)
	}
	if err := imageFile.Close(); err != nil {
		t.Fatal(err)
	}

	pdfPath := filepath.Join(dir, name)
	if err := pdf.ImportImages([]string{imagePath}, pdfPath); err != nil {
		t.Fatal(err)
	}
	return pdfPath
}

func startFor(device scanner.Scanner) startScan {
	return func(ctx context.Context, deviceID string, options scanner.ScanOptions) (*collate.ScanSession, error) {
		if device.Info().ID != deviceID {
			return nil, errors.New("scanner not found")
		}
		return collate.StartScan(ctx, device, options)
	}
}

func capabilitiesFor(capabilities scanner.Capabilities) scannerCapabilities {
	return func(context.Context, string) (scanner.Capabilities, error) {
		return capabilities, nil
	}
}

func runScanCommand(
	args []string,
	input string,
	start startScan,
	discover discoverScannerInfos,
	capabilities scannerCapabilities,
) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runScan(
		context.Background(), args, strings.NewReader(input), &stdout, &stderr, start, discover, capabilities,
		func() time.Time { return time.Date(2026, time.July, 26, 12, 34, 56, 0, time.Local) },
	)
	return code, stdout.String(), stderr.String()
}

func equalOptions(left, right []scanner.ScanOptions) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
