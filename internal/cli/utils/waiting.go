package cliutils

import (
	"context"
	"fmt"
	"io"
	"time"
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

	if !isTerminal(output) {
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
		fmt.Fprintf(output, "\r\x1b[2K%s %s", spinnerFrames[frame], w.Message)
		frame = (frame + 1) % len(spinnerFrames)
	}
	render()

	for {
		select {
		case result := <-result:
			fmt.Fprint(output, "\r\x1b[2K")
			if result.err == nil {
				fmt.Fprintln(output, result.message)
			}
			return result.err
		case <-ticker.C:
			render()
		case <-ctx.Done():
			fmt.Fprint(output, "\r\x1b[2K")
			return ctx.Err()
		}
	}
}
