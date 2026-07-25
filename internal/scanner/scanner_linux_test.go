//go:build linux

package scanner

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/slobbe/collate/internal/pdf"
)

func TestLinuxScannerSaveWritesScannedPageAsPDF(t *testing.T) {
	workingDir := t.TempDir()
	pagePath := filepath.Join(workingDir, "page-0001.png")
	page, err := os.Create(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(page, image.NewGray(image.Rect(0, 0, 100, 200))); err != nil {
		page.Close()
		t.Fatal(err)
	}
	if err := page.Close(); err != nil {
		t.Fatal(err)
	}

	scanner := &LinuxScanner{workingDir: workingDir}
	outputPath := filepath.Join(t.TempDir(), "output.pdf")
	if err := scanner.Save(context.Background(), outputPath); err != nil {
		t.Fatal(err)
	}

	document, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := document.PageCount(), 1; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
}

func TestLinuxScannerSaveWritesAllScannedPagesAsPDF(t *testing.T) {
	workingDir := t.TempDir()
	for _, name := range []string{"page-0001.png", "page-0002.png"} {
		page, err := os.Create(filepath.Join(workingDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(page, image.NewGray(image.Rect(0, 0, 100, 200))); err != nil {
			page.Close()
			t.Fatal(err)
		}
		if err := page.Close(); err != nil {
			t.Fatal(err)
		}
	}

	scanner := &LinuxScanner{workingDir: workingDir}
	outputPath := filepath.Join(t.TempDir(), "output.pdf")
	if err := scanner.Save(context.Background(), outputPath); err != nil {
		t.Fatal(err)
	}

	document, err := pdf.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := document.PageCount(), 2; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}
}

func TestIsFeederSource(t *testing.T) {
	for _, test := range []struct {
		source Source
		want   bool
	}{
		{source: "Flatbed", want: false},
		{source: "ADF", want: true},
		{source: "ADF Duplex", want: true},
		{source: "Document Feeder", want: true},
	} {
		if got := isFeederSource(test.source); got != test.want {
			t.Fatalf("isFeederSource(%q) = %t, want %t", test.source, got, test.want)
		}
	}
}

func TestLinuxScannerSaveRequiresScan(t *testing.T) {
	scanner := &LinuxScanner{}
	if err := scanner.Save(context.Background(), filepath.Join(t.TempDir(), "output.pdf")); err == nil {
		t.Fatal("Save succeeded without a scanned page")
	}
}
