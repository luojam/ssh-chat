package tui

import "strings"

const appName = "ssh-chat"

// renderHeader fills the full width with the header background. The app name
// is the only content — lipgloss pads the remainder with the background color.
func (m model) renderHeader(width int) string {
	return renderFullWidth(m.styles.headerTitle, width, appName)
}

// renderHeaderDivider creates a visual split between the title bar and feed.
func (m model) renderHeaderDivider(width int) string {
	line := strings.Repeat(string(inputSepRune), max(1, width))
	return renderFullWidth(m.styles.headerDivider, width, line)
}
