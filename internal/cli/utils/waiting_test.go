package cliutils

import (
	"bytes"
	"context"
	"testing"
)

func TestWaitingWritesProgressAndCompletion(t *testing.T) {
	var output bytes.Buffer
	err := (Waiting{Message: "Scanning front pages"}).Run(
		context.Background(),
		&output,
		func(context.Context) (string, error) { return "Scanned front pages", nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Scanning front pages...\nScanned front pages\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
