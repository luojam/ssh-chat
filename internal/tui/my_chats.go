package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	chatListTitle          = "My Chats"
	chatListHintLine       = "↑/up ↓/down • enter select • esc back"
	myChatsTargetWidth     = 56
	myChatsTargetHeight    = 20
	myChatsListTargetWidth = 32
	myChatsFramePaddingX   = 2
	myChatsFramePaddingY   = 1
	myChatsContainerPadX   = 2
	myChatsContainerPadY   = 1
)

var chatListBorderStyle = lipgloss.NewStyle()

type chatRoomItem struct {
	title string
}

func (i chatRoomItem) Title() string       { return i.title }
func (i chatRoomItem) Description() string { return "" }
func (i chatRoomItem) FilterValue() string { return i.title }

func NewMyChats(config Config) tea.Model {
	m := myChatsModel{
		width:  config.Width,
		height: config.Height,
		isDark: true,
		list:   newMyChatsList(true),
	}
	m.resize(config.Width, config.Height)
	return m
}

type myChatsModel struct {
	width  int
	height int
	isDark bool

	list list.Model
}

func (m myChatsModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m myChatsModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m myChatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case keyQuitCtrlC:
			return m, requestQuit
		case keyQuitEsc:
			return m, requestBack
		case keySend:
			return m, requestContinue
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *myChatsModel) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}
	m.isDark = isDark
	m.list.SetDelegate(newMyChatsDelegate(isDark))
	m.list.Styles = myChatsListStyles(isDark, m.list.Width())
}

func (m *myChatsModel) resize(width, height int) {
	m.width = width
	m.height = height

	layout := myChatsLayoutFor(width, height)
	listHeight := max(1, layout.listContent.height)
	m.list.SetSize(layout.listContent.width, listHeight)
	m.list.Styles = myChatsListStyles(m.isDark, layout.listContent.width)
}

func (m myChatsModel) render() string {
	layout := myChatsLayoutFor(m.width, m.height)
	listHeight := max(1, layout.listContent.height)
	listBox := chatListBorderStyle.
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
		lipgloss.NewStyle().
			Width(layout.content.width).
			Align(lipgloss.Center).
			Faint(true).
			Foreground(lipgloss.Color(dashboardInactiveButtonFg)).
			Render(chatListHintLine),
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

func newMyChatsList(isDark bool) list.Model {
	l := list.New(myChatRoomItems(), newMyChatsDelegate(isDark), 1, 1)
	l.Title = chatListTitle
	l.Styles = myChatsListStyles(isDark, 1)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()

	return l
}

func myChatRoomItems() []list.Item {
	return []list.Item{
		chatRoomItem{title: "Town Square"},
	}
}

func newMyChatsDelegate(isDark bool) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles = list.NewDefaultItemStyles(isDark)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "→"}).BorderForeground(lipgloss.Color("2"))
	delegate.ShowDescription = false
	return delegate
}

func myChatsListStyles(isDark bool, width int) list.Styles {
	styles := list.DefaultStyles(isDark)
	width = safeDimension(width)
	// The default list title bar has left padding, which makes a centered title
	// look slightly right-shifted inside our bordered list. Center the title bar
	// itself and keep the title style content-sized.
	styles.TitleBar = styles.TitleBar.
		Padding(0, 0, 1, 0).
		Width(width).
		Align(lipgloss.Center)
	styles.Title = styles.Title.
		Foreground(lipgloss.Color(dashboardHeaderColor)).
		Bold(true).
		UnsetBackground().
		Padding(0, 0)
	return styles
}

type myChatsLayout struct {
	frame       frameSize
	container   frameSize
	content     frameSize
	listBox     frameSize
	listContent frameSize
}

func myChatsLayoutFor(width, height int) myChatsLayout {
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
		width:  max(1, listBox.width-chatListBorderStyle.GetHorizontalFrameSize()),
		height: max(1, listBox.height-chatListBorderStyle.GetVerticalFrameSize()),
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
