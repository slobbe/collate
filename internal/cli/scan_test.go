package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/slobbe/collate/internal/scanner"
)

type scanTestScanner struct {
	info        scanner.Info
	scanErr     error
	saveErr     error
	scanCalled  bool
	scanOptions scanner.ScanOptions
	savedPath   string
	closed      bool
}

func (scanner *scanTestScanner) Info() scanner.Info {
	return scanner.info
}

func (scanner *scanTestScanner) Scan(_ context.Context, options scanner.ScanOptions) error {
	scanner.scanCalled = true
	scanner.scanOptions = options
	return scanner.scanErr
}

func (scanner *scanTestScanner) Save(_ context.Context, outputPath string) error {
	scanner.savedPath = outputPath
	return scanner.saveErr
}

func (scanner *scanTestScanner) Close() error {
	scanner.closed = true
	return nil
}

func TestRunScanScansAndSavesSelectedDevice(t *testing.T) {
	selected := &scanTestScanner{info: scanner.Info{Device: "test:0"}}
	code, stdout, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "document.pdf",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{selected}, nil
	})

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if !selected.scanCalled {
		t.Fatal("Scan was not called")
	}
	if selected.scanOptions != (scanner.ScanOptions{}) {
		t.Fatalf("scan options = %#v, want defaults", selected.scanOptions)
	}
	if !strings.HasSuffix(selected.savedPath, "/document.pdf") {
		t.Fatalf("Save path = %q, want document.pdf", selected.savedPath)
	}
	if !selected.closed {
		t.Fatal("Close was not called")
	}
	if !strings.Contains(stdout, "scanned successfully:") {
		t.Fatalf("stdout = %q, want success message", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanRequiresFlags(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "device", args: []string{"--output", "output.pdf"}, want: "error: --device is required"},
		{name: "output", args: []string{"--device", "test:0"}, want: "error: --output is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, _, stderr := runScanCommand(test.args, func(context.Context) ([]scanner.Scanner, error) {
				t.Fatal("discover was called")
				return nil, nil
			})
			if code != 2 {
				t.Fatalf("RunScan() exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
		})
	}
}

func TestRunScanRequiresSourceForBatch(t *testing.T) {
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
		"--batch",
	}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 2 {
		t.Fatalf("RunScan() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "error: --source is required when --batch is used") {
		t.Fatalf("stderr = %q, want batch source error", stderr)
	}
}

func TestRunScanPassesFeederOptions(t *testing.T) {
	selected := &scanTestScanner{info: scanner.Info{Device: "test:0"}}
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
		"--source", "ADF",
		"--batch",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{selected}, nil
	})

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := selected.scanOptions, (scanner.ScanOptions{Source: "ADF", Batch: true}); got != want {
		t.Fatalf("scan options = %#v, want %#v", got, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanPassesPaperOption(t *testing.T) {
	selected := &scanTestScanner{info: scanner.Info{Device: "test:0"}}
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
		"--paper", "a4",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{selected}, nil
	})

	if code != 0 {
		t.Fatalf("RunScan() exit code = %d, want 0", code)
	}
	if got, want := selected.scanOptions.Paper, scanner.PaperA4; got != want {
		t.Fatalf("paper = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScanRejectsInvalidPaper(t *testing.T) {
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
		"--paper", "letter",
	}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 2 {
		t.Fatalf("RunScan() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: invalid paper format "letter"`) {
		t.Fatalf("stderr = %q, want invalid-paper error", stderr)
	}
}

func TestRunScanRejectsUnknownScanner(t *testing.T) {
	code, _, stderr := runScanCommand([]string{
		"--device", "missing",
		"--output", "output.pdf",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{&scanTestScanner{info: scanner.Info{Device: "test:0"}}}, nil
	})

	if code != 2 {
		t.Fatalf("RunScan() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: scanner "missing" not found`) {
		t.Fatalf("stderr = %q, want unknown-scanner error", stderr)
	}
}

func TestRunScanReportsScanFailureAndCleansUp(t *testing.T) {
	selected := &scanTestScanner{
		info:    scanner.Info{Device: "test:0"},
		scanErr: errors.New("device failure"),
	}
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{selected}, nil
	})

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
	selected := &scanTestScanner{
		info:    scanner.Info{Device: "test:0"},
		scanErr: context.Canceled,
	}
	code, _, stderr := runScanCommand([]string{
		"--device", "test:0",
		"--output", "output.pdf",
	}, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{selected}, nil
	})

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

func runScanCommand(args []string, discover discoverScanners) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runScan(context.Background(), args, &stdout, &stderr, discover)
	return code, stdout.String(), stderr.String()
}
