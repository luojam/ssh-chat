package tui

import (
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	dashboardTitleLine       = "MENU"
	dashboardTownSquareTitle = "My Chats"

	dashboardFramePaddingX  = 2
	dashboardFramePaddingY  = 1
	dashboardNavTargetWidth = 22
	dashboardNavMinWidth    = 12
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
	box        lipgloss.Style
	navPane    lipgloss.Style
	panelPane  lipgloss.Style
	panelTitle lipgloss.Style
}

type dashboardLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
	nav     frameSize
	panel   frameSize
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
	m.resize(m.width, m.height)
}

func (m *dashboardModel) resize(width, height int) {
	m.width = width
	m.height = height

	layout := dashboardLayoutFor(width, height, m.styles)
	navContent := dashboardPaneContentSize(layout.nav, m.styles.navPane)
	m.list.SetSize(navContent.width, navContent.height)
	m.list.Styles = dashboardListStyles(m.isDark, navContent.width)
}

func (m dashboardModel) render() string {
	layout := dashboardLayoutFor(m.width, m.height, m.styles)

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderDashboardBox(layout))
}

func (m dashboardModel) renderDashboardBox(layout dashboardLayout) string {
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderDashboardNav(layout),
		m.renderDashboardPanel(layout),
	)

	return m.styles.box.
		Width(layout.box.width).
		Height(layout.box.height).
		Render(content)
}

func (m dashboardModel) renderDashboardNav(layout dashboardLayout) string {
	navContent := dashboardPaneContentSize(layout.nav, m.styles.navPane)
	return m.styles.navPane.
		Width(layout.nav.width).
		Height(layout.nav.height).
		Render(fitBlockHeight(m.list.View(), navContent.height))
}

func (m dashboardModel) renderDashboardPanel(layout dashboardLayout) string {
	contentWidth := max(1, layout.panel.width-m.styles.panelPane.GetHorizontalFrameSize())
	contentHeight := max(1, layout.panel.height-m.styles.panelPane.GetVerticalFrameSize())
	content := fitBlockHeight(m.renderSelectedDashboardPanel(contentWidth), contentHeight)

	return m.styles.panelPane.
		Width(layout.panel.width).
		Height(layout.panel.height).
		Render(content)
}

func (m dashboardModel) renderSelectedDashboardPanel(width int) string {
	title := dashboardTownSquareTitle
	if item, ok := m.list.SelectedItem().(dashboardItem); ok && item.title != "" {
		title = item.title
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.styles.panelTitle.Render(wrapCenter(ansi.Truncate(title, width, ""), width)),
	)
}

func newDashboardList(isDark bool) list.Model {
	delegate := newDashboardDelegate(isDark)

	l := list.New(dashboardItems(), delegate, 1, 1)
	l.Title = dashboardTitleLine
	l.Styles = dashboardListStyles(isDark, 1)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
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
	borderColor := lipgloss.Color(darkWelcomeBorder)
	if !isDark {
		borderColor = lipgloss.Color(lightWelcomeBorder)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		BorderForeground(borderColor)

	navPane := lipgloss.NewStyle().
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRightForeground(borderColor)

	panelPane := lipgloss.NewStyle().
		Padding(0, 2)

	return dashboardStyles{
		box:        box,
		navPane:    navPane,
		panelPane:  panelPane,
		panelTitle: lipgloss.NewStyle().Bold(true),
	}
}

// dashboardLayoutFor keeps sizing logic centralized so list sizing matches render output.
func dashboardLayoutFor(width, height int, styles dashboardStyles) dashboardLayout {
	frame := safeFrameSize(width, height)
	box := dashboardBoxSize(frame, styles.box)

	content := frameSize{
		width:  max(1, box.width-styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-styles.box.GetVerticalFrameSize()),
	}

	navPaneWidth := dashboardNavPaneWidth(content.width, styles.navPane)
	nav := frameSize{
		width:  navPaneWidth,
		height: content.height,
	}
	panel := frameSize{
		width:  max(1, content.width-navPaneWidth),
		height: content.height,
	}

	return dashboardLayout{
		frame:   frame,
		box:     box,
		content: content,
		nav:     nav,
		panel:   panel,
	}
}

func dashboardNavPaneWidth(contentWidth int, style lipgloss.Style) int {
	contentWidth = safeDimension(contentWidth)
	if contentWidth <= dashboardNavMinWidth+style.GetHorizontalFrameSize() {
		return max(1, contentWidth/2)
	}
	return min(dashboardNavTargetWidth, max(dashboardNavMinWidth, contentWidth/3))
}

func dashboardPaneContentSize(pane frameSize, style lipgloss.Style) frameSize {
	return frameSize{
		width:  max(1, pane.width-style.GetHorizontalFrameSize()),
		height: max(1, pane.height-style.GetVerticalFrameSize()),
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
