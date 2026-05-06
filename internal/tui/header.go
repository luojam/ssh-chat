package tui

import "strings"

const roomViewHeaderTitle = "Town Square"

// renderHeader fills the full width with the chat title. Lip Gloss pads the
// remainder with the header background color.
func (m roomViewModel) renderHeader(width int) string {
	return renderFullWidth(m.styles.headerTitle, width, roomViewHeaderTitle)
}

// renderHeaderDivider creates a visual split between the title bar and feed.
func (m roomViewModel) renderHeaderDivider(width int) string {
	line := strings.Repeat(string(inputSepRune), max(1, width))
	return renderFullWidth(m.styles.headerDivider, width, line)
}
