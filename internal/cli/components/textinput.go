package clicomponent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type TextInput struct {
	Prompt    string
	Default   string
	Normalize func(string) (string, error)
}

func (t TextInput) Run(ctx context.Context, input *bufio.Reader, output io.Writer) (string, error) {
	rewrite := isTerminal(output)
	renderedLines := 0

	for {
		fmt.Fprintf(output, "%s [%s]: ", t.Prompt, t.Default)
		line, err := readLine(ctx, input)
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		if rewrite {
			renderedLines++
		}

		value := strings.TrimSpace(line)
		if value == "" {
			value = t.Default
		}
		if t.Normalize != nil {
			value, err = t.Normalize(value)
			if err != nil {
				if !rewrite {
					fmt.Fprintln(output)
				}
				fmt.Fprintf(output, "Invalid value: %v\n", err)
				if rewrite {
					renderedLines++
				}
				continue
			}
		}

		if rewrite {
			fmt.Fprintf(output, "\x1b[%dA\x1b[J", renderedLines)
		} else {
			fmt.Fprintln(output)
		}
		fmt.Fprintf(output, "%s: %s\n", t.Prompt, value)
		return value, nil
	}
}
