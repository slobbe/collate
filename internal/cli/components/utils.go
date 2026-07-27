package clicomponent

import (
	"bufio"
	"context"
	"io"
	"os"
)

type lineResult struct {
	line string
	err  error
}

func readLine(ctx context.Context, input *bufio.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	result := make(chan lineResult, 1)
	go func() {
		line, err := input.ReadString('\n')
		result <- lineResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-result:
		if result.err != nil && result.err != io.EOF {
			return "", result.err
		}
		if result.err == io.EOF && result.line == "" {
			return "", io.EOF
		}
		return result.line, nil
	}
}

// isTerminal returns whether the output writer is a terminal.
// used to determine whether to enable terminal features such as cursor movement and color output.
func isTerminal(output io.Writer) bool {
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
