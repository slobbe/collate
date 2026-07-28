package clicomponent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/slobbe/collate/pkg/ansi"
)

const spinnerInterval = 100 * time.Millisecond

var spinnerFrames = [...]string{"|", "/", "-", "\\"}

type Waiting struct {
	Message string
}

func (w Waiting) Run(ctx context.Context, output io.Writer, action func(context.Context) (string, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if action == nil {
		return fmt.Errorf("waiting action is required")
	}

	type actionResult struct {
		message string
		err     error
	}
	result := make(chan actionResult, 1)
	go func() {
		message, err := action(ctx)
		result <- actionResult{message: message, err: err}
	}()

	if !ansi.IsTerminal(output) {
		fmt.Fprintf(output, "%s...\n", w.Message)
		select {
		case result := <-result:
			if result.err == nil {
				fmt.Fprintln(output, result.message)
			}
			return result.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()
	frame := 0
	render := func() {
		fmt.Fprintf(output, "\r%s%s %s", ansi.ClearLine, spinnerFrames[frame], w.Message)
		frame = (frame + 1) % len(spinnerFrames)
	}
	render()

	for {
		select {
		case result := <-result:
			_, cleanupErr := fmt.Fprint(output, "\r", ansi.ClearLine)
			if result.err == nil {
				fmt.Fprintln(output, result.message)
			}
			return errors.Join(result.err, cleanupErr)
		case <-ticker.C:
			render()
		case <-ctx.Done():
			_, cleanupErr := fmt.Fprint(output, "\r", ansi.ClearLine)
			return errors.Join(ctx.Err(), cleanupErr)
		}
	}
}
