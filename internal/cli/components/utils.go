package clicomponent

import (
	"bufio"
	"context"
	"io"
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
