package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	manageRoomsHeadingLine     = "MANAGE ROOMS"
	manageRoomsCreateTitle     = "Create Room"
	manageRoomsJoinTitle       = "Join Room"
	manageRoomsRoomTitleLabel  = "Room title"
	manageRoomsJoinCodeLabel   = "Join code"
	manageRoomsMenuHintLine    = "←/→ • enter • esc back • ctrl+c quit"
	manageRoomsCreateHintLine  = "enter create • esc back • ctrl+c quit"
	manageRoomsJoinHintLine    = "enter join • esc back • ctrl+c quit"
	manageRoomsTargetBoxWidth  = 60
	manageRoomsTargetBoxHeight = 14
	manageRoomsFramePaddingX   = 2
	manageRoomsFramePaddingY   = 1
	manageRoomsButtonGap       = 3
)

type manageRoomsMode int

const (
	manageRoomsModeMenu manageRoomsMode = iota
	manageRoomsModeCreate
	manageRoomsModeJoin
)

type manageRoomsSection int

const (
	manageRoomsSectionCreate manageRoomsSection = iota
	manageRoomsSectionJoin
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

func NewManageRooms(config Config) tea.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "room name"
	input.SetStyles(inputStyles(true))

	m := manageRoomsModel{
		screen:        newScreenState(config),
		styles:        newAuthStyles(true),
		mode:          manageRoomsModeMenu,
		selectedIndex: 0,
		input:         input,
	}
	m.resize(config.Width, config.Height)
	return m
}

type manageRoomsModel struct {
	screen screenState

	styles        authStyles
	mode          manageRoomsMode
	selectedIndex int
	input         textinput.Model
	errorMessage  string
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
	case tea.KeyPressMsg:
		if handled, cmd := m.handleKeyPress(msg); handled {
			return m, cmd
		}
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
	if m.mode == manageRoomsModeCreate || m.mode == manageRoomsModeJoin {
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
	default:
		return nil
	}
	m.errorMessage = ""
	m.input.Reset()
	m.syncInputPlaceholder()
	return m.input.Focus()
}

func (m *manageRoomsModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newAuthStyles(isDark)
	m.input.SetStyles(inputStyles(isDark))
}

func (m *manageRoomsModel) resize(width, height int) {
	m.screen.resize(width, height)
	layout := manageRoomsLayoutFor(width, height, m.styles)
	m.input.SetWidth(m.fieldInputWidth(layout.content.width))
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
	if m.mode == manageRoomsModeCreate || m.mode == manageRoomsModeJoin {
		sections = append(sections, m.renderInputForm(layout.content.width))
		if m.errorMessage != "" {
			sections = append(sections, "", m.styles.error.Render(wrapCenter(m.errorMessage, layout.content.width)))
		}
		hint := manageRoomsCreateHintLine
		if m.mode == manageRoomsModeJoin {
			hint = manageRoomsJoinHintLine
		}
		sections = append(sections, "", m.styles.hint.Render(wrapCenter(hint, layout.content.width)))
	} else {
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

var manageRoomsItems = []manageRoomsItem{
	{section: manageRoomsSectionCreate, title: manageRoomsCreateTitle},
	{section: manageRoomsSectionJoin, title: manageRoomsJoinTitle},
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
