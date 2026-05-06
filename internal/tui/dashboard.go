package tui

import (
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	dashboardTitleLine       = "MENU"
	dashboardTownSquareTitle = "Town Square"

	dashboardFramePaddingX = 2
	dashboardFramePaddingY = 1
)

// dashboardItem is presentation data only. Application actions for a selected
// row belong in the session layer; this view emits only navigation intent.
type dashboardItem struct {
	title string
}

func (i dashboardItem) Title() string       { return i.title }
func (i dashboardItem) Description() string { return "" }
func (i dashboardItem) FilterValue() string { return i.title }

type dashboardStyles struct {
	box lipgloss.Style
}

type dashboardLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
}

func NewDashboard(config Config) tea.Model {
	m := dashboardModel{
		width:  config.Width,
		height: config.Height,
		isDark: true,
		styles: newDashboardStyles(true),
		list:   newDashboardList(true),
	}
	m.resize(config.Width, config.Height)
	return m
}

type dashboardModel struct {
	width  int
	height int
	isDark bool

	styles dashboardStyles
	list   list.Model
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m dashboardModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *dashboardModel) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyQuitCtrlC, keyQuitEsc:
		return true, requestQuit
	case keySend:
		return true, requestContinue
	default:
		return false, nil
	}
}

func (m *dashboardModel) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}

	m.isDark = isDark
	m.styles = newDashboardStyles(isDark)
	m.list.SetDelegate(newDashboardDelegate(isDark))
	m.list.Styles = dashboardListStyles(isDark, m.list.Width())
}

func (m *dashboardModel) resize(width, height int) {
	m.width = width
	m.height = height

	layout := dashboardLayoutFor(width, height, m.styles.box)
	m.list.SetSize(layout.content.width, layout.content.height)
	m.list.Styles = dashboardListStyles(m.isDark, layout.content.width)
}

func (m dashboardModel) render() string {
	layout := dashboardLayoutFor(m.width, m.height, m.styles.box)

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderDashboardBox(layout))
}

func (m dashboardModel) renderDashboardBox(layout dashboardLayout) string {
	return m.styles.box.
		Width(layout.box.width).
		Height(layout.box.height).
		Render(fitBlockHeight(m.list.View(), layout.content.height))
}

func newDashboardList(isDark bool) list.Model {
	delegate := newDashboardDelegate(isDark)

	l := list.New(dashboardItems(), delegate, 1, 1)
	l.Title = dashboardTitleLine
	l.Styles = dashboardListStyles(isDark, 1)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(true)
	l.SetShowPagination(true)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()

	return l
}

func dashboardItems() []list.Item {
	return []list.Item{
		dashboardItem{title: dashboardTownSquareTitle},
	}
}

func newDashboardDelegate(isDark bool) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles = list.NewDefaultItemStyles(isDark)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderLeft(true).
		BorderStyle(lipgloss.Border{
			Left: ">",
		})
	return delegate
}

func dashboardListStyles(isDark bool, width int) list.Styles {
	styles := list.DefaultStyles(isDark)
	width = safeDimension(width)
	styles.TitleBar = styles.TitleBar.Padding(0, 0, 3, 0).Width(width).Align(lipgloss.Center)
	return styles
}

func newDashboardStyles(isDark bool) dashboardStyles {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Align(lipgloss.Center)

	if isDark {
		box = box.BorderForeground(lipgloss.Color(darkWelcomeBorder))
	} else {
		box = box.BorderForeground(lipgloss.Color(lightWelcomeBorder))
	}

	return dashboardStyles{
		box: box,
	}
}

// dashboardLayoutFor keeps sizing logic centralized so list sizing matches render output.
func dashboardLayoutFor(width, height int, style lipgloss.Style) dashboardLayout {
	frame := safeFrameSize(width, height)
	box := dashboardBoxSize(frame, style)

	content := frameSize{
		width:  max(1, box.width-style.GetHorizontalFrameSize()),
		height: max(1, box.height-style.GetVerticalFrameSize()),
	}

	return dashboardLayout{
		frame:   frame,
		box:     box,
		content: content,
	}
}

func dashboardBoxSize(frame frameSize, style lipgloss.Style) frameSize {
	width := frame.width
	if frame.width > dashboardFramePaddingX*2+style.GetHorizontalFrameSize() {
		width -= dashboardFramePaddingX * 2
	}

	height := frame.height
	if frame.height > dashboardFramePaddingY*2+style.GetVerticalFrameSize() {
		height -= dashboardFramePaddingY * 2
	}

	return frameSize{
		width:  max(1, width),
		height: max(1, height),
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
