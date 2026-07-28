package ansi

import "fmt"

const (
	Reset           = "\x1b[0m"
	ResetForeground = "\x1b[39m"
	ResetBackground = "\x1b[49m"

	Bold           = "\x1b[1m"
	Dim            = "\x1b[2m"
	Italic         = "\x1b[3m"
	Underline      = "\x1b[4m"
	Invert         = "\x1b[7m"
	Strikethrough  = "\x1b[9m"
)

type BasicColor uint8

const (
	Black BasicColor = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// foreground
func ForegroundColor(color BasicColor) string {
	return fmt.Sprintf("\x1b[3%dm", color)
}

func BrightForegroundColor(color BasicColor) string {
	return fmt.Sprintf("\x1b[9%dm", color)
}

func ForegroundRGB(r, g, b uint8) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func Foreground256(color uint8) string {
	return fmt.Sprintf("\x1b[38;5;%dm", color)
}

// background
func BackgroundColor(color BasicColor) string {
	return fmt.Sprintf("\x1b[4%dm", color)
}

func BrightBackgroundColor(color BasicColor) string {
	return fmt.Sprintf("\x1b[10%dm", color)
}

func BackgroundRGB(r, g, b uint8) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

func Background256(color uint8) string {
	return fmt.Sprintf("\x1b[48;5;%dm", color)
}
