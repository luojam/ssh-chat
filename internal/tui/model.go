package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	appName             = "ssh-chat"
	statusConnected     = "connected"
	authorColumnWidth   = 8
	composerPrompt      = "> "
	composerPlaceholder = "Message..."
)

type Config struct {
	Term   string
	Width  int
	Height int
}

func New(config Config) tea.Model {
	input := textinput.New()
	input.Prompt = composerPrompt
	input.Placeholder = composerPlaceholder
	input.SetStyles(inputStyles(true))
	focusCmd := input.Focus()

	m := model{
		term:       config.Term,
		width:      config.Width,
		height:     config.Height,
		isDark:     true,
		styles:     newStyles(true),
		input:      input,
		viewport:   viewport.New(),
		initialCmd: focusCmd,
	}
	m.resize(config.Width, config.Height)
	m.refreshViewport()
	return m
}

type message struct {
	author string
	body   string
	mine   bool
}

type model struct {
	term    string
	profile string
	width   int
	height  int
	isDark  bool

	styles   styles
	input    textinput.Model
	viewport viewport.Model
	messages []message

	initialCmd tea.Cmd
}

type styles struct {
	header       lipgloss.Style
	headerTitle  lipgloss.Style
	headerStatus lipgloss.Style
	empty        lipgloss.Style
	author       lipgloss.Style
	mineAuthor   lipgloss.Style
	body         lipgloss.Style
	composer     lipgloss.Style
}

func newStyles(isDark bool) styles {
	if isDark {
		return styles{
			header:       lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")),
			headerTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Bold(true),
			headerStatus: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("62")),
			empty:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			author:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
			mineAuthor:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
			body:         lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
			composer:     lipgloss.NewStyle(),
		}
	}

	return styles{
		header:       lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")),
		headerTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Bold(true),
		headerStatus: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("62")),
		empty:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		author:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		mineAuthor:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		body:         lipgloss.NewStyle(),
		composer:     lipgloss.NewStyle(),
	}
}

func inputStyles(isDark bool) textinput.Styles {
	s := textinput.DefaultStyles(isDark)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s.Focused.Text = lipgloss.NewStyle()
	s.Blurred.Prompt = s.Focused.Prompt
	s.Blurred.Placeholder = s.Focused.Placeholder
	s.Blurred.Text = s.Focused.Text
	return s
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.initialCmd)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.ColorProfileMsg:
		m.profile = msg.String()
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.sendLocalMessage()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m *model) setDark(isDark bool) {
	if m.isDark == isDark {
		return
	}
	m.isDark = isDark
	m.styles = newStyles(isDark)
	m.input.SetStyles(inputStyles(isDark))
	m.refreshViewport()
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height

	w := safeDimension(width)
	m.input.SetWidth(inputWidth(w))
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(messageAreaHeight(height))
	m.viewport.SoftWrap = false
	m.viewport.FillHeight = true
	m.viewport.MouseWheelEnabled = false
	m.refreshViewport()
}

func (m *model) sendLocalMessage() {
	body := strings.TrimSpace(m.input.Value())
	if body == "" {
		return
	}

	m.messages = append(m.messages, message{
		author: "you",
		body:   body,
		mine:   true,
	})
	m.input.Reset()
	m.refreshViewport()
}

func (m *model) refreshViewport() {
	m.viewport.SetContentLines(m.messageLines(safeDimension(m.width)))
	m.viewport.GotoBottom()
}

func (m model) render() string {
	width := safeDimension(m.width)
	height := safeDimension(m.height)

	switch height {
	case 1:
		return m.renderComposer(width)
	case 2:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderMessages(width, 1),
			m.renderComposer(width),
		)
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(width),
			m.renderMessages(width, height-2),
			m.renderComposer(width),
		)
	}
}

func (m model) renderHeader(width int) string {
	title := appName
	if width <= lipgloss.Width(title) {
		return m.styles.header.Width(width).MaxWidth(width).Inline(true).Render(ansi.Truncate(title, width, ""))
	}

	statusWidth := max(0, width-lipgloss.Width(title)-1)
	status := ansi.Truncate(statusConnected, statusWidth, "")
	gap := max(1, width-lipgloss.Width(title)-lipgloss.Width(status))
	content := m.styles.headerTitle.Render(title) +
		strings.Repeat(" ", gap) +
		m.styles.headerStatus.Render(status)

	return m.styles.header.Width(width).MaxWidth(width).Inline(true).Render(ansi.Truncate(content, width, ""))
}

func (m model) renderMessages(width, height int) string {
	vp := m.viewport
	vp.SetWidth(width)
	vp.SetHeight(height)
	vp.SetContentLines(m.messageLines(width))
	vp.GotoBottom()
	return vp.View()
}

func (m model) renderComposer(width int) string {
	input := m.input
	input.SetWidth(inputWidth(width))
	line := ansi.Truncate(input.View(), width, "")
	return m.styles.composer.Width(width).MaxWidth(width).Inline(true).Render(line)
}

func (m model) messageLines(width int) []string {
	if len(m.messages) == 0 {
		return []string{m.styles.empty.Render(ansi.Truncate("No messages yet.", width, ""))}
	}

	lines := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		lines = append(lines, m.renderMessage(msg, width)...)
	}
	return lines
}

func (m model) renderMessage(msg message, width int) []string {
	if width <= authorColumnWidth+1 {
		return []string{m.styles.body.Render(ansi.Truncate(msg.body, width, ""))}
	}

	bodyWidth := max(1, width-authorColumnWidth-1)
	wrapped := ansi.Wrap(msg.body, bodyWidth, " ")
	bodyLines := strings.Split(wrapped, "\n")
	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}

	author := msg.author
	if msg.mine {
		author = "you"
	}
	if author == "" {
		author = "unknown"
	}
	authorCell := fixedCell(author, authorColumnWidth)
	authorStyle := m.styles.author
	if msg.mine {
		authorStyle = m.styles.mineAuthor
	}

	lines := make([]string, 0, len(bodyLines))
	for i, bodyLine := range bodyLines {
		if i == 0 {
			lines = append(lines, authorStyle.Render(authorCell)+" "+m.styles.body.Render(bodyLine))
			continue
		}
		lines = append(lines, strings.Repeat(" ", authorColumnWidth+1)+m.styles.body.Render(bodyLine))
	}
	return lines
}

func fixedCell(s string, width int) string {
	if lipgloss.Width(s) > width {
		return ansi.Truncate(s, width, "")
	}
	return fmt.Sprintf("%-*s", width, s)
}

func safeDimension(n int) int {
	return max(1, n)
}

func inputWidth(width int) int {
	return max(1, width-lipgloss.Width(composerPrompt))
}

func messageAreaHeight(height int) int {
	switch {
	case height <= 1:
		return 0
	case height == 2:
		return 1
	default:
		return max(1, height-2)
	}
}
