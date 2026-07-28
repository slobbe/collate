package clicomponent

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/slobbe/collate/pkg/ansi"
)

type Confirm struct {
	Prompt string
	Done   string
}

func (c Confirm) Run(ctx context.Context, input *bufio.Reader, output io.Writer) error {
	rewrite := ansi.IsTerminal(output)
	if rewrite {
		if _, err := fmt.Fprint(output, ansi.SaveCursor); err != nil {
			return fmt.Errorf("save cursor: %w", err)
		}
	}

	fmt.Fprintf(output, "%s [press Enter]", c.Prompt)
	_, err := readLine(ctx, input)
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}

	if rewrite {
		if _, err := fmt.Fprint(output, ansi.RestoreCursor, ansi.ClearScreenFromCursor); err != nil {
			return fmt.Errorf("restore cursor: %w", err)
		}
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
