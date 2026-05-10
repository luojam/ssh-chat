package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	sshKeySettingsTitle              = "SSH Key Settings"
	sshKeySettingsCurrentLabel       = "Detected SSH key:"
	sshKeySettingsLinkedLabel        = "Currently linked SSH key:"
	sshKeySettingsLinkButton         = "Link current key"
	sshKeySettingsDeleteButton       = "Delete linked key"
	sshKeySettingsHintLine           = "←/→ • enter • esc back • ctrl+c quit"
	sshKeySettingsMissingFingerprint = ""
	sshKeySettingsBoxWidth           = 64
	sshKeySettingsBoxHeight          = 18
)

type sshKeySettingsModel struct {
	screen     screenState
	styles     mainMenuStyles
	currentKey string
	linkedKey  string
	selected   int
}

func NewSSHKeySettings(config Config) tea.Model {
	m := sshKeySettingsModel{
		screen:     newScreenState(config),
		styles:     newMainMenuStyles(true),
		currentKey: strings.TrimSpace(config.SSHKeyFingerprint),
		linkedKey:  strings.TrimSpace(config.LinkedSSHKeyFingerprint),
	}
	m.resize(config.Width, config.Height)
	return m
}

func (m sshKeySettingsModel) Init() tea.Cmd { return fullScreenInit() }

func (m sshKeySettingsModel) View() tea.View { return fullScreenView(m.render()) }

func (m sshKeySettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *sshKeySettingsModel) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case keyQuit:
		return requestQuit
	case keyBack:
		return requestBack
	case "left", "right", "tab":
		m.selected = (m.selected + 1) % 2
	case keySend:
		if m.selected == 0 {
			return requestLinkCurrentSSHKey()
		}
		return requestDeleteLinkedSSHKey()
	}
	return nil
}

func (m *sshKeySettingsModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newMainMenuStyles(isDark)
}

func (m *sshKeySettingsModel) resize(width, height int) { m.screen.resize(width, height) }

func (m sshKeySettingsModel) render() string {
	frame := m.screen.frame()
	box := sshKeySettingsBoxSize(frame, m.styles.box)
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

func (m sshKeySettingsModel) renderBox(box, content frameSize) string {
	body := lipgloss.JoinVertical(
		lipgloss.Center,
		m.styles.heading.Render(wrapCenter(sshKeySettingsTitle, content.width)),
		"",
		m.renderField(content.width, sshKeySettingsCurrentLabel, m.currentKey),
		"",
		m.renderField(content.width, sshKeySettingsLinkedLabel, m.linkedKey),
		"",
		m.renderActions(content.width),
		"",
		m.styles.hint.Render(wrapCenter(sshKeySettingsHintLine, content.width)),
	)

	body = lipgloss.NewStyle().
		Width(content.width).
		Height(content.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(fitBlockHeight(body, content.height))

	return m.styles.box.Width(box.width).Height(box.height).Render(body)
}

func (m sshKeySettingsModel) renderField(width int, label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = sshKeySettingsMissingFingerprint
	}
	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.styles.hint.Render(wrapCenter(label, width)),
		wrapCenter(value, width),
	)
}

func (m sshKeySettingsModel) renderActions(width int) string {
	link := m.styles.button.Render(sshKeySettingsLinkButton)
	deleteKey := m.styles.button.Render(sshKeySettingsDeleteButton)
	if m.selected == 0 {
		link = m.styles.selectedButton.Render(sshKeySettingsLinkButton)
	} else {
		deleteKey = m.styles.selectedButton.Render(sshKeySettingsDeleteButton)
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, link, "  ", deleteKey),
	)
}

func sshKeySettingsBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	maxWidth := frame.width
	if frame.width > mainMenuFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= mainMenuFramePaddingX * 2
	}
	maxHeight := frame.height
	if frame.height > mainMenuFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= mainMenuFramePaddingY * 2
	}
	return frameSize{width: max(1, min(sshKeySettingsBoxWidth, maxWidth)), height: max(1, min(sshKeySettingsBoxHeight, maxHeight))}
}
