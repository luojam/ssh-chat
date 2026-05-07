package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	linkSSHKeyQuestionLine = "Link this SSH key for passwordless auth next time?"
	linkSSHKeyFingerprint  = "Fingerprint:"
	linkSSHKeyYesLabel     = "Yes"
	linkSSHKeyNoLabel      = "No"
	linkSSHKeyHintLine     = "←/→ switch • enter choose • esc no • ctrl+c quit"
	linkSSHKeyBoxWidth     = 58
	linkSSHKeyBoxHeight    = 13
)

type linkSSHKeyModel struct {
	screen      screenState
	styles      mainMenuStyles
	fingerprint string
	selectedYes bool
}

func NewLinkSSHKey(config Config) tea.Model {
	m := linkSSHKeyModel{
		screen:      newScreenState(config),
		styles:      newMainMenuStyles(true),
		fingerprint: strings.TrimSpace(config.SSHKeyFingerprint),
		selectedYes: true,
	}
	m.resize(config.Width, config.Height)
	return m
}

func (m linkSSHKeyModel) Init() tea.Cmd {
	return fullScreenInit()
}

func (m linkSSHKeyModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m linkSSHKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *linkSSHKeyModel) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case keyQuit:
		return requestQuit
	case keyBack:
		return requestSSHKeyLinkSelection(false)
	case "left", "right", "tab":
		m.selectedYes = !m.selectedYes
	case keySend:
		return requestSSHKeyLinkSelection(m.selectedYes)
	}
	return nil
}

func (m *linkSSHKeyModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newMainMenuStyles(isDark)
}

func (m *linkSSHKeyModel) resize(width, height int) {
	m.screen.resize(width, height)
}

func (m linkSSHKeyModel) render() string {
	frame := m.screen.frame()
	box := linkSSHKeyBoxSize(frame, m.styles.box)
	content := frameSize{
		width:  max(1, box.width-m.styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-m.styles.box.GetVerticalFrameSize()),
	}

	return lipgloss.NewStyle().
		Width(frame.width).
		Height(frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderBox(box, content))
}

func (m linkSSHKeyModel) renderBox(box, content frameSize) string {
	body := lipgloss.JoinVertical(
		lipgloss.Center,
		m.styles.heading.Render(wrapCenter(linkSSHKeyQuestionLine, content.width)),
		"",
		m.styles.hint.Render(wrapCenter(linkSSHKeyFingerprint, content.width)),
		wrapCenter(m.fingerprint, content.width),
		"",
		m.renderActions(content.width),
		"",
		m.styles.hint.Render(wrapCenter(linkSSHKeyHintLine, content.width)),
	)

	body = lipgloss.NewStyle().
		Width(content.width).
		Height(content.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(fitBlockHeight(body, content.height))

	return m.styles.box.
		Width(box.width).
		Height(box.height).
		Render(body)
}

func (m linkSSHKeyModel) renderActions(width int) string {
	yes := m.styles.button.Render(linkSSHKeyYesLabel)
	no := m.styles.button.Render(linkSSHKeyNoLabel)
	if m.selectedYes {
		yes = m.styles.selectedButton.Render(linkSSHKeyYesLabel)
	} else {
		no = m.styles.selectedButton.Render(linkSSHKeyNoLabel)
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, yes, "  ", no),
	)
}

func linkSSHKeyBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	maxWidth := frame.width
	if frame.width > mainMenuFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= mainMenuFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > mainMenuFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= mainMenuFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, min(linkSSHKeyBoxWidth, maxWidth)),
		height: max(1, min(linkSSHKeyBoxHeight, maxHeight)),
	}
}
