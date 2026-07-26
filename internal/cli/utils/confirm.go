package cliutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

type Confirm struct {
	Prompt string
	Done   string
}

func (c Confirm) Run(ctx context.Context, input *bufio.Reader, output io.Writer) error {
	rewrite := isTerminal(output)
	if rewrite {
		fmt.Fprint(output, "\x1b[s")
	}

	fmt.Fprintf(output, "%s [press Enter]", c.Prompt)
	_, err := readLine(ctx, input)
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}

	if rewrite {
		fmt.Fprint(output, "\x1b[u\x1b[J")
	} else {
		fmt.Fprintln(output)
	}
	done := c.Done
	if done == "" {
		done = "Confirmed"
	}
	fmt.Fprintf(output, "%s: %s\n", c.Prompt, done)
	return nil
}
