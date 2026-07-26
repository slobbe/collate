package clicomponent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestSelectStopsWhenContextIsCancelled(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := (Select[string]{
			Prompt:  "Select value",
			Options: []SelectOption[string]{{Value: "one", Label: "One"}},
			Default: 0,
		}).Run(ctx, bufio.NewReader(reader), &bytes.Buffer{})
		result <- err
	}()

	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Select did not stop after cancellation")
	}
}
