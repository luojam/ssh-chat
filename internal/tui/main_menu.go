package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	mainMenuHeadingLine      = "MENU"
	mainMenuMyChatsTitle     = "My Chats"
	mainMenuManageChatsTitle = "+/-"
	mainMenuSettingsTitle    = "Settings"
	mainMenuHintLine         = "←/→ • enter • esc back • ctrl+c quit"
	mainMenuTargetBoxWidth   = 56
	mainMenuTargetBoxHeight  = 12
	mainMenuFramePaddingX    = 2
	mainMenuFramePaddingY    = 1
	mainMenuButtonGap        = 3
)

var mainMenuHeaderLines = []string{
	"█▀▄▀█ █▀▀ █▄░█ █░█",
	"█░▀░█ ██▄ █░▀█ █▄█",
}

type mainMenuSection int

const (
	mainMenuSectionMyChats mainMenuSection = iota
	mainMenuSectionManageChats
	mainMenuSectionSettings
)

// mainMenuItem is presentation data only. Application actions for a selected
// button belong in the session layer; this view only tracks selection.
type mainMenuItem struct {
	section mainMenuSection
	action  MainMenuAction
	title   string
}

type mainMenuLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
}

func NewMainMenu(config Config) tea.Model {
	m := mainMenuModel{
		screen:        newScreenState(config),
		styles:        newMainMenuStyles(true),
		selectedIndex: 0,
	}
	m.resize(config.Width, config.Height)
	return m
}

type mainMenuModel struct {
	screen screenState

	styles        mainMenuStyles
	selectedIndex int
}

func (m mainMenuModel) Init() tea.Cmd {
	return fullScreenInit()
}

func (m mainMenuModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *mainMenuModel) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case keyQuit:
		return requestQuit
	case keyBack:
		return requestBack
	case "left", "h":
		m.selectPrevious()
	case "right", "l", "tab":
		m.selectNext()
	case keySend:
		return requestMainMenuSelection(mainMenuItems[m.selectedIndex].action)
	}
	return nil
}

func (m *mainMenuModel) selectPrevious() {
	m.selectedIndex = (m.selectedIndex - 1 + len(mainMenuItems)) % len(mainMenuItems)
}

func (m *mainMenuModel) selectNext() {
	m.selectedIndex = (m.selectedIndex + 1) % len(mainMenuItems)
}

func (m *mainMenuModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}

	m.styles = newMainMenuStyles(isDark)
	m.resize(m.screen.width, m.screen.height)
}

func (m *mainMenuModel) resize(width, height int) {
	m.screen.resize(width, height)
}

func (m mainMenuModel) render() string {
	layout := mainMenuLayoutFor(m.screen.width, m.screen.height, m.styles)

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderMainMenuBox(layout))
}

func (m mainMenuModel) renderMainMenuBox(layout mainMenuLayout) string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.renderMainMenuHeader(layout.content.width),
		"",
		m.renderButtonRow(layout.content.width),
		"",
		m.renderMainMenuHint(layout.content.width),
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

func (m mainMenuModel) renderMainMenuHeader(width int) string {
	lines := make([]string, len(mainMenuHeaderLines))
	for i, line := range mainMenuHeaderLines {
		lines[i] = m.styles.heading.Render(wrapCenter(ansi.Truncate(line, width, ""), width))
	}
	return strings.Join(lines, "\n")
}

func (m mainMenuModel) renderMainMenuHint(width int) string {
	return m.styles.hint.Render(wrapCenter(mainMenuHintLine, width))
}

func (m mainMenuModel) renderButtonRow(width int) string {
	buttons := make([]string, 0, len(mainMenuItems))
	for i, item := range mainMenuItems {
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

var mainMenuItems = []mainMenuItem{
	{section: mainMenuSectionMyChats, action: MainMenuActionMyChats, title: mainMenuMyChatsTitle},
	{section: mainMenuSectionManageChats, action: MainMenuActionManageChats, title: mainMenuManageChatsTitle},
	{section: mainMenuSectionSettings, action: MainMenuActionSettings, title: mainMenuSettingsTitle},
}

// mainMenuLayoutFor keeps sizing logic centralized so render output and tests
// agree on the centered container dimensions.
func mainMenuLayoutFor(width, height int, styles mainMenuStyles) mainMenuLayout {
	frame := safeFrameSize(width, height)
	box := mainMenuBoxSize(frame, styles.box)

	content := frameSize{
		width:  max(1, box.width-styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-styles.box.GetVerticalFrameSize()),
	}

	return mainMenuLayout{
		frame:   frame,
		box:     box,
		content: content,
	}
}

func mainMenuBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	maxWidth := frame.width
	if frame.width > mainMenuFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= mainMenuFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > mainMenuFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= mainMenuFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, min(mainMenuTargetBoxWidth, maxWidth)),
		height: max(1, min(mainMenuTargetBoxHeight, maxHeight)),
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
