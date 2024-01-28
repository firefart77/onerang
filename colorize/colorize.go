package colorize

import "fmt"

const escape = "\x1b"

const (
	NONE = iota
	RED
	GREEN
	YELLOW
	BLUE
	PURPLE
)

func color(c int) string {
	if c == NONE {
		return fmt.Sprintf("%s[%dm", escape, c)
	}
	return fmt.Sprintf("%s[3%dm", escape, c)
}

func Colorize(c int, text string) string {
	return color(c) + text + color(NONE)
}
