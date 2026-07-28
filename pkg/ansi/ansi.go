package ansi

import (
	"io"

	"golang.org/x/term"
)

type fdWriter interface {
	Fd() uintptr
}

func IsTerminal(output io.Writer) bool {
	writer, ok := output.(fdWriter)
	return ok && term.IsTerminal(int(writer.Fd()))
}
