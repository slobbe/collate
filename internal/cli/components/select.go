package clicomponent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Select[T any] struct {
	Prompt  string
	Options []SelectOption[T]
	Default int
}

type SelectOption[T any] struct {
	Value T
	Label string
}

func (s Select[T]) Run(ctx context.Context, input *bufio.Reader, output io.Writer) (T, error) {
	var zero T
	if len(s.Options) == 0 {
		return zero, fmt.Errorf("%s has no options", s.Prompt)
	}
	if s.Default < 0 || s.Default >= len(s.Options) {
		return zero, fmt.Errorf("%s default %d is out of range", s.Prompt, s.Default)
	}

	rewrite := isTerminal(output)
	renderedLines := len(s.Options)
	for index, option := range s.Options {
		fmt.Fprintf(output, "[%d] %s\n", index+1, option.Label)
	}

	for {
		fmt.Fprintf(output, "%s [%d]: ", s.Prompt, s.Default+1)
		line, err := readLine(ctx, input)
		if err != nil {
			return zero, fmt.Errorf("read selection: %w", err)
		}

		if rewrite {
			renderedLines++
		}

		value := strings.TrimSpace(line)
		index := s.Default
		if value != "" {
			selected, parseErr := strconv.Atoi(value)
			if parseErr != nil || selected < 1 || selected > len(s.Options) {
				if !rewrite {
					fmt.Fprintln(output)
				}
				fmt.Fprintf(output, "Please enter a number from 1 to %d.\n", len(s.Options))
				if rewrite {
					renderedLines++
				}
				continue
			}
			index = selected - 1
		}

		if rewrite {
			fmt.Fprintf(output, "\x1b[%dA\x1b[J", renderedLines)
		} else {
			fmt.Fprintln(output)
		}
		fmt.Fprintf(output, "%s: %s\n", s.Prompt, s.Options[index].Label)
		return s.Options[index].Value, nil
	}
}
