package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	dashboardTitleLine = "Dashboard"
	dashboardJoinLine  = "Press enter to join chat"
)

func NewDashboard(config Config) tea.Model {
	m := dashboardModel{
		width:  config.Width,
		height: config.Height,
		isDark: true,
		styles: newStyles(true),
	}
	m.resize(config.Width, config.Height)
	return m
}

type dashboardModel struct {
	width  int
	height int
	isDark bool

	styles styles
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m dashboardModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *dashboardModel) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}
	m.isDark = isDark
	m.styles = newStyles(isDark)
}

func (m *dashboardModel) resize(width, height int) {
	m.width = width
	m.height = height
}

func (m dashboardModel) render() string {
	frame := safeFrameSize(m.width, m.height)
	boxW := min(frame.width, welcomeTargetBoxWidth)

	box := m.renderDashboardBox(boxW)
	return lipgloss.NewStyle().
		Width(frame.width).
		Height(frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

func (m dashboardModel) renderDashboardBox(width int) string {
	contentW := max(1, width-m.styles.welcomeBox.GetHorizontalFrameSize())
	lines := []string{
		m.styles.welcomeTitle.Render(ansi.Truncate(dashboardTitleLine, contentW, "")),
		"",
		m.styles.welcomePrimary.Render(wrapCenter(dashboardJoinLine, contentW)),
	}
	body := strings.Join(lines, "\n")

	return m.styles.welcomeBox.
		Width(width).
		Render(body)
}
