package collate

import (
	"context"
	"image"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type sessionTestResult struct {
	path   string
	closed bool
}

func (r *sessionTestResult) SavePDF(_ context.Context, path string) error {
	source, err := os.Open(r.path)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return err
	}
	return destination.Close()
}

func (r *sessionTestResult) Close() error {
	r.closed = true
	return nil
}

type sessionTestScanner struct {
	results []ScanResult
	next    int
}

func (s *sessionTestScanner) Info() ScannerInfo { return ScannerInfo{ID: "scanner"} }

func (s *sessionTestScanner) Capabilities(context.Context) (Capabilities, error) {
	return Capabilities{}, nil
}

func (s *sessionTestScanner) Scan(context.Context, ScanOptions) (ScanResult, error) {
	result := s.results[s.next]
	s.next++
	return result, nil
}

func TestScanSessionSavesChunksInDocumentOrder(t *testing.T) {
	dir := t.TempDir()
	results := []*sessionTestResult{
		{path: createImagePDF(t, dir, "front-one.pdf", []image.Point{{X: 101, Y: 102}, {X: 103, Y: 104}})},
		{path: createImagePDF(t, dir, "back-one.pdf", []image.Point{{X: 201, Y: 202}, {X: 203, Y: 204}})},
		{path: createImagePDF(t, dir, "front-two.pdf", []image.Point{{X: 301, Y: 302}, {X: 303, Y: 304}})},
		{path: createImagePDF(t, dir, "back-two.pdf", []image.Point{{X: 401, Y: 402}, {X: 403, Y: 404}})},
	}
	scanner := &sessionTestScanner{results: []ScanResult{results[0], results[1], results[2], results[3]}}
	ctx := context.Background()

	session, err := StartScan(ctx, scanner, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.ScanBackInOrder(ctx, BackOrderReverse); err != nil {
		t.Fatal(err)
	}
	if err := session.ScanNextChunk(ctx); err != nil {
		t.Fatal(err)
	}
	if err := session.ScanBackInOrder(ctx, BackOrderForward); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "output.pdf")
	if err := session.Save(ctx, outputPath); err != nil {
		t.Fatal(err)
	}
	want := []image.Point{
		{X: 101, Y: 102}, {X: 203, Y: 204}, {X: 103, Y: 104}, {X: 201, Y: 202},
		{X: 301, Y: 302}, {X: 401, Y: 402}, {X: 303, Y: 304}, {X: 403, Y: 404},
	}
	if got := pageDimensions(t, outputPath); !equalPoints(got, want) {
		t.Fatalf("page dimensions = %v, want %v", got, want)
	}
}
