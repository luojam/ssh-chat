package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	dashboardHeadingLine      = "MENU"
	dashboardMyChatsTitle     = "My Chats"
	dashboardManageChatsTitle = "+/-"
	dashboardSettingsTitle    = "Settings"
	dashboardHintLine         = "←/→ move • enter select • esc exit"
	dashboardTargetBoxWidth   = 56
	dashboardTargetBoxHeight  = 12
	dashboardFramePaddingX    = 2
	dashboardFramePaddingY    = 1
	dashboardButtonGap        = 3
)

var dashboardHeaderLines = []string{
	"█▀▄▀█ █▀▀ █▄░█ █░█",
	"█░▀░█ ██▄ █░▀█ █▄█",
}

type dashboardSection int

const (
	dashboardSectionMyChats dashboardSection = iota
	dashboardSectionManageChats
	dashboardSectionSettings
)

// dashboardItem is presentation data only. Application actions for a selected
// button belong in the session layer; this view only tracks selection.
type dashboardItem struct {
	section dashboardSection
	action  DashboardAction
	title   string
}

type dashboardLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
}

func NewDashboard(config Config) tea.Model {
	m := dashboardModel{
		screen:        newScreenState(config),
		styles:        newDashboardStyles(true),
		selectedIndex: 0,
	}
	m.resize(config.Width, config.Height)
	return m
}

type dashboardModel struct {
	screen screenState

	styles        dashboardStyles
	selectedIndex int
}

func (m dashboardModel) Init() tea.Cmd {
	return screenInit()
}

func (m dashboardModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		return m, m.handleKeyPress(msg)
	}

	return m, nil
}

func (m *dashboardModel) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case keyQuitCtrlC, keyQuitEsc:
		return requestQuit
	case "left", "h":
		m.selectPrevious()
	case "right", "l", "tab":
		m.selectNext()
	case keySend:
		return requestDashboardSelection(dashboardItems[m.selectedIndex].action)
	}
	return nil
}

func (m *dashboardModel) selectPrevious() {
	m.selectedIndex = (m.selectedIndex - 1 + len(dashboardItems)) % len(dashboardItems)
}

func (m *dashboardModel) selectNext() {
	m.selectedIndex = (m.selectedIndex + 1) % len(dashboardItems)
}

func (m *dashboardModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}

	m.styles = newDashboardStyles(isDark)
	m.resize(m.screen.width, m.screen.height)
}

func (m *dashboardModel) resize(width, height int) {
	m.screen.resize(width, height)
}

func (m dashboardModel) render() string {
	layout := dashboardLayoutFor(m.screen.width, m.screen.height, m.styles)

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderDashboardBox(layout))
}

func (m dashboardModel) renderDashboardBox(layout dashboardLayout) string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.renderDashboardHeader(layout.content.width),
		"",
		m.renderButtonRow(layout.content.width),
		"",
		m.renderDashboardHint(layout.content.width),
	)

	content = lipgloss.NewStyle().
		Width(layout.content.width).
		Height(layout.content.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(fitBlockHeight(content, layout.content.height))

	return m.styles.box.
		Width(layout.box.width).
		Height(layout.box.height).
		Render(content)
}

func (m dashboardModel) renderDashboardHeader(width int) string {
	lines := make([]string, len(dashboardHeaderLines))
	for i, line := range dashboardHeaderLines {
		lines[i] = m.styles.heading.Render(wrapCenter(ansi.Truncate(line, width, ""), width))
	}
	return strings.Join(lines, "\n")
}

func (m dashboardModel) renderDashboardHint(width int) string {
	return m.styles.hint.Render(wrapCenter(dashboardHintLine, width))
}

func (m dashboardModel) renderButtonRow(width int) string {
	buttons := make([]string, 0, len(dashboardItems))
	for i, item := range dashboardItems {
		style := m.styles.button
		if i == m.selectedIndex {
			style = m.styles.selectedButton
		}
		buttons = append(buttons, style.Render(item.title))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Center, buttons...)
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(row)
}

var dashboardItems = []dashboardItem{
	{section: dashboardSectionMyChats, action: DashboardActionMyChats, title: dashboardMyChatsTitle},
	{section: dashboardSectionManageChats, action: DashboardActionManageChats, title: dashboardManageChatsTitle},
	{section: dashboardSectionSettings, action: DashboardActionSettings, title: dashboardSettingsTitle},
}

// dashboardLayoutFor keeps sizing logic centralized so render output and tests
// agree on the centered container dimensions.
func dashboardLayoutFor(width, height int, styles dashboardStyles) dashboardLayout {
	frame := safeFrameSize(width, height)
	box := dashboardBoxSize(frame, styles.box)

	content := frameSize{
		width:  max(1, box.width-styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-styles.box.GetVerticalFrameSize()),
	}

	return dashboardLayout{
		frame:   frame,
		box:     box,
		content: content,
	}
}

func dashboardBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	maxWidth := frame.width
	if frame.width > dashboardFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= dashboardFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > dashboardFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= dashboardFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, min(dashboardTargetBoxWidth, maxWidth)),
		height: max(1, min(dashboardTargetBoxHeight, maxHeight)),
	}
}

func fitBlockHeight(content string, height int) string {
	height = safeDimension(height)
	lines := strings.SplitN(content, "\n", height+1)
	if len(lines) <= height {
		return content
	}
	return strings.Join(lines[:height], "\n")
}
