package tui

import "strings"

const defaultRoomViewTitle = "Room"

// renderHeader fills the full width with the chat title. Lip Gloss pads the
// remainder with the header background color.
func (m roomViewModel) renderHeader(width int) string {
	title := m.title
	if m.joinCode != "" {
		title += " • Join code: " + m.joinCode
	}
	return renderFullWidth(m.styles.headerTitle, width, title)
}

// renderHeaderDivider creates a visual split between the title bar and feed.
func (m roomViewModel) renderHeaderDivider(width int) string {
	line := strings.Repeat(string(inputSepRune), max(1, width))
	return renderFullWidth(m.styles.headerDivider, width, line)
}
