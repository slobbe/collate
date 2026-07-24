package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/slobbe/collate/internal/scanner"
)

type testScanner struct {
	info scanner.Info
}

func (scanner testScanner) Info() scanner.Info {
	return scanner.info
}

func (testScanner) Scan(context.Context, scanner.ScanOptions) error {
	return nil
}

func (testScanner) Save(context.Context, string) error {
	return nil
}

func (testScanner) Close() error {
	return nil
}

func TestRunScannerListListsAvailableScanners(t *testing.T) {
	code, stdout, stderr := runScannerListCommand(nil, func(context.Context) ([]scanner.Scanner, error) {
		return []scanner.Scanner{
			testScanner{info: scanner.Info{Device: "airscan:e0:OfficeJet", Description: "HP OfficeJet Pro 9010"}},
			testScanner{info: scanner.Info{Device: "test:0", Description: "Virtual scanner"}},
		}, nil
	})

	if code != 0 {
		t.Fatalf("RunScanner() exit code = %d, want 0", code)
	}
	const want = "Available scanners:\n- HP OfficeJet Pro 9010\n  device: airscan:e0:OfficeJet\n- Virtual scanner\n  device: test:0\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScannerListReportsNoScanners(t *testing.T) {
	code, stdout, stderr := runScannerListCommand(nil, func(context.Context) ([]scanner.Scanner, error) {
		return nil, nil
	})

	if code != 0 {
		t.Fatalf("RunScanner() exit code = %d, want 0", code)
	}
	if stdout != "No scanners found.\n" {
		t.Fatalf("stdout = %q, want no-scanners message", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScannerListReportsDiscoveryFailure(t *testing.T) {
	code, _, stderr := runScannerListCommand(nil, func(context.Context) ([]scanner.Scanner, error) {
		return nil, errors.New("scanimage is not installed")
	})

	if code != 1 {
		t.Fatalf("RunScanner() exit code = %d, want 1", code)
	}
	if stderr != "error: discover scanners: scanimage is not installed\n" {
		t.Fatalf("stderr = %q, want discovery error", stderr)
	}
}

func TestRunScannerListRejectsPositionalArguments(t *testing.T) {
	code, _, stderr := runScannerListCommand([]string{"extra"}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 2 {
		t.Fatalf("RunScanner() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "error: unexpected positional arguments: [extra]") {
		t.Fatalf("stderr = %q, want positional-argument error", stderr)
	}
	if !strings.Contains(stderr, "usage: collate scanner") {
		t.Fatalf("stderr = %q, want usage", stderr)
	}
}

func TestRunScannerListHelp(t *testing.T) {
	code, _, stderr := runScannerListCommand([]string{"-h"}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 0 {
		t.Fatalf("RunScannerList() exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr, "usage: collate scanner list") {
		t.Fatalf("stderr = %q, want usage", stderr)
	}
}

func TestRunScannerRequiresCommand(t *testing.T) {
	code, _, stderr := runScannerCommand(nil, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 2 {
		t.Fatalf("RunScanner() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "usage: collate scanner <command>") {
		t.Fatalf("stderr = %q, want scanner usage", stderr)
	}
}

func TestRunScannerHelp(t *testing.T) {
	code, stdout, stderr := runScannerCommand([]string{"-h"}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 0 {
		t.Fatalf("RunScanner() exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "usage: collate scanner <command>") {
		t.Fatalf("stdout = %q, want usage", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunScannerRejectsUnknownCommand(t *testing.T) {
	code, _, stderr := runScannerCommand([]string{"inspect"}, func(context.Context) ([]scanner.Scanner, error) {
		t.Fatal("discover was called")
		return nil, nil
	})

	if code != 2 {
		t.Fatalf("RunScanner() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: unknown scanner command "inspect"`) {
		t.Fatalf("stderr = %q, want unknown-command error", stderr)
	}
}

func TestRunScannerReportsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer
	code := runScanner(ctx, nil, &stdout, &stderr, func(context.Context) ([]scanner.Scanner, error) {
		return nil, context.Canceled
	})
	if code != 130 {
		t.Fatalf("RunScanner() exit code = %d, want 130", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "interrupted\n" {
		t.Fatalf("stderr = %q, want interruption message", stderr.String())
	}
}

func runScannerCommand(args []string, discover discoverScanners) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runScanner(context.Background(), args, &stdout, &stderr, discover)
	return code, stdout.String(), stderr.String()
}

func runScannerListCommand(args []string, discover discoverScanners) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runScannerList(context.Background(), args, &stdout, &stderr, discover)
	return code, stdout.String(), stderr.String()
}
