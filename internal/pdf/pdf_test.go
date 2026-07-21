package pdf

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	core "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestOpenAndNewCopyPagesInRequestedOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	frontPath := createImagePDF(t, dir, "front.pdf", []image.Point{{X: 101, Y: 102}, {X: 103, Y: 104}})
	backPath := createImagePDF(t, dir, "back.pdf", []image.Point{{X: 201, Y: 202}, {X: 203, Y: 204}})

	front, err := Open(frontPath)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Open(backPath)
	if err != nil {
		t.Fatal(err)
	}
	output, err := New()
	if err != nil {
		t.Fatal(err)
	}

	for _, source := range []struct {
		document Document
		index    int
	}{
		{front, 0},
		{back, 1},
		{front, 1},
		{back, 0},
	} {
		page, err := source.document.PageAt(source.index)
		if err != nil {
			t.Fatal(err)
		}
		if err := output.AppendPage(page); err != nil {
			t.Fatal(err)
		}
	}

	outputPath := filepath.Join(dir, "output.pdf")
	if err := output.Save(outputPath); err != nil {
		t.Fatal(err)
	}

	result, err := Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.PageCount(), 4; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}

	if got, want := pageDimensions(t, outputPath), []image.Point{{X: 101, Y: 102}, {X: 203, Y: 204}, {X: 103, Y: 104}, {X: 201, Y: 202}}; !equalPoints(got, want) {
		t.Fatalf("page dimensions = %v, want %v", got, want)
	}
}

func TestDocumentPageAtRejectsOutOfRangeIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := createImagePDF(t, dir, "document.pdf", []image.Point{{X: 101, Y: 102}})

	document, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, index := range []int{-1, 1} {
		if _, err := document.PageAt(index); err == nil {
			t.Errorf("PageAt(%d) succeeded, want an out-of-range error", index)
		}
	}
}

func TestDocumentAppendPageRejectsForeignPage(t *testing.T) {
	t.Parallel()

	output, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := output.AppendPage(foreignPage{}); err == nil {
		t.Fatal("AppendPage accepted a page from another implementation")
	}
}

type foreignPage struct{}

func (foreignPage) PDFPage() {}

func createImagePDF(t *testing.T, dir, name string, dimensions []image.Point) string {
	t.Helper()

	images := make([]string, 0, len(dimensions))
	for i, dimension := range dimensions {
		path := filepath.Join(dir, name+string(rune('0'+i))+".png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}

		if err := png.Encode(file, image.NewGray(image.Rectangle{Max: dimension})); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}

		images = append(images, path)
	}

	path := filepath.Join(dir, name)
	for i, imagePath := range images {
		imp := core.DefaultImportConfig()
		imp.PageDim.Width = float64(dimensions[i].X)
		imp.PageDim.Height = float64(dimensions[i].Y)
		imp.UserDim = true

		if err := api.ImportImagesFile([]string{imagePath}, path, imp, nil); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func pageDimensions(t *testing.T, path string) []image.Point {
	t.Helper()

	pages, err := api.PageDimsFile(path)
	if err != nil {
		t.Fatal(err)
	}

	dimensions := make([]image.Point, len(pages))
	for i, page := range pages {
		dimensions[i] = image.Point{X: int(page.Width), Y: int(page.Height)}
	}
	return dimensions
}

func equalPoints(a, b []image.Point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
