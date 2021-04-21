package utils

import (
	"github.com/mgutz/ansi"
)

// Variables for colors. Expand the color pallete here
var (
	RED    = ansi.ColorFunc("red")
	YELLOW = ansi.ColorFunc("yellow")
	GREEN  = ansi.ColorFunc("green")
	GRAY   = ansi.ColorFunc("black+h")
)

// Red wrapper for a string
func Red(format string) string {
	return RED(format)
}

// Yellow wrapper for a string
func Yellow(format string) string {
	return YELLOW(format)
}

// Green wrapper for a string
func Green(format string) string {
	return GREEN(format)
}

// Gray wrapper for a string
func Gray(format string) string {
	return GRAY(format)
}
