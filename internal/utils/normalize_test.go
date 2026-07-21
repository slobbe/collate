package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("home directory is unavailable: %v", err)
	}

	absolute := filepath.Join(t.TempDir(), "nested", "..", "absolute.pdf")
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "bare filename", path: "front.pdf", want: filepath.Join(cwd, "front.pdf")},
		{name: "current directory", path: "./path/to.pdf", want: filepath.Join(cwd, "path", "to.pdf")},
		{name: "relative path is cleaned", path: "path/../back.pdf", want: filepath.Join(cwd, "back.pdf")},
		{name: "absolute path is cleaned", path: absolute, want: filepath.Clean(absolute)},
		{name: "home directory", path: "~", want: filepath.Clean(home)},
		{name: "home directory child", path: "~/scans/../front.pdf", want: filepath.Join(home, "front.pdf")},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizePath(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("normalizePath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func TestNormalizePDFPathRequiresPDFExtension(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		path string
		err  bool
	}{
		{path: "document.pdf"},
		{path: "document.PDF"},
		{path: "document", err: true},
		{path: "document.png", err: true},
	} {
		t.Run(test.path, func(t *testing.T) {
			_, err := NormalizePDFPath(test.path)
			if test.err && err == nil {
				t.Fatal("NormalizePDFPath succeeded, want an error")
			}
			if !test.err && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNormalizePathRejectsUnsupportedInput(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"", "~another-user/scans.pdf"} {
		if _, err := normalizePath(path); err == nil {
			t.Errorf("normalizePath(%q) succeeded, want an error", path)
		}
	}
}
