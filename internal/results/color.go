package results

import (
	"fmt"
	"os"
	"runtime"
)

// ANSI color codes.
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	green   = "\033[32m"
	cyan    = "\033[36m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	boldRed = "\033[1;31m"
	white   = "\033[37m"
	grey    = "\033[90m"
)

var colorEnabled = detectColor()

func detectColor() bool {
	// Check if output is a terminal.
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return false // piped
	}

	// Check environment hints.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// Windows 10+ supports VT100 sequences.
	if runtime.GOOS == "windows" {
		return enableWindowsVT100()
	}

	return true
}

// SetColor overrides automatic color detection.
func SetColor(enabled bool) {
	colorEnabled = enabled
}

// ColorGrade returns the grade string with appropriate color.
func ColorGrade(g Grade) string {
	if !colorEnabled {
		return string(g)
	}
	var c string
	switch g {
	case GradeA:
		c = green
	case GradeB:
		c = cyan
	case GradeC:
		c = yellow
	case GradeD:
		c = red
	case GradeF:
		c = boldRed
	}
	return c + bold + string(g) + reset
}

// Colorize wraps text in the given color.
func Colorize(color, text string) string {
	if !colorEnabled {
		return text
	}
	return color + text + reset
}

// Header formats a section header.
func Header(text string) string {
	if !colorEnabled {
		return fmt.Sprintf("=== %s ===", text)
	}
	return bold + white + "=== " + text + bold + " ===" + reset
}

// Dim formats text in grey.
func Dim(text string) string {
	return Colorize(grey, text)
}

// Bold formats text in bold.
func Bold(text string) string {
	return Colorize(bold, text)
}

// Green formats text in green.
func Green(text string) string {
	return Colorize(green, text)
}

// Red formats text in red.
func Red(text string) string {
	return Colorize(red, text)
}
