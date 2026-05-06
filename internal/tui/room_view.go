package tui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	keyQuit = "ctrl+c"
	keyBack = "esc"
	keySend = "enter"
)

type Config struct {
	Width  int
	Height int
	Rooms  []RoomListItem
}

type RoomListItem struct {
	ID    string
	Title string
}

func NewRoomView(config Config) tea.Model {
	input, focusCmd := newComposer(true)

	m := roomViewModel{
		screen:     newScreenState(config),
		styles:     newBaseStyles(true),
		input:      input,
		viewport:   newMessageViewport(),
		initialCmd: focusCmd,
	}
	m.resize(config.Width, config.Height)
	return m
}

type roomViewModel struct {
	screen screenState

	styles   baseStyles
	input    textinput.Model
	viewport viewport.Model
	messages []message

	initialCmd tea.Cmd
}

func (m roomViewModel) Init() tea.Cmd {
	return fullScreenInit(m.initialCmd)
}

func (m roomViewModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m roomViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *roomViewModel) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		return true, requestLeave
	case keySend:
		return true, m.requestSend()
	default:
		return false, nil
	}
}

func (m *roomViewModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}

	m.styles = newBaseStyles(isDark)
	m.input.SetStyles(inputStyles(isDark))
	m.syncMessageViewport(false)
}

func (m *roomViewModel) resize(width, height int) {
	m.screen.resize(width, height)

	m.syncComposer()
	m.syncMessageViewport(false)
}

func (m *roomViewModel) syncComposer() {
	width := inputWidth(m.frameWidth())
	m.input.SetWidth(width)
	m.input.Placeholder = buildPlaceholder(width)
}

func (m roomViewModel) render() string {
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
