package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/scanner"
)

type scanTestScanner struct {
	id          string
	scanErr     error
	saveErr     error
	scanOptions []scanner.ScanOptions
	savedPaths  []string
	closed      bool
}

func (s *scanTestScanner) ID() string {
	return s.id
}

func (s *scanTestScanner) Capabilities(context.Context, bool) scanner.Capabilities {
	return scanner.Capabilities{}
}

func (s *scanTestScanner) Scan(_ context.Context, options scanner.ScanOptions) error {
	s.scanOptions = append(s.scanOptions, options)
	return s.scanErr
}

func (s *scanTestScanner) Save(_ context.Context, outputPath string) error {
	s.savedPaths = append(s.savedPaths, outputPath)
	return s.saveErr
}

func (s *scanTestScanner) Close(context.Context) error {
	s.closed = true
	return nil
}

type mergeCall struct {
	frontPath  string
	backPath   string
	outputPath string
	order      collate.BackOrder
}

func TestRunScanSavesFrontOnly(t *testing.T) {
	selected := &scanTestScanner{id: "test:0"}
	var merges []mergeCall
	code, stdout, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "document.pdf"},
		"\nn\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		func(_ context.Context, frontPath, backPath, outputPath string, order collate.BackOrder) error {
			merges = append(merges, mergeCall{frontPath, backPath, outputPath, order})
			return nil
		},
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := len(selected.scanOptions), 1; got != want {
		t.Fatalf("scan count = %d, want %d", got, want)
	}
	if got, want := len(selected.savedPaths), 1; got != want {
		t.Fatalf("save count = %d, want %d", got, want)
	}
	if filepath.Base(selected.savedPaths[0]) != "document.pdf" {
		t.Fatalf("saved path = %q, want document.pdf", selected.savedPaths[0])
	}
	if len(merges) != 0 {
		t.Fatalf("merge count = %d, want 0", len(merges))
	}
	if !selected.closed {
		t.Fatal("Close was not called")
	}
	if !strings.Contains(stdout, "Load the front pages.") || !strings.Contains(stdout, "scanned successfully:") {
		t.Fatalf("stdout = %q, want front prompt and success", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanCollatesBackPagesInReverseOrderByDefault(t *testing.T) {
	selected := &scanTestScanner{id: "test:0"}
	var merges []mergeCall
	code, stdout, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "document.pdf"},
		"\ny\n\n\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		func(_ context.Context, frontPath, backPath, outputPath string, order collate.BackOrder) error {
			merges = append(merges, mergeCall{frontPath, backPath, outputPath, order})
			return nil
		},
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := len(selected.scanOptions), 2; got != want {
		t.Fatalf("scan count = %d, want %d", got, want)
	}
	if got, want := len(selected.savedPaths), 2; got != want {
		t.Fatalf("save count = %d, want %d", got, want)
	}
	if filepath.Base(selected.savedPaths[0]) != "front.pdf" || filepath.Base(selected.savedPaths[1]) != "back.pdf" {
		t.Fatalf("saved paths = %v, want front.pdf and back.pdf", selected.savedPaths)
	}
	if got, want := len(merges), 1; got != want {
		t.Fatalf("merge count = %d, want %d", got, want)
	}
	if merges[0].order != collate.BackOrderReverse {
		t.Fatalf("back order = %q, want %q", merges[0].order, collate.BackOrderReverse)
	}
	if filepath.Base(merges[0].outputPath) != "document.pdf" {
		t.Fatalf("merge output = %q, want document.pdf", merges[0].outputPath)
	}
	if !strings.Contains(stdout, "Load the back pages.") || !strings.Contains(stdout, "scanned and collated successfully:") {
		t.Fatalf("stdout = %q, want back prompt and duplex success", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanCollatesBackPagesInForwardOrder(t *testing.T) {
	selected := &scanTestScanner{id: "test:0"}
	var mergeOrder collate.BackOrder
	code, _, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "document.pdf"},
		"\ny\nforward\n\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		func(_ context.Context, _, _, _ string, order collate.BackOrder) error {
			mergeOrder = order
			return nil
		},
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if mergeOrder != collate.BackOrderForward {
		t.Fatalf("back order = %q, want %q", mergeOrder, collate.BackOrderForward)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanListsAvailableDevices(t *testing.T) {
	code, stdout, stderr := runScanCommand(
		[]string{"--device-list"},
		"",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{
				&scanTestScanner{id: "airscan:e0:OfficeJet"},
				&scanTestScanner{id: "test:0"},
			}, nil
		},
		nil,
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	const want = "Available scanners:\n- airscan:e0:OfficeJet\n- test:0\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanRejectsDeviceListWithScanOptions(t *testing.T) {
	code, _, stderr := runScanCommand(
		[]string{"--device-list", "--device", "test:0"},
		"",
		func(context.Context) ([]scanner.Scanner, error) {
			t.Fatal("discover was called")
			return nil, nil
		},
		nil,
	)

	if code != 2 {
		t.Fatalf("RunScan() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "error: --device-list cannot be combined with scan options") {
		t.Fatalf("stderr = %q, want device-list conflict error", stderr)
	}
}

func TestRunScanPassesSourceAndPaper(t *testing.T) {
	selected := &scanTestScanner{id: "test:0"}
	code, _, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "output.pdf", "--source", "ADF Duplex", "--paper", "a4"},
		"\nn\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		nil,
	)

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	want := scanner.ScanOptions{Source: "ADF Duplex", Paper: scanner.PaperA4}
	if got := selected.scanOptions[0]; got != want {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanReportsScanFailureAndCleansUp(t *testing.T) {
	selected := &scanTestScanner{id: "test:0", scanErr: errors.New("device failure")}
	code, _, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "output.pdf"},
		"\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		nil,
	)

	if code != 1 {
		t.Fatalf("RunScan() exit code = %d, want 1", code)
	}
	if !selected.closed {
		t.Fatal("Close was not called")
	}
	if !strings.Contains(stderr, "error: device failure") {
		t.Fatalf("stderr = %q, want scan error", stderr)
	}
}

func TestRunScanReportsCancellation(t *testing.T) {
	selected := &scanTestScanner{id: "test:0", scanErr: context.Canceled}
	code, _, stderr := runScanCommand(
		[]string{"--device", "test:0", "--output", "output.pdf"},
		"\n",
		func(context.Context) ([]scanner.Scanner, error) {
			return []scanner.Scanner{selected}, nil
		},
		nil,
	)

	if code != 130 {
		t.Fatalf("RunScan() exit code = %d, want 130", code)
	}
	if !selected.closed {
		t.Fatal("Close was not called")
	}
	if stderr != "interrupted\n" {
		t.Fatalf("stderr = %q, want interruption message", stderr)
	}
}

func runScanCommand(args []string, input string, discover discoverScanners, merge mergeScans) (int, string, string) {
	if merge == nil {
		merge = func(context.Context, string, string, string, collate.BackOrder) error {
			return nil
		}
	}

	var stdout, stderr bytes.Buffer
	code := runScan(context.Background(), args, strings.NewReader(input), &stdout, &stderr, discover, merge)
	return code, stdout.String(), stderr.String()
}
