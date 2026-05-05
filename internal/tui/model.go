package tui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	keyQuitCtrlC = "ctrl+c"
	keyQuitEsc   = "esc"
	keySend      = "enter"
)

type Config struct {
	Width  int
	Height int
}

func New(config Config) tea.Model {
	input, focusCmd := newComposer(true)

	m := model{
		width:      config.Width,
		height:     config.Height,
		isDark:     true,
		styles:     newStyles(true),
		input:      input,
		viewport:   newMessageViewport(),
		initialCmd: focusCmd,
	}
	m.resize(config.Width, config.Height)
	return m
}

type model struct {
	width  int
	height int
	isDark bool

	styles   styles
	input    textinput.Model
	viewport viewport.Model
	messages []message

	initialCmd tea.Cmd
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.initialCmd)
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case MessageReceived:
		m.receiveMessage(msg)
	case tea.KeyPressMsg:
		if handled, cmd := m.handleKeyPress(msg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuitCtrlC, keyQuitEsc:
		return true, requestQuit
	case keySend:
		return true, m.requestSend()
	default:
		return false, nil
	}
}

func (m *model) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}

	m.isDark = isDark
	m.styles = newStyles(isDark)
	m.input.SetStyles(inputStyles(isDark))
	m.syncMessageViewport()
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height

	m.input.SetWidth(inputWidth(m.frameWidth()))
	m.syncMessageViewport()
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
			m.renderHeaderDivider(frame.width),
			m.renderMessages(),
			m.renderComposerSection(frame.width, frame.height),
		)
	}
}
