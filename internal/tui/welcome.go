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

	welcomeLogoFullWidth       = 60
	welcomeLogoStackedWidth    = 33
	welcomeLogoExtraHorizontal = 8
)

var welcomeLogoFull = []string{
	"███████╗███████╗██╗  ██╗    ██████╗██╗  ██╗ █████╗ ████████╗",
	"██╔════╝██╔════╝██║  ██║   ██╔════╝██║  ██║██╔══██╗╚══██╔══╝",
	"███████╗███████╗███████║   ██║     ███████║███████║   ██║   ",
	"╚════██║╚════██║██╔══██║   ██║     ██╔══██║██╔══██║   ██║   ",
	"███████║███████║██║  ██║   ╚██████╗██║  ██║██║  ██║   ██║   ",
	"╚══════╝╚══════╝╚═╝  ╚═╝    ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ",
}

var welcomeLogoStacked = []string{
	"███████╗███████╗██╗  ██╗",
	"██╔════╝██╔════╝██║  ██║",
	"███████╗███████╗███████║",
	"╚════██║╚════██║██╔══██║",
	"███████║███████║██║  ██║",
	"╚══════╝╚══════╝╚═╝  ╚═╝",
	"",
	" ██████╗██╗  ██╗ █████╗ ████████╗",
	"██╔════╝██║  ██║██╔══██╗╚══██╔══╝",
	"██║     ███████║███████║   ██║   ",
	"██║     ██╔══██║██╔══██║   ██║   ",
	"╚██████╗██║  ██║██║  ██║   ██║   ",
	" ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ",
}

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
	boxW := m.welcomeBoxWidth(frame.width, frame.height)

	box := m.renderWelcomeBox(boxW, frame.height)
	return lipgloss.NewStyle().
		Width(frame.width).
		Height(frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

func (m welcomeModel) welcomeBoxWidth(frameWidth, frameHeight int) int {
	boxFrameW := m.styles.welcomeBox.GetHorizontalFrameSize()
	boxFrameH := m.styles.welcomeBox.GetVerticalFrameSize()
	if canRenderWelcomeLogo(frameWidth-boxFrameW, frameHeight, boxFrameH, welcomeLogoFull) {
		return min(frameWidth, welcomeLogoFullWidth+boxFrameW+welcomeLogoExtraHorizontal)
	}
	if canRenderWelcomeLogo(frameWidth-boxFrameW, frameHeight, boxFrameH, welcomeLogoStacked) {
		return min(frameWidth, welcomeLogoStackedWidth+boxFrameW+welcomeLogoExtraHorizontal)
	}
	return min(frameWidth, welcomeTargetBoxWidth)
}

func (m welcomeModel) renderWelcomeBox(width, frameHeight int) string {
	contentW := max(1, width-m.styles.welcomeBox.GetHorizontalFrameSize())
	lines := m.welcomeLogoLines(contentW, frameHeight)
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	if len(lines) == 0 {
		lines = append(lines,
			m.styles.welcomeTitle.Render(ansi.Truncate(welcomeTitleLine, contentW, "")),
			"",
		)
	}
	lines = append(lines,
		m.styles.welcomePrimary.Render(wrapCenter(welcomeContinueLine, contentW)),
		m.styles.welcomeSecondary.Render(wrapCenter(welcomeQuitLine, contentW)),
	)
	body := strings.Join(lines, "\n")

	return m.styles.welcomeBox.
		Width(width).
		Render(body)
}

func (m welcomeModel) welcomeLogoLines(contentWidth, frameHeight int) []string {
	switch {
	case canRenderWelcomeLogo(contentWidth, frameHeight, m.styles.welcomeBox.GetVerticalFrameSize(), welcomeLogoFull):
		return m.centerWelcomeLogo(welcomeLogoFull, contentWidth)
	case canRenderWelcomeLogo(contentWidth, frameHeight, m.styles.welcomeBox.GetVerticalFrameSize(), welcomeLogoStacked):
		return m.centerWelcomeLogo(welcomeLogoStacked, contentWidth)
	default:
		return nil
	}
}

func canRenderWelcomeLogo(contentWidth, frameHeight, boxFrameHeight int, logo []string) bool {
	const baseBodyRows = 3 // spacer after logo, continue hint, quit hint
	return contentWidth >= maxLineWidth(logo) && frameHeight >= baseBodyRows+len(logo)+boxFrameHeight
}

func (m welcomeModel) centerWelcomeLogo(logo []string, width int) []string {
	lines := make([]string, len(logo))
	for i, line := range logo {
		lines[i] = m.styles.welcomeLogo.Width(width).Align(lipgloss.Center).Render(line)
	}
	return lines
}

func maxLineWidth(lines []string) int {
	width := 0
	for _, line := range lines {
		width = max(width, lipgloss.Width(line))
	}
	return width
}

func wrapCenter(text string, width int) string {
	wrapped := strings.Split(ansi.Wrap(text, width, " "), "\n")
	for i := range wrapped {
		wrapped[i] = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(wrapped[i])
	}
	return strings.Join(wrapped, "\n")
}
