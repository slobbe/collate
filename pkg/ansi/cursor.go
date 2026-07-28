package ansi

import "fmt"

const (
	SaveCursor    = "\x1b[s"
	RestoreCursor = "\x1b[u"

	ShowCursor = "\x1b[?25h"
	HideCursor = "\x1b[?25l"

	SaveCursorDEC    = "\x1b7"
	RestoreCursorDEC = "\x1b8"

	RequestCursorPosition = "\x1b[6n"

	ClearScreen          = "\x1b[2J"
	ClearScreenFromCursor = "\x1b[0J"
	ClearLine            = "\x1b[2K"
	ClearLineFromCursor  = "\x1b[0K"
)

func CursorUp(n int) string {
	return fmt.Sprintf("\x1b[%dA", n)
}

func CursorDown(n int) string {
	return fmt.Sprintf("\x1b[%dB", n)
}

func CursorRight(n int) string {
	return fmt.Sprintf("\x1b[%dC", n)
}

func CursorLeft(n int) string {
	return fmt.Sprintf("\x1b[%dD", n)
}

func CursorNextLine(n int) string {
	return fmt.Sprintf("\x1b[%dE", n)
}

func CursorPreviousLine(n int) string {
	return fmt.Sprintf("\x1b[%dF", n)
}

func CursorColumn(column int) string {
	return fmt.Sprintf("\x1b[%dG", column)
}

func CursorRow(row int) string {
	return fmt.Sprintf("\x1b[%dd", row)
}

func CursorPosition(row, column int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, column)
}
