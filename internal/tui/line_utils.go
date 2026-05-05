package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// renderFullWidth renders a single-row component that owns a whole terminal row.
// rows: callers provide their desired content, and this function clips it to the
// available cells before Lip Gloss pads the row out to the exact frame width.
func renderFullWidth(style lipgloss.Style, width int, content string) string {
	width = safeDimension(width)
	return style.Width(width).Render(ansi.Truncate(content, width, ""))
}

// fixedCell returns content that occupies exactly width terminal cells.
// It is useful for columns: long text is clipped, short text is padded so the
// next column starts at a stable position.
func fixedCell(s string, width int) string {
	if lipgloss.Width(s) > width {
		return ansi.Truncate(s, width, "")
	}
	return fmt.Sprintf("%-*s", width, s)
}
