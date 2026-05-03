package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m model) render() string {
	frame := m.frame()

	switch frame.height {
	case 1:
		return m.renderComposer(frame.width)
	case 2:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderMessages(),
			m.renderComposer(frame.width),
		)
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(frame.width),
			m.renderMessages(),
			m.renderComposer(frame.width),
		)
	}
}
