package tui

import (
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	manageRoomsHeadingLine           = "MANAGE ROOMS"
	manageRoomsCreateTitle           = "Create Room"
	manageRoomsJoinTitle             = "Join Room"
	manageRoomsDeleteTitle           = "Delete Room"
	manageRoomsRoomTitleLabel        = "Room title"
	manageRoomsJoinCodeLabel         = "Join code"
	manageRoomsDeleteEmptyState      = "You don't own any rooms."
	manageRoomsMenuHintLine          = "←/→ • enter • esc back • ctrl+c quit"
	manageRoomsCreateHintLine        = "enter create • esc back • ctrl+c quit"
	manageRoomsJoinHintLine          = "enter join • esc back • ctrl+c quit"
	manageRoomsDeleteHintLine        = "↑/↓ • enter select • esc back • ctrl+c quit"
	manageRoomsConfirmHintLine       = "←/→ • enter • esc cancel • ctrl+c quit"
	manageRoomsTargetBoxWidth        = 60
	manageRoomsTargetBoxHeight       = 14
	manageRoomsListTargetWidth       = 56
	manageRoomsDeleteMaxVisibleRooms = 8
	manageRoomsDeleteTitleHeight     = 2
	manageRoomsDeleteFooterHeight    = 2
	manageRoomsFramePaddingX         = 2
	manageRoomsFramePaddingY         = 1
	manageRoomsButtonGap             = 3
)

type manageRoomsMode int

const (
	manageRoomsModeMenu manageRoomsMode = iota
	manageRoomsModeCreate
	manageRoomsModeJoin
	manageRoomsModeDelete
	manageRoomsModeDeleteConfirm
)

type manageRoomsSection int

const (
	manageRoomsSectionCreate manageRoomsSection = iota
	manageRoomsSectionJoin
	manageRoomsSectionDelete
)

type manageRoomsItem struct {
	section manageRoomsSection
	title   string
}

type manageRoomsLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
}

type manageRoomsListLayout struct {
	frame   frameSize
	content frameSize
	list    frameSize
}

func NewManageRooms(config Config) tea.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "room name"
	input.SetStyles(inputStyles(true))

	listStyles := newMyChatsStyles(true)
	m := manageRoomsModel{
		screen:        newScreenState(config),
		styles:        newAuthStyles(true),
		listStyles:    listStyles,
		mode:          manageRoomsModeMenu,
		selectedIndex: 0,
		input:         input,
		deleteList:    newManageRoomsDeleteList(listStyles, config.OwnedRooms),
	}
	m.resize(config.Width, config.Height)
	return m
}

type manageRoomsModel struct {
	screen screenState

	styles                authStyles
	listStyles            myChatsStyles
	mode                  manageRoomsMode
	selectedIndex         int
	input                 textinput.Model
	deleteList            list.Model
	deleteConfirmRoomID   string
	deleteConfirmRoomName string
	deleteConfirmIndex    int
	errorMessage          string
}

func (m manageRoomsModel) Init() tea.Cmd {
	return fullScreenInit(m.focusInputIfNeeded())
}

func (m manageRoomsModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m manageRoomsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case CreateRoomFailed:
		m.mode = manageRoomsModeCreate
		m.errorMessage = msg.Message
		m.syncInputPlaceholder()
		return m, m.input.Focus()
	case JoinRoomFailed:
		m.mode = manageRoomsModeJoin
		m.errorMessage = msg.Message
		m.syncInputPlaceholder()
		return m, m.input.Focus()
	case DeleteRoomFailed:
		m.mode = manageRoomsModeDeleteConfirm
		m.errorMessage = msg.Message
		return m, nil
	case DeleteRoomSucceeded:
		m.mode = manageRoomsModeMenu
		m.errorMessage = ""
		m.deleteConfirmRoomID = ""
		m.deleteConfirmRoomName = ""
		return m, nil
	case tea.KeyPressMsg:
		if handled, cmd := m.handleKeyPress(msg); handled {
			return m, cmd
		}
	}

	if m.mode == manageRoomsModeDelete {
		var cmd tea.Cmd
		m.deleteList, cmd = m.deleteList.Update(msg)
		return m, cmd
	}
	if m.mode != manageRoomsModeCreate && m.mode != manageRoomsModeJoin {
		return m, nil
	}
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.errorMessage = ""
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *manageRoomsModel) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch m.mode {
	case manageRoomsModeCreate, manageRoomsModeJoin:
		return m.handleInputKey(msg)
	case manageRoomsModeDelete:
		return m.handleDeleteListKey(msg)
	case manageRoomsModeDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	}

	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		return true, requestBack
	case "left":
		m.selectPrevious()
	case "right", "tab":
		m.selectNext()
	case keySend:
		return true, m.selectCurrent()
	default:
		return false, nil
	}
	return true, nil
}

func (m *manageRoomsModel) handleInputKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		m.mode = manageRoomsModeMenu
		m.errorMessage = ""
		m.input.Blur()
		return true, nil
	case keySend:
		value := strings.TrimSpace(m.input.Value())
		if m.mode == manageRoomsModeJoin {
			return true, requestJoinRoom(value)
		}
		return true, requestCreateRoom(value)
	default:
		return false, nil
	}
}

func (m *manageRoomsModel) handleDeleteListKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		m.mode = manageRoomsModeMenu
		m.errorMessage = ""
		return true, nil
	case keySend:
		item, ok := m.deleteList.SelectedItem().(myChatsRoomItem)
		if !ok || item.id == "" {
			return true, nil
		}
		m.mode = manageRoomsModeDeleteConfirm
		m.errorMessage = ""
		m.deleteConfirmRoomID = item.id
		m.deleteConfirmRoomName = item.title
		m.deleteConfirmIndex = 1 // Default to Cancel for destructive confirmations.
		return true, nil
	default:
		return false, nil
	}
}

func (m *manageRoomsModel) handleDeleteConfirmKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		m.mode = manageRoomsModeDelete
		m.errorMessage = ""
		return true, nil
	case "left", "right", "tab":
		m.deleteConfirmIndex = (m.deleteConfirmIndex + 1) % 2
		return true, nil
	case keySend:
		if m.deleteConfirmIndex == 1 {
			m.mode = manageRoomsModeDelete
			m.errorMessage = ""
			return true, nil
		}
		return true, requestDeleteRoom(m.deleteConfirmRoomID)
	default:
		return false, nil
	}
}

func (m *manageRoomsModel) selectPrevious() {
	m.selectedIndex = (m.selectedIndex - 1 + len(manageRoomsItems)) % len(manageRoomsItems)
}

func (m *manageRoomsModel) selectNext() {
	m.selectedIndex = (m.selectedIndex + 1) % len(manageRoomsItems)
}

func (m *manageRoomsModel) selectCurrent() tea.Cmd {
	switch manageRoomsItems[m.selectedIndex].section {
	case manageRoomsSectionCreate:
		m.mode = manageRoomsModeCreate
	case manageRoomsSectionJoin:
		m.mode = manageRoomsModeJoin
	case manageRoomsSectionDelete:
		m.mode = manageRoomsModeDelete
	default:
		return nil
	}
	m.errorMessage = ""
	m.input.Reset()
	m.syncInputPlaceholder()
	return m.focusInputIfNeeded()
}

func (m *manageRoomsModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newAuthStyles(isDark)
	m.listStyles = newMyChatsStyles(isDark)
	m.input.SetStyles(inputStyles(isDark))
	m.deleteList.SetDelegate(manageRoomsDeleteDelegate(m.listStyles))
	m.deleteList.Styles = m.listStyles.listStyles(m.deleteList.Width())
	m.deleteList.SetSize(m.deleteList.Width(), m.deleteList.Height())
}

func (m *manageRoomsModel) resize(width, height int) {
	m.screen.resize(width, height)
	layout := manageRoomsLayoutFor(width, height, m.styles)
	m.input.SetWidth(m.fieldInputWidth(layout.content.width))
	listLayout := manageRoomsListLayoutFor(width, height, len(m.deleteList.Items()))
	m.deleteList.Styles = m.listStyles.listStyles(listLayout.content.width)
	m.deleteList.SetSize(listLayout.content.width, listLayout.list.height)
}

func (m *manageRoomsModel) focusInputIfNeeded() tea.Cmd {
	if m.mode != manageRoomsModeCreate && m.mode != manageRoomsModeJoin {
		m.input.Blur()
		return nil
	}
	m.syncInputPlaceholder()
	return m.input.Focus()
}

func (m *manageRoomsModel) syncInputPlaceholder() {
	if m.mode == manageRoomsModeJoin {
		m.input.Placeholder = "XXXX-XXXX"
		return
	}
	m.input.Placeholder = "room name"
}

func (m manageRoomsModel) render() string {
	if m.mode == manageRoomsModeDelete {
		return m.renderDeleteListView()
	}
	layout := manageRoomsLayoutFor(m.screen.width, m.screen.height, m.styles)
	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderBox(layout))
}

func (m manageRoomsModel) renderBox(layout manageRoomsLayout) string {
	sections := []string{
		m.styles.heading.Render(wrapCenter(manageRoomsHeadingLine, layout.content.width)),
		"",
	}
	switch m.mode {
	case manageRoomsModeCreate, manageRoomsModeJoin:
		sections = append(sections, m.renderInputForm(layout.content.width))
		if m.errorMessage != "" {
			sections = append(sections, "", m.styles.error.Render(wrapCenter(m.errorMessage, layout.content.width)))
		}
		hint := manageRoomsCreateHintLine
		if m.mode == manageRoomsModeJoin {
			hint = manageRoomsJoinHintLine
		}
		sections = append(sections, "", m.styles.hint.Render(wrapCenter(hint, layout.content.width)))
	case manageRoomsModeDeleteConfirm:
		sections = append(sections, m.renderDeleteConfirm(layout.content.width))
		if m.errorMessage != "" {
			sections = append(sections, "", m.styles.error.Render(wrapCenter(m.errorMessage, layout.content.width)))
		}
		sections = append(sections, "", m.styles.hint.Render(wrapCenter(manageRoomsConfirmHintLine, layout.content.width)))
	default:
		sections = append(sections,
			m.renderButtonRow(layout.content.width),
			"",
			m.styles.hint.Render(wrapCenter(manageRoomsMenuHintLine, layout.content.width)),
		)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)
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

func (m manageRoomsModel) renderButtonRow(width int) string {
	buttons := make([]string, 0, len(manageRoomsItems))
	for i, item := range manageRoomsItems {
		style := m.styles.tab.MarginRight(manageRoomsButtonGap)
		if i == m.selectedIndex {
			style = m.styles.activeTab.MarginRight(manageRoomsButtonGap)
		}
		buttons = append(buttons, style.Render(item.title))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Center, buttons...)
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(row)
}

func (m manageRoomsModel) renderInputForm(width int) string {
	labelText := m.inputLabel()
	labelWidth := lipgloss.Width(labelText)
	label := m.styles.activeLabel.
		Width(labelWidth).
		Align(lipgloss.Right).
		Render(ansi.Truncate(labelText, labelWidth, ""))
	input := m.styles.inputLine.
		Width(m.fieldInputWidth(width) + m.styles.inputLine.GetHorizontalFrameSize()).
		Render(m.input.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, label, " ", input)
}

func (m manageRoomsModel) renderDeleteListView() string {
	layout := manageRoomsListLayoutFor(m.screen.width, m.screen.height, len(m.deleteList.Items()))
	listContent := fitBlockHeight(m.deleteList.View(), layout.list.height)
	if len(m.deleteList.Items()) == 0 {
		listContent = lipgloss.NewStyle().
			Width(layout.content.width).
			Height(layout.list.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(wrapCenter(manageRoomsDeleteEmptyState, layout.content.width))
	}

	sections := []string{listContent}
	if m.errorMessage != "" {
		sections = append(sections, m.styles.error.Render(wrapCenter(m.errorMessage, layout.content.width)))
	}
	sections = append(sections, "", m.styles.hint.Width(layout.content.width).Align(lipgloss.Center).Render(manageRoomsDeleteHintLine))
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	content = lipgloss.NewStyle().
		Width(layout.content.width).
		Height(layout.content.height).
		Render(fitBlockHeight(content, layout.content.height))

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m manageRoomsModel) renderDeleteConfirm(width int) string {
	prompt := m.styles.heading.Render(wrapCenter("Delete \""+m.deleteConfirmRoomName+"\"?", width))
	deleteButton := m.styles.tab.Render("Delete")
	cancelButton := m.styles.tab.Render("Cancel")
	if m.deleteConfirmIndex == 0 {
		deleteButton = m.styles.activeTab.Render("Delete")
	} else {
		cancelButton = m.styles.activeTab.Render("Cancel")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, deleteButton, "  ", cancelButton)
	buttons = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(buttons)
	return lipgloss.JoinVertical(lipgloss.Center, prompt, "", buttons)
}

func (m manageRoomsModel) fieldInputWidth(width int) int {
	labelWidth := lipgloss.Width(m.inputLabel())
	return max(1, width-labelWidth-1-m.styles.inputLine.GetHorizontalFrameSize())
}

func (m manageRoomsModel) inputLabel() string {
	if m.mode == manageRoomsModeJoin {
		return manageRoomsJoinCodeLabel
	}
	return manageRoomsRoomTitleLabel
}

func newManageRoomsDeleteList(styles myChatsStyles, rooms []RoomListItem) list.Model {
	l := list.New(myChatsRoomItems(rooms), manageRoomsDeleteDelegate(styles), 1, 1)
	l.Title = manageRoomsDeleteTitle
	l.Styles = styles.listStyles(1)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	return l
}

func manageRoomsDeleteDelegate(styles myChatsStyles) list.DefaultDelegate {
	delegate := styles.listDelegate()
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	return delegate
}

var manageRoomsItems = []manageRoomsItem{
	{section: manageRoomsSectionCreate, title: manageRoomsCreateTitle},
	{section: manageRoomsSectionJoin, title: manageRoomsJoinTitle},
	{section: manageRoomsSectionDelete, title: manageRoomsDeleteTitle},
}

func manageRoomsLayoutFor(width, height int, styles authStyles) manageRoomsLayout {
	frame := safeFrameSize(width, height)
	box := manageRoomsBoxSize(frame, styles.box)
	content := frameSize{
		width:  max(1, box.width-styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-styles.box.GetVerticalFrameSize()),
	}
	return manageRoomsLayout{frame: frame, box: box, content: content}
}

func manageRoomsBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	maxWidth := frame.width
	if frame.width > manageRoomsFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= manageRoomsFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > manageRoomsFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= manageRoomsFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, min(manageRoomsTargetBoxWidth, maxWidth)),
		height: max(1, min(manageRoomsTargetBoxHeight, maxHeight)),
	}
}

func manageRoomsListLayoutFor(width, height, itemCount int) manageRoomsListLayout {
	frame := safeFrameSize(width, height)
	visibleItems := min(max(1, itemCount), manageRoomsDeleteMaxVisibleRooms)
	listHeight := manageRoomsDeleteTitleHeight + visibleItems
	content := frameSize{
		width:  max(1, min(manageRoomsListTargetWidth, frame.width-manageRoomsFramePaddingX*2)),
		height: max(1, min(listHeight+manageRoomsDeleteFooterHeight, frame.height-manageRoomsFramePaddingY*2)),
	}
	return manageRoomsListLayout{
		frame:   frame,
		content: content,
		list: frameSize{
			width:  content.width,
			height: min(listHeight, content.height),
		},
	}
}
