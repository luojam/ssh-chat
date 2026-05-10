package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	MinimumTerminalWidth  = 70
	MinimumTerminalHeight = 20
)

// WithMinimumSize wraps a full-screen model with a resize prompt when the
// terminal is too small.
func WithMinimumSize(inner tea.Model, width, height int) tea.Model {
	return minimumSizeModel{inner: inner, width: width, height: height}
}

type minimumSizeModel struct {
	inner  tea.Model
	width  int
	height int
}

func (m minimumSizeModel) Init() tea.Cmd {
	return m.inner.Init()
}

func (m minimumSizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		return m.updateInner(msg)
	}

	if m.tooSmall() {
		if msg, ok := msg.(tea.KeyPressMsg); ok && msg.String() == keyQuit {
			return m.updateInner(msg)
		}
		return m, nil
	}

	return m.updateInner(msg)
}

func (m minimumSizeModel) View() tea.View {
	if m.tooSmall() {
		return fullScreenView(renderMinimumSizePrompt(m.width, m.height))
	}
	return m.inner.View()
}

func (m minimumSizeModel) updateInner(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	return m, cmd
}

func (m minimumSizeModel) tooSmall() bool {
	return m.width < MinimumTerminalWidth || m.height < MinimumTerminalHeight
}

func renderMinimumSizePrompt(width, height int) string {
	frame := safeFrameSize(width, height)
	message := fmt.Sprintf(
		"Window too small\n\nResize to at least %d×%d\nCurrent size: %d×%d",
		MinimumTerminalWidth,
		MinimumTerminalHeight,
		width,
		height,
	)
	return lipgloss.NewStyle().
		Width(frame.width).
		Height(frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(message)
}
