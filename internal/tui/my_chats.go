package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	myChatsTitle           = "My Chats"
	myChatsHintLine        = "↑/↓ move • enter • esc back • ctrl+c quit"
	myChatsTargetWidth     = 56
	myChatsTargetHeight    = 20
	myChatsListTargetWidth = 32
	myChatsFramePaddingX   = 2
	myChatsFramePaddingY   = 1
	myChatsContainerPadX   = 2
	myChatsContainerPadY   = 1
)

type myChatsRoomItem struct {
	id    string
	title string
}

func (i myChatsRoomItem) Title() string       { return i.title }
func (i myChatsRoomItem) Description() string { return "" }
func (i myChatsRoomItem) FilterValue() string { return i.title }

func NewMyChats(config Config) tea.Model {
	styles := newMyChatsStyles(true)
	m := myChatsModel{
		screen: newScreenState(config),
		styles: styles,
		list:   newMyChatsList(styles, config.Rooms),
	}
	m.resize(config.Width, config.Height)
	return m
}

type myChatsModel struct {
	screen screenState

	styles myChatsStyles
	list   list.Model
}

func (m myChatsModel) Init() tea.Cmd {
	return fullScreenInit()
}

func (m myChatsModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m myChatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case keyQuit:
			return m, requestQuit
		case keyBack:
			return m, requestBack
		case keySend:
			return m, m.requestSelectedRoom()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *myChatsModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newMyChatsStyles(isDark)
	m.list.SetDelegate(m.styles.listDelegate())
	m.list.Styles = m.styles.listStyles(m.list.Width())
}

func (m *myChatsModel) resize(width, height int) {
	m.screen.resize(width, height)

	layout := myChatsLayoutFor(width, height, m.styles)
	listHeight := max(1, layout.listContent.height)
	m.list.SetSize(layout.listContent.width, listHeight)
	m.list.Styles = m.styles.listStyles(layout.listContent.width)
}

func (m myChatsModel) render() string {
	layout := myChatsLayoutFor(m.screen.width, m.screen.height, m.styles)
	listHeight := max(1, layout.listContent.height)
	listBox := m.styles.listBorder.
		Width(layout.listBox.width).
		Height(layout.listBox.height).
		Render(fitBlockHeight(m.list.View(), listHeight))

	centeredListBox := lipgloss.NewStyle().
		Width(layout.content.width).
		Align(lipgloss.Center).
		Render(listBox)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		centeredListBox,
		"",
		m.styles.hint.
			Width(layout.content.width).
			Align(lipgloss.Center).
			Render(myChatsHintLine),
	)

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

func (m myChatsModel) requestSelectedRoom() tea.Cmd {
	item, ok := m.list.SelectedItem().(myChatsRoomItem)
	if !ok || item.id == "" {
		return nil
	}
	return requestRoomSelection(item.id)
}

func newMyChatsList(styles myChatsStyles, rooms []RoomListItem) list.Model {
	l := list.New(myChatsRoomItems(rooms), styles.listDelegate(), 1, 1)
	l.Title = myChatsTitle
	l.Styles = styles.listStyles(1)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()

	return l
}

func myChatsRoomItems(rooms []RoomListItem) []list.Item {
	items := make([]list.Item, 0, len(rooms))
	for _, room := range rooms {
		items = append(items, myChatsRoomItem{id: room.ID, title: room.Title})
	}
	return items
}

type myChatsLayout struct {
	frame       frameSize
	container   frameSize
	content     frameSize
	listBox     frameSize
	listContent frameSize
}

func myChatsLayoutFor(width, height int, styles myChatsStyles) myChatsLayout {
	frame := safeFrameSize(width, height)
	container := myChatsContainerSize(frame)
	content := frameSize{
		width:  max(1, container.width-myChatsContainerPadX*2),
		height: max(1, container.height-myChatsContainerPadY*2),
	}
	listBox := frameSize{
		width:  max(1, min(myChatsListTargetWidth, content.width)),
		height: max(1, content.height-2), // spacer plus hint line below the list.
	}
	listContent := frameSize{
		width:  max(1, listBox.width-styles.listBorder.GetHorizontalFrameSize()),
		height: max(1, listBox.height-styles.listBorder.GetVerticalFrameSize()),
	}

	return myChatsLayout{
		frame:       frame,
		container:   container,
		content:     content,
		listBox:     listBox,
		listContent: listContent,
	}
}

func myChatsContainerSize(frame frameSize) frameSize {
	maxWidth := frame.width
	if frame.width > myChatsFramePaddingX*2 {
		maxWidth -= myChatsFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > myChatsFramePaddingY*2 {
		maxHeight -= myChatsFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, min(myChatsTargetWidth, maxWidth)),
		height: max(1, min(myChatsTargetHeight, maxHeight)),
	}
}
