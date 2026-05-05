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
	keyLeaveChat = "ctrl+l"
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
	case keyLeaveChat:
		return true, requestLeave
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
	m.syncMessageViewport(false)
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height

	m.syncComposer()
	m.syncMessageViewport(false)
}

func (m *model) syncComposer() {
	width := inputWidth(m.frameWidth())
	m.input.SetWidth(width)
	m.input.Placeholder = buildPlaceholder(width)
}

func (m model) render() string {
	layout := m.layout()
	sections := make([]string, 0, 4)

	if layout.showHeader {
		sections = append(sections,
			m.renderHeader(layout.width),
			m.renderHeaderDivider(layout.width),
		)
	}
	if layout.messageRows > 0 {
		sections = append(sections, m.renderMessages())
	}
	sections = append(sections, m.renderComposerSection(layout.width, layout.showInputFrame))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
