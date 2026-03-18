package client

import (
	"fmt"
	"os"
	"strings"
)

// Progress displays real-time test progress in the terminal.
type Progress struct {
	enabled bool
}

// NewProgress creates a progress display. Disabled for non-TTY or JSON mode.
func NewProgress(jsonMode bool) *Progress {
	if jsonMode {
		return &Progress{enabled: false}
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return &Progress{enabled: false}
	}
	return &Progress{enabled: fi.Mode()&os.ModeCharDevice != 0}
}

// Status prints a status message that overwrites the current line.
func (p *Progress) Status(format string, args ...any) {
	if !p.enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	// Pad to overwrite previous content.
	if len(msg) < 60 {
		msg += strings.Repeat(" ", 60-len(msg))
	}
	fmt.Fprintf(os.Stderr, "\r  %s", msg)
}

// Clear clears the status line.
func (p *Progress) Clear() {
	if !p.enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 70))
}

// TestStart announces a test is beginning.
func (p *Progress) TestStart(name string) {
	if !p.enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "  Running %s...\n", name)
}

// FormatBPS formats bits per second for display.
func FormatBPS(bps float64) string {
	switch {
	case bps >= 1e9:
		return fmt.Sprintf("%.2f Gbps", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f Mbps", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f Kbps", bps/1e3)
	default:
		return fmt.Sprintf("%.0f bps", bps)
	}
}
