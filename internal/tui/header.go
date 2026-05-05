package tui

const appName = "ssh-chat"

// renderHeader fills the full width with the header background. The app name
// is the only content — lipgloss pads the remainder with the background color.
func (m model) renderHeader(width int) string {
	return m.styles.headerTitle.Width(width).MaxWidth(width).Render(appName)
}
