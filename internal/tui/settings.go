package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	settingsTitle           = "Settings"
	settingsUsernameLabel   = "Signed in as"
	settingsHintLine        = "↑/↓ move • enter select • esc back • ctrl+c quit"
	settingsConfirmHintLine = "←/→ switch • enter choose • esc cancel • ctrl+c quit"
	settingsTargetWidth     = 56
	settingsMaxVisibleRows  = 4
	settingsFramePaddingX   = 2
	settingsFramePaddingY   = 1
	settingsContainerPadX   = 2
	settingsContainerPadY   = 1
	settingsTitleHeight     = 2
	settingsAccountHeight   = 1
	settingsListTopPadding  = 1
	settingsHintHeight      = 1
)

type settingsMode int

const (
	settingsModeMenu settingsMode = iota
	settingsModeDeleteConfirm
)

type settingsOptionAction int

const (
	settingsOptionSSHKeys settingsOptionAction = iota
	settingsOptionDeleteAccount
)

type settingsOptionItem struct {
	action      settingsOptionAction
	title       string
	description string
}

func (i settingsOptionItem) Title() string       { return i.title }
func (i settingsOptionItem) Description() string { return i.description }
func (i settingsOptionItem) FilterValue() string { return i.title }

func NewSettings(config Config) tea.Model {
	styles := newMyChatsStyles(true)
	m := settingsModel{
		screen:   newScreenState(config),
		styles:   styles,
		username: config.Username,
		list:     newSettingsList(styles),
	}
	m.resize(config.Width, config.Height)
	return m
}

type settingsModel struct {
	screen       screenState
	styles       myChatsStyles
	username     string
	list         list.Model
	mode         settingsMode
	confirmIndex int
}

func (m settingsModel) Init() tea.Cmd {
	return fullScreenInit()
}

func (m settingsModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		if handled, cmd := m.handleKeyPress(msg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *settingsModel) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.mode == settingsModeDeleteConfirm {
		return m.handleDeleteConfirmKey(msg)
	}

	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		return true, requestBack
	case keySend:
		return true, m.selectCurrent()
	default:
		return false, nil
	}
}

func (m *settingsModel) handleDeleteConfirmKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		m.mode = settingsModeMenu
		return true, nil
	case "left", "right", "tab":
		m.confirmIndex = (m.confirmIndex + 1) % 2
		return true, nil
	case keySend:
		if m.confirmIndex == 1 {
			m.mode = settingsModeMenu
			return true, nil
		}
		return true, requestDeleteAccount
	default:
		return false, nil
	}
}

func (m *settingsModel) selectCurrent() tea.Cmd {
	item, ok := m.list.SelectedItem().(settingsOptionItem)
	if !ok {
		return nil
	}
	if item.action != settingsOptionDeleteAccount {
		return nil
	}
	m.mode = settingsModeDeleteConfirm
	m.confirmIndex = 1 // Default to Cancel for destructive confirmations.
	return nil
}

func (m *settingsModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newMyChatsStyles(isDark)
	m.list.SetDelegate(settingsDelegate(m.styles))
	m.list.Styles = m.styles.listStyles(m.list.Width())
	m.list.SetSize(m.list.Width(), m.list.Height())
}

func (m *settingsModel) resize(width, height int) {
	m.screen.resize(width, height)
	layout := settingsLayoutFor(width, height, m.styles, len(m.list.Items()))
	m.list.Styles = m.styles.listStyles(layout.listContent.width)
	m.list.SetSize(layout.listContent.width, layout.listContent.height)
}

func (m settingsModel) render() string {
	layout := settingsLayoutFor(m.screen.width, m.screen.height, m.styles, len(m.list.Items()))

	sections := []string{
		m.renderTitle(layout.content.width),
		m.renderAccount(layout.content.width),
		"",
	}
	if m.mode == settingsModeDeleteConfirm {
		sections = append(sections,
			m.renderDeleteConfirm(layout.content.width),
			"",
			m.styles.hint.Width(layout.content.width).Align(lipgloss.Center).Render(settingsConfirmHintLine),
		)
	} else {
		sections = append(sections,
			m.list.View(),
			m.styles.hint.Width(layout.content.width).Align(lipgloss.Center).Render(settingsHintLine),
		)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	content = lipgloss.NewStyle().
		Width(layout.content.width).
		Height(layout.content.height).
		Render(fitBlockHeight(content, layout.content.height))

	container := lipgloss.NewStyle().
		Width(layout.container.width).
		Height(layout.container.height).
		Render(content)

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(container)
}

func (m settingsModel) renderTitle(width int) string {
	style := m.styles.listStyles(width).Title
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(style.Render(settingsTitle))
}

func (m settingsModel) renderAccount(width int) string {
	username := m.username
	if username == "" {
		username = "unknown"
	}
	line := m.styles.hint.Render(settingsUsernameLabel+" ") + username
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(line)
}

func (m settingsModel) renderDeleteConfirm(width int) string {
	actions := newAuthStyles(m.screen.isDark)
	prompt := m.styles.hint.Render(wrapCenter("Delete your account permanently?", width))
	deleteButton := actions.tab.Render("Delete")
	cancelButton := actions.tab.Render("Cancel")
	if m.confirmIndex == 0 {
		deleteButton = actions.activeTab.Render("Delete")
	} else {
		cancelButton = actions.activeTab.Render("Cancel")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, deleteButton, "  ", cancelButton)
	buttons = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(buttons)
	return lipgloss.JoinVertical(lipgloss.Center, prompt, "", buttons)
}

func newSettingsList(styles myChatsStyles) list.Model {
	l := list.New(settingsOptions(), settingsDelegate(styles), 1, 1)
	l.Title = settingsTitle
	l.Styles = styles.listStyles(1)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	return l
}

func settingsDelegate(styles myChatsStyles) list.DefaultDelegate {
	delegate := styles.listDelegate()
	delegate.ShowDescription = true
	return delegate
}

func settingsOptions() []list.Item {
	return []list.Item{
		settingsOptionItem{action: settingsOptionSSHKeys, title: "SSH key settings", description: "Add/remove SSH key"},
		settingsOptionItem{action: settingsOptionDeleteAccount, title: "Delete account", description: "Delete account permanently"},
	}
}

type settingsLayout struct {
	frame       frameSize
	container   frameSize
	content     frameSize
	listContent frameSize
}

func settingsLayoutFor(width, height int, styles myChatsStyles, itemCount int) settingsLayout {
	frame := safeFrameSize(width, height)
	container := settingsContainerSize(frame, styles, itemCount)
	content := frameSize{
		width:  max(1, container.width-settingsContainerPadX*2),
		height: max(1, container.height-settingsContainerPadY*2),
	}
	listContent := frameSize{
		width:  content.width,
		height: max(1, content.height-settingsTitleHeight-settingsAccountHeight-settingsListTopPadding-settingsHintHeight),
	}
	return settingsLayout{frame: frame, container: container, content: content, listContent: listContent}
}

func settingsContainerSize(frame frameSize, styles myChatsStyles, itemCount int) frameSize {
	maxWidth := frame.width
	if frame.width > settingsFramePaddingX*2 {
		maxWidth -= settingsFramePaddingX * 2
	}
	maxHeight := frame.height
	if frame.height > settingsFramePaddingY*2 {
		maxHeight -= settingsFramePaddingY * 2
	}

	visibleItems := min(max(1, itemCount), settingsMaxVisibleRows)
	delegate := settingsDelegate(styles)
	listItemsHeight := visibleItems*delegate.Height() + max(0, visibleItems-1)*(delegate.Spacing()+1)
	listHeight := settingsTitleHeight + settingsAccountHeight + listItemsHeight
	desiredHeight := settingsAccountHeight + settingsListTopPadding + listHeight + settingsHintHeight + settingsContainerPadY*2
	return frameSize{
		width:  max(1, min(settingsTargetWidth, maxWidth)),
		height: max(1, min(desiredHeight, maxHeight)),
	}
}
