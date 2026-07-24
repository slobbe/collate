package collate

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	core "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestParseBackOrder(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		value string
		want  BackOrder
		err   bool
	}{
		{name: "reverse", value: "reverse", want: BackOrderReverse},
		{name: "forward", value: "forward", want: BackOrderForward},
		{name: "invalid", value: "sideways", err: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseBackOrder(test.value)
			if test.err {
				if err == nil {
					t.Fatal("ParseBackOrder succeeded, want an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("ParseBackOrder(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestCollateBackOrder(t *testing.T) {
	dir := t.TempDir()
	frontPath := createImagePDF(t, dir, "front.pdf", []image.Point{{X: 101, Y: 102}, {X: 103, Y: 104}})
	backPath := createImagePDF(t, dir, "back.pdf", []image.Point{{X: 201, Y: 202}, {X: 203, Y: 204}})

	for _, test := range []struct {
		name  string
		order BackOrder
		want  []image.Point
	}{
		{
			name:  "reverse",
			order: BackOrderReverse,
			want:  []image.Point{{X: 101, Y: 102}, {X: 203, Y: 204}, {X: 103, Y: 104}, {X: 201, Y: 202}},
		},
		{
			name:  "forward",
			order: BackOrderForward,
			want:  []image.Point{{X: 101, Y: 102}, {X: 201, Y: 202}, {X: 103, Y: 104}, {X: 203, Y: 204}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(dir, test.name+".pdf")
			if err := Collate(context.Background(), frontPath, backPath, outputPath, test.order); err != nil {
				t.Fatal(err)
			}

			if got := pageDimensions(t, outputPath); !equalPoints(got, test.want) {
				t.Fatalf("page dimensions = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCollateAllowsOneExtraFrontPage(t *testing.T) {
	dir := t.TempDir()
	frontPath := createImagePDF(t, dir, "front.pdf", []image.Point{{X: 101, Y: 102}, {X: 103, Y: 104}, {X: 105, Y: 106}})
	backPath := createImagePDF(t, dir, "back.pdf", []image.Point{{X: 201, Y: 202}, {X: 203, Y: 204}})

	for _, test := range []struct {
		name  string
		order BackOrder
		want  []image.Point
	}{
		{
			name:  "reverse",
			order: BackOrderReverse,
			want:  []image.Point{{X: 101, Y: 102}, {X: 203, Y: 204}, {X: 103, Y: 104}, {X: 201, Y: 202}, {X: 105, Y: 106}},
		},
		{
			name:  "forward",
			order: BackOrderForward,
			want:  []image.Point{{X: 101, Y: 102}, {X: 201, Y: 202}, {X: 103, Y: 104}, {X: 203, Y: 204}, {X: 105, Y: 106}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(dir, test.name+".pdf")
			if err := Collate(context.Background(), frontPath, backPath, outputPath, test.order); err != nil {
				t.Fatal(err)
			}

			if got := pageDimensions(t, outputPath); !equalPoints(got, test.want) {
				t.Fatalf("page dimensions = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCollateRejectsInvalidPageCounts(t *testing.T) {
	for _, test := range []struct {
		name       string
		frontPages []image.Point
		backPages  []image.Point
	}{
		{
			name:       "more back pages than front pages",
			frontPages: []image.Point{{X: 101, Y: 102}},
			backPages:  []image.Point{{X: 201, Y: 202}, {X: 203, Y: 204}},
		},
		{
			name:       "more than one extra front page",
			frontPages: []image.Point{{X: 101, Y: 102}, {X: 103, Y: 104}, {X: 105, Y: 106}},
			backPages:  []image.Point{{X: 201, Y: 202}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			frontPath := createImagePDF(t, dir, "front.pdf", test.frontPages)
			backPath := createImagePDF(t, dir, "back.pdf", test.backPages)

			err := Collate(context.Background(), frontPath, backPath, filepath.Join(dir, "output.pdf"), BackOrderReverse)
			if err == nil {
				t.Fatal("Collate succeeded, want a page-count mismatch error")
			}
		})
	}
}

func TestCollateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Collate(ctx, "front.pdf", "back.pdf", "output.pdf", BackOrderReverse)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Collate() error = %v, want context.Canceled", err)
	}
}

func createImagePDF(t *testing.T, dir, name string, dimensions []image.Point) string {
	t.Helper()

	images := make([]string, 0, len(dimensions))
	for i, dimension := range dimensions {
		path := filepath.Join(dir, name+strconv.Itoa(i)+".png")
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
