package clicomponent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type YesNo struct {
	Prompt  string
	Default bool
}

func (y YesNo) Run(ctx context.Context, input *bufio.Reader, output io.Writer) (bool, error) {
	rewrite := isTerminal(output)
	if rewrite {
		fmt.Fprint(output, "\x1b[s")
	}

	choices := "y/N"
	if y.Default {
		choices = "Y/n"
	}

	for {
		fmt.Fprintf(output, "%s [%s]: ", y.Prompt, choices)
		line, err := readLine(ctx, input)
		if err != nil {
			return false, fmt.Errorf("read response: %w", err)
		}

		answer := strings.ToLower(strings.TrimSpace(line))
		selected := y.Default
		switch answer {
		case "":
		case "y", "yes":
			selected = true
		case "n", "no":
			selected = false
		default:
			if !rewrite {
				fmt.Fprintln(output)
			}
			fmt.Fprintln(output, "Please answer yes or no.")
			continue
		}

		if rewrite {
			fmt.Fprint(output, "\x1b[u\x1b[J")
		} else {
			fmt.Fprintln(output)
		}
		label := "No"
		if selected {
			label = "Yes"
		}
		fmt.Fprintf(output, "%s: %s\n", y.Prompt, label)
		return selected, nil
	}
}
