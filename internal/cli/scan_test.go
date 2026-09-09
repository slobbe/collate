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
	info        collate.ScannerInfo
	scanErr     error
	scanErrors  []error
	scanOptions []collate.ScanOptions
	results     []*scanTestResult
	nextResult  int
}

func (s *scanTestScanner) Info() collate.ScannerInfo {
	return collate.ScannerInfo{ID: s.info.ID, Name: s.info.Name}
}

func (s *scanTestScanner) Capabilities(context.Context) (collate.Capabilities, error) {
	return collate.Capabilities{}, nil
}

func (s *scanTestScanner) Scan(_ context.Context, options collate.ScanOptions) (collate.ScanResult, error) {
	s.scanOptions = append(s.scanOptions, options)
	if s.scanErr != nil {
		return nil, s.scanErr
	}
	callIndex := len(s.scanOptions) - 1
	if callIndex < len(s.scanErrors) && s.scanErrors[callIndex] != nil {
		return nil, s.scanErrors[callIndex]
	}
	if s.nextResult == len(s.results) {
		s.results = append(s.results, &scanTestResult{})
	}
	result := s.results[s.nextResult]
	s.nextResult++
	return result, nil
}

var testCapabilities = collate.Capabilities{
	Sources: []collate.ScanSource{
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

func TestResolversPreferExactMatchOverPartialMatches(t *testing.T) {
	t.Run("scanner", func(t *testing.T) {
		infos := []collate.ScannerInfo{{ID: "scan"}, {ID: "scanner"}}
		var output bytes.Buffer

		got, err := resolveScanner(context.Background(), nil, &output, infos, " SCAN ")
		if err != nil {
			t.Fatal(err)
		}
		if got != infos[0] {
			t.Fatalf("resolveScanner() = %#v, want %#v", got, infos[0])
		}
	})

	t.Run("source", func(t *testing.T) {
		sources := []collate.ScanSource{{ID: "ADF"}, {ID: "ADF Duplex"}}

		got, err := resolveSource(sources, " adf ")
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != sources[0].ID {
			t.Fatalf("resolveSource() = %#v, want %#v", got, sources[0])
		}
	})

	t.Run("mode", func(t *testing.T) {
		modes := []string{"Gray", "Grayscale8"}

		got, err := resolveMode(modes, " gray ")
		if err != nil {
			t.Fatal(err)
		}
		if got != modes[0] {
			t.Fatalf("resolveMode() = %q, want %q", got, modes[0])
		}
	})
}

func TestRunScanUsesOnlyScannerAndDefaultOptions(t *testing.T) {
	front := &scanTestResult{}
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1", Name: "Scanner One"}, results: []*scanTestResult{front}}

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\nn\nn\n\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) {
			return []collate.ScannerInfo{selected.info}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := selected.scanOptions, []collate.ScanOptions{{Source: "Platen", Paper: collate.PaperA4, Mode: "RGB24", Resolution: 300}}; !equalOptions(got, want) {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if got, want := filepath.Base(front.savedPaths[0]), "scan_20260726_123456.pdf"; got != want {
		t.Fatalf("output name = %q, want %q", got, want)
	}
	if !front.closed {
		t.Fatal("front result was not closed")
	}
	if !strings.Contains(stdout, "Using scanner: Scanner One") || !strings.Contains(stdout, "Load the front pages: Ready") || !strings.Contains(stdout, "Output path [scan_20260726_123456.pdf]:") {
		t.Fatalf("stdout = %q, want guided scan prompts", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanLetsUserSelectScannerAndOptions(t *testing.T) {
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-2", Name: "Scanner Two"}}
	code, stdout, stderr := runScanCommand(
		nil,
		"2\n2\n3\n2\n2\n\nn\nn\ncustom.pdf\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) {
			return []collate.ScannerInfo{{ID: "scanner-1", Name: "Scanner One"}, selected.info}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	want := collate.ScanOptions{Source: "ADF Simplex", Paper: collate.PaperLetter, Mode: "Grayscale8", Resolution: 600}
	if got := selected.scanOptions[0]; got != want {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if !strings.Contains(stdout, "[1] Scanner One") || !strings.Contains(stdout, "[2] Scanner Two") || !strings.Contains(stdout, "Select scanner: Scanner Two") {
		t.Fatalf("stdout = %q, want scanner selection", stdout)
	}
	if got := filepath.Base(selected.results[0].savedPaths[0]); got != "custom.pdf" {
		t.Fatalf("output name = %q, want custom.pdf", got)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanFlagsSkipScannerOptionAndOutputPrompts(t *testing.T) {
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-2", Name: "Scanner Two"}}
	outputPath := filepath.Join(t.TempDir(), "flagged.pdf")

	code, stdout, stderr := runScanCommand(
		[]string{
			"--device", "scanner-2",
			"--source", "adf",
			"--paper", "letter",
			"--mode", "gray",
			"--resolution", "600",
			"--output", outputPath,
		},
		"\nn\nn\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) {
			return []collate.ScannerInfo{{ID: "scanner-1", Name: "Scanner One"}, selected.info}, nil
		},
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0; stderr = %q", code, stderr)
	}
	want := collate.ScanOptions{Source: "ADF Simplex", Paper: collate.PaperLetter, Mode: "Grayscale8", Resolution: 600}
	if got := selected.scanOptions[0]; got != want {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	for _, prompt := range []string{"Select scanner [", "Select source [", "Select paper [", "Select color mode [", "Select resolution [", "Output path ["} {
		if strings.Contains(stdout, prompt) {
			t.Fatalf("stdout contains skipped prompt %q: %q", prompt, stdout)
		}
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanFlagRejectsUnsupportedResolution(t *testing.T) {
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1"}}
	code, _, stderr := runScanCommand(
		[]string{"--source", "platen", "--paper", "a4", "--mode", "RGB24", "--resolution", "1200"},
		"",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 1 {
		t.Fatalf("RunScan() exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, `source "Platen" does not support 1200 DPI`) {
		t.Fatalf("stderr = %q", stderr)
	}
	if len(selected.scanOptions) != 0 {
		t.Fatal("scan started with unsupported resolution")
	}
}

func TestRunScanCollatesBackPagesBeforeAskingForOutput(t *testing.T) {
	dir := t.TempDir()
	front := &scanTestResult{documentPath: createTestPDF(t, dir, "front.pdf")}
	back := &scanTestResult{documentPath: createTestPDF(t, dir, "back.pdf")}
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1"}, results: []*scanTestResult{front, back}}
	outputPath := filepath.Join(dir, "document.pdf")

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\ny\n\n\nn\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
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
	if outputPrompt, backPrompt := strings.Index(stdout, "Output path ["), strings.Index(stdout, "Load the back pages: Ready"); outputPrompt < backPrompt {
		t.Fatalf("output was requested before back scan: %q", stdout)
	}
	if !front.closed || !back.closed {
		t.Fatal("scan results were not closed")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanCombinesMultipleDuplexChunks(t *testing.T) {
	dir := t.TempDir()
	frontOne := &scanTestResult{documentPath: createTestPDF(t, dir, "front-one.pdf")}
	backOne := &scanTestResult{documentPath: createTestPDF(t, dir, "back-one.pdf")}
	frontTwo := &scanTestResult{documentPath: createTestPDF(t, dir, "front-two.pdf")}
	backTwo := &scanTestResult{documentPath: createTestPDF(t, dir, "back-two.pdf")}
	results := []*scanTestResult{frontOne, backOne, frontTwo, backTwo}
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1"}, results: results}
	outputPath := filepath.Join(dir, "document.pdf")

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\ny\n\n\ny\n\ny\n\n\nn\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0; stderr = %q", code, stderr)
	}
	output, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.PageCount(), 4; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
	wantOptions := []collate.ScanOptions{
		{Source: "Platen", Paper: collate.PaperA4, Mode: "RGB24", Resolution: 300},
		{Source: "Platen", Paper: collate.PaperA4, Mode: "RGB24", Resolution: 300},
		{Source: "Platen", Paper: collate.PaperA4, Mode: "RGB24", Resolution: 300},
		{Source: "Platen", Paper: collate.PaperA4, Mode: "RGB24", Resolution: 300},
	}
	if !equalOptions(selected.scanOptions, wantOptions) {
		t.Fatalf("scan options = %#v, want %#v", selected.scanOptions, wantOptions)
	}
	if got := strings.Count(stdout, "Scan another chunk? [y/N]"); got != 2 {
		t.Fatalf("another chunk prompt count = %d, want 2; stdout = %q", got, stdout)
	}
	if secondFront, firstAnother := strings.LastIndex(stdout, "Load the front pages: Ready"), strings.Index(stdout, "Scan another chunk?"); secondFront < firstAnother {
		t.Fatalf("second front scan was prompted before asking for another chunk: %q", stdout)
	}
	for index, result := range results {
		if !result.closed {
			t.Fatalf("scan result %d was not closed", index+1)
		}
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanCombinesSimplexAndDuplexChunks(t *testing.T) {
	dir := t.TempDir()
	results := []*scanTestResult{
		{documentPath: createTestPDF(t, dir, "simplex.pdf")},
		{documentPath: createTestPDF(t, dir, "duplex-front.pdf")},
		{documentPath: createTestPDF(t, dir, "duplex-back.pdf")},
	}
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1"}, results: results}
	outputPath := filepath.Join(dir, "document.pdf")

	code, _, stderr := runScanCommand(
		nil,
		"\n\n\n\n\nn\ny\n\ny\n2\n\nn\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0; stderr = %q", code, stderr)
	}
	output, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.PageCount(), 3; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
	for index, result := range results {
		if !result.closed {
			t.Fatalf("scan result %d was not closed", index+1)
		}
	}
}

func TestRunScanRetriesFailedChunkWithoutLosingCompletedPages(t *testing.T) {
	dir := t.TempDir()
	first := &scanTestResult{documentPath: createTestPDF(t, dir, "first.pdf")}
	second := &scanTestResult{documentPath: createTestPDF(t, dir, "second.pdf")}
	selected := &scanTestScanner{
		info:       collate.ScannerInfo{ID: "scanner-1"},
		results:    []*scanTestResult{first, second},
		scanErrors: []error{nil, errors.New("temporary timeout"), nil},
	}
	outputPath := filepath.Join(dir, "document.pdf")

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\nn\ny\n\n\n\nn\nn\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0; stderr = %q", code, stderr)
	}
	output, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.PageCount(), 2; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
	if !strings.Contains(stdout, "Scanning front pages failed: temporary timeout") || !strings.Contains(stdout, "Retry scanning front pages?") {
		t.Fatalf("stdout = %q, want retry guidance", stdout)
	}
	if !first.closed || !second.closed {
		t.Fatal("successful scan results were not closed")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanSavesCompletedPagesWhenRetryIsDeclined(t *testing.T) {
	dir := t.TempDir()
	first := &scanTestResult{documentPath: createTestPDF(t, dir, "first.pdf")}
	selected := &scanTestScanner{
		info:       collate.ScannerInfo{ID: "scanner-1"},
		results:    []*scanTestResult{first},
		scanErrors: []error{nil, errors.New("temporary timeout")},
	}
	outputPath := filepath.Join(dir, "recovered.pdf")

	code, stdout, stderr := runScanCommand(
		nil,
		"\n\n\n\n\nn\ny\n\nn\n"+outputPath+"\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
		capabilitiesFor(testCapabilities),
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0; stderr = %q", code, stderr)
	}
	output, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.PageCount(), 1; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
	if !strings.Contains(stdout, "Saved completed pages successfully: "+outputPath) {
		t.Fatalf("stdout = %q, want recovered-pages confirmation", stdout)
	}
	if !first.closed {
		t.Fatal("successful scan result was not closed")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanReportsNoScanners(t *testing.T) {
	code, _, stderr := runScanCommand(
		nil,
		"",
		func(context.Context, string, collate.ScanOptions) (*collate.ScanSession, error) {
			t.Fatal("scan start was called")
			return nil, nil
		},
		func(context.Context) ([]collate.ScannerInfo, error) { return nil, nil },
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
		func(context.Context, string, collate.ScanOptions) (*collate.ScanSession, error) {
			t.Fatal("scan start was called")
			return nil, nil
		},
		func(context.Context) ([]collate.ScannerInfo, error) {
			return []collate.ScannerInfo{{Name: "Scanner One", ID: "scanner-1"}, {Name: "Scanner Two", ID: "scanner-2"}}, nil
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
	selected := &scanTestScanner{info: collate.ScannerInfo{ID: "scanner-1"}, scanErr: errors.New("device failure")}
	code, _, stderr := runScanCommand(
		nil,
		"\n\n\n\n\n",
		startFor(selected),
		func(context.Context) ([]collate.ScannerInfo, error) { return []collate.ScannerInfo{selected.info}, nil },
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

func startFor(device collate.Scanner) startScan {
	return func(ctx context.Context, deviceID string, options collate.ScanOptions) (*collate.ScanSession, error) {
		if device.Info().ID != deviceID {
			return nil, errors.New("scanner not found")
		}
		return collate.StartScan(ctx, device, options)
	}
}

func capabilitiesFor(capabilities collate.Capabilities) scannerCapabilities {
	return func(context.Context, string) (collate.Capabilities, error) {
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

func equalOptions(left, right []collate.ScanOptions) bool {
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
