package clicomponent

import (
	"io"
	"os"
)

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
