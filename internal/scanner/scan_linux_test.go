package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLinuxScannerScansSavesAndCleansUp(t *testing.T) {
	workDir := t.TempDir()
	var gotDevice string
	var gotPages []string

	scanner := &linuxScanner{
		info: Info{Device: "test:0"},
		scan: func(_ context.Context, device string, _ ScanOptions, outputDir string) error {
			gotDevice = device
			for _, name := range []string{"page-0002.png", "page-0001.png"} {
				if err := os.WriteFile(filepath.Join(outputDir, name), []byte(name), 0o600); err != nil {
					return err
				}
			}
			return nil
		},
		writePDF: func(pages []string, outputPath string, _ Paper) error {
			gotPages = append([]string(nil), pages...)
			return os.WriteFile(outputPath, []byte("pdf"), 0o600)
		},
		createTemp: func() (string, error) { return workDir, nil },
	}

	if err := scanner.Scan(context.Background(), ScanOptions{}); err != nil {
		t.Fatal(err)
	}
	if got, want := gotDevice, "test:0"; got != want {
		t.Fatalf("device = %q, want %q", got, want)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")
	if err := scanner.Save(context.Background(), outputPath); err != nil {
		t.Fatal(err)
	}
	if got, want := gotPages, []string{
		filepath.Join(workDir, "page-0001.png"),
		filepath.Join(workDir, "page-0002.png"),
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved pages = %v, want %v", got, want)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output PDF = %v, want it to exist", err)
	}

	if err := scanner.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scan workspace error = %v, want not exist", err)
	}
}

func TestScanimageArgsScanOnePage(t *testing.T) {
	got := scanimageArgs("test:0", ScanOptions{}, "/tmp/scans")
	want := []string{"--device-name", "test:0", "--format=png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scanimage arguments = %v, want %v", got, want)
	}
}

func TestScanimageArgsA4(t *testing.T) {
	got := scanimageArgs("test:0", ScanOptions{Paper: PaperA4}, "/tmp/scans")
	want := []string{
		"--device-name", "test:0",
		"--format=png",
		"-x", "210",
		"-y", "297",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scanimage arguments = %v, want %v", got, want)
	}
}

func TestScanimageArgsBatchFromSource(t *testing.T) {
	got := scanimageArgs("test:0", ScanOptions{Source: "ADF", Batch: true}, "/tmp/scans")
	want := []string{
		"--device-name", "test:0",
		"--format=png",
		"--source", "ADF",
		"--batch=/tmp/scans/page-%04d.png",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scanimage arguments = %v, want %v", got, want)
	}
}

func TestParsePaper(t *testing.T) {
	for _, test := range []struct {
		value string
		want  Paper
		err   bool
	}{
		{value: "a4", want: PaperA4},
		{value: "DIN-A4", want: PaperA4},
		{value: "letter", err: true},
	} {
		t.Run(test.value, func(t *testing.T) {
			got, err := ParsePaper(test.value)
			if test.err {
				if err == nil {
					t.Fatal("ParsePaper succeeded, want an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("ParsePaper(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestLinuxScannerSaveRequiresScan(t *testing.T) {
	scanner := newLinuxScanner(Info{Device: "test:0"})
	if err := scanner.Save(context.Background(), filepath.Join(t.TempDir(), "output.pdf")); err == nil {
		t.Fatal("Save succeeded without Scan")
	}
}

func TestLinuxScannerCleansUpAfterScanFailure(t *testing.T) {
	workDir := t.TempDir()
	scanner := &linuxScanner{
		info: Info{Device: "test:0"},
		scan: func(context.Context, string, ScanOptions, string) error {
			return errors.New("device failure")
		},
		createTemp: func() (string, error) { return workDir, nil },
	}

	if err := scanner.Scan(context.Background(), ScanOptions{}); err == nil {
		t.Fatal("Scan succeeded, want error")
	}
	if _, err := os.Stat(workDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scan workspace error = %v, want not exist", err)
	}
}

func TestLinuxScannerDoesNotScanCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	scanner := &linuxScanner{
		info: Info{Device: "test:0"},
		scan: func(context.Context, string, ScanOptions, string) error {
			called = true
			return nil
		},
	}

	err := scanner.Scan(ctx, ScanOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Scan error = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("scan command was called for a cancelled context")
	}
}
