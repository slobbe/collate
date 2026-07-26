package escl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanResultCloseIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.pdf")
	if err := os.WriteFile(path, []byte("%PDF-test"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := &scanResult{path: path}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if result.path != "" {
		t.Fatalf("result path = %q, want empty after close", result.path)
	}
}
