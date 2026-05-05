package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	welcomeTitleLine      = "Welcome!"
	welcomeContinueLine   = "Press enter to continue..."
	welcomeQuitLine       = "Esc to exit"
	welcomeTargetBoxWidth = 40
)

func NewWelcome(config Config) tea.Model {
	m := welcomeModel{
		width:      config.Width,
		height:     config.Height,
		isDark:     true,
		styles:     newStyles(true),
		initialCmd: nil,
	}
	m.resize(config.Width, config.Height)
	return m
}

type welcomeModel struct {
	width  int
	height int
	isDark bool

	styles styles

	initialCmd tea.Cmd
}

func (m welcomeModel) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.initialCmd)
}

func (m welcomeModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m welcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case keyQuitCtrlC, keyQuitEsc:
			return m, requestQuit
		case keySend:
			return m, requestContinue
		}
	}

	return m, nil
}

func (m *welcomeModel) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}
	m.isDark = isDark
	m.styles = newStyles(isDark)
}

func (m *welcomeModel) resize(width, height int) {
	m.width = width
	m.height = height
}

func (m welcomeModel) render() string {
	frame := safeFrameSize(m.width, m.height)
	boxW := min(frame.width, welcomeTargetBoxWidth)

	box := m.renderWelcomeBox(boxW)
	return lipgloss.NewStyle().
		Width(frame.width).
		Height(frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

func (m welcomeModel) renderWelcomeBox(width int) string {
	contentW := max(1, width-m.styles.welcomeBox.GetHorizontalFrameSize())
	lines := []string{
		m.styles.welcomeTitle.Render(ansi.Truncate(welcomeTitleLine, contentW, "")),
		"",
		m.styles.welcomePrimary.Render(wrapCenter(welcomeContinueLine, contentW)),
		m.styles.welcomeSecondary.Render(wrapCenter(welcomeQuitLine, contentW)),
	}
	body := strings.Join(lines, "\n")

	return m.styles.welcomeBox.
		Width(width).
		Render(body)
}

func wrapCenter(text string, width int) string {
	wrapped := strings.Split(ansi.Wrap(text, width, " "), "\n")
	for i := range wrapped {
		wrapped[i] = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(wrapped[i])
	}
	return strings.Join(wrapped, "\n")
}
