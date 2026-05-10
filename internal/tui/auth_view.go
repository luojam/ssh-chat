package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	authTitleLine           = "SSH Chat"
	authLoginTab            = "Log in"
	authSignupTab           = "Sign up"
	authUsernameLabel       = "Username"
	authPasswordLabel       = "Password"
	authConfirmLabel        = "Confirm password"
	authLoginHint           = "↑/↓/←/→ switch • enter login • esc back"
	authSignupHint          = "↑/↓/←/→ switch • enter create • esc back"
	authTargetBoxWidth      = 58
	authFramePaddingX       = 2
	authFramePaddingY       = 1
	authVerticalPaddingRows = 3
	authModeLogin           = AuthModeLogin
	authModeSignup          = AuthModeSignup
)

type AuthMode int

const (
	AuthModeLogin AuthMode = iota
	AuthModeSignup
)

func NewAuth(config Config) tea.Model {
	inputs := newAuthInputs(true)
	m := authModel{
		screen:     newScreenState(config),
		styles:     newAuthStyles(true),
		mode:       authModeLogin,
		focusIndex: 0,
		inputs:     inputs,
	}
	m.resize(config.Width, config.Height)
	m.focusActiveInput()
	return m
}

type authModel struct {
	screen screenState

	styles       authStyles
	mode         AuthMode
	focusIndex   int
	inputs       []textinput.Model
	errorMessage string
}

func (m authModel) Init() tea.Cmd {
	return fullScreenInit(m.focusActiveInput())
}

func (m authModel) View() tea.View {
	return fullScreenView(m.render())
}

func (m authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setDark(msg.IsDark())
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case AuthFailed:
		m.errorMessage = msg.Message
	case tea.KeyPressMsg:
		if handled, cmd := m.handleKeyPress(msg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m *authModel) handleKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if msg.String() != keySend {
		m.errorMessage = ""
	}
	switch msg.String() {
	case keyQuit:
		return true, requestQuit
	case keyBack:
		return true, requestBack
	case "tab", "down":
		return true, m.focusNext()
	case "shift+tab", "up":
		return true, m.focusPrevious()
	case "left", "right":
		return true, m.toggleMode()
	case keySend:
		return true, requestAuthSubmission(m.authSubmission())
	default:
		return false, nil
	}
}

func (m *authModel) setDark(isDark bool) {
	if !m.screen.setDark(isDark) {
		return
	}
	m.styles = newAuthStyles(isDark)
	styles := inputStyles(isDark)
	for i := range m.inputs {
		m.inputs[i].SetStyles(styles)
	}
}

func (m *authModel) resize(width, height int) {
	m.screen.resize(width, height)
	layout := authLayoutFor(width, height, m.styles, 1)
	inputWidth := m.fieldInputWidth(layout.content.width)
	for i := range m.inputs {
		m.inputs[i].SetWidth(inputWidth)
	}
}

func (m *authModel) focusNext() tea.Cmd {
	m.focusIndex = (m.focusIndex + 1) % m.activeInputCount()
	return m.focusActiveInput()
}

func (m *authModel) focusPrevious() tea.Cmd {
	m.focusIndex = (m.focusIndex - 1 + m.activeInputCount()) % m.activeInputCount()
	return m.focusActiveInput()
}

func (m *authModel) toggleMode() tea.Cmd {
	if m.mode == AuthModeLogin {
		m.mode = AuthModeSignup
	} else {
		m.mode = AuthModeLogin
	}
	if m.focusIndex >= m.activeInputCount() {
		m.focusIndex = m.activeInputCount() - 1
	}
	return m.focusActiveInput()
}

func (m *authModel) focusActiveInput() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.inputs {
		if i == m.focusIndex {
			cmd = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()
	}
	return cmd
}

func (m authModel) activeInputCount() int {
	if m.mode == AuthModeSignup {
		return 3
	}
	return 2
}

func (m authModel) authSubmission() AuthSubmissionRequested {
	return AuthSubmissionRequested{
		Mode:            m.mode,
		Username:        strings.TrimSpace(m.inputs[0].Value()),
		Password:        m.inputs[1].Value(),
		ConfirmPassword: m.inputs[2].Value(),
	}
}

func (m authModel) render() string {
	layout := authLayoutFor(m.screen.width, m.screen.height, m.styles, 1)
	body := m.renderAuthBody(layout.content.width)
	layout = authLayoutFor(m.screen.width, m.screen.height, m.styles, m.desiredContentHeight(layout.content.width))

	return lipgloss.NewStyle().
		Width(layout.frame.width).
		Height(layout.frame.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.renderAuthBox(layout, body))
}

func (m authModel) renderAuthBox(layout authLayout, body string) string {
	content := body
	if rows := m.topPaddingRows(layout.content.height, body); rows > 0 {
		content = strings.Repeat("\n", rows) + body
	}

	content = lipgloss.NewStyle().
		Width(layout.content.width).
		Height(layout.content.height).
		Render(fitBlockHeight(content, layout.content.height))

	return m.styles.box.
		Width(layout.box.width).
		Height(layout.box.height).
		Render(content)
}

func (m authModel) renderAuthBody(width int) string {
	sections := []string{
		m.renderTabs(width),
		"",
		m.renderFields(width),
	}
	if m.errorMessage != "" {
		sections = append(sections, "", m.renderError(width))
	}
	sections = append(sections, "", m.renderHint(width))
	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func (m authModel) desiredContentHeight(width int) int {
	loginBodyHeight := m.authBodyHeightForMode(width, AuthModeLogin)
	signupBodyHeight := m.authBodyHeightForMode(width, AuthModeSignup)
	return max(loginBodyHeight, signupBodyHeight) + authVerticalPaddingRows*2
}

func (m authModel) authBodyHeightForMode(width int, mode AuthMode) int {
	m.mode = mode
	return lipgloss.Height(m.renderAuthBody(width))
}

func (m authModel) topPaddingRows(contentHeight int, body string) int {
	return max(0, min(authVerticalPaddingRows, contentHeight-lipgloss.Height(body)))
}

func (m authModel) renderTitle(width int) string {
	return m.styles.heading.Render(wrapCenter(authTitleLine, width))
}

func (m authModel) renderTabs(width int) string {
	login := m.styles.tab.Render(authLoginTab)
	signup := m.styles.tab.Render(authSignupTab)
	if m.mode == AuthModeLogin {
		login = m.styles.activeTab.Render(authLoginTab)
	} else {
		signup = m.styles.activeTab.Render(authSignupTab)
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, login, "  ", signup),
	)
}

func (m authModel) renderFields(width int) string {
	labels := []string{authUsernameLabel, authPasswordLabel, authConfirmLabel}
	// Use the widest auth label in both modes so login inputs line up with
	// signup inputs when switching tabs.
	labelWidth := authLabelWidth(labels)
	inputWidth := m.fieldInputWidth(width)

	rows := make([]string, 0, m.activeInputCount())
	for i := 0; i < m.activeInputCount(); i++ {
		labelStyle := m.styles.label
		if i == m.focusIndex {
			labelStyle = m.styles.activeLabel
		}

		label := labelStyle.
			Width(labelWidth).
			Align(lipgloss.Right).
			Render(ansi.Truncate(labels[i], labelWidth, ""))
		input := m.styles.inputLine.
			Width(inputWidth + m.styles.inputLine.GetHorizontalFrameSize()).
			Render(m.inputs[i].View())
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, " ", input))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m authModel) fieldInputWidth(width int) int {
	labels := []string{authUsernameLabel, authPasswordLabel, authConfirmLabel}
	labelWidth := authLabelWidth(labels)
	return max(1, width-labelWidth-1-m.styles.inputLine.GetHorizontalFrameSize())
}

func authLabelWidth(labels []string) int {
	width := 0
	for _, label := range labels {
		width = max(width, lipgloss.Width(label))
	}
	return width
}

func (m authModel) renderError(width int) string {
	return m.styles.error.Render(wrapCenter(m.errorMessage, width))
}

func (m authModel) renderHint(width int) string {
	hint := authLoginHint
	if m.mode == AuthModeSignup {
		hint = authSignupHint
	}
	return m.styles.hint.Render(wrapCenter(hint, width))
}

func newAuthInputs(isDark bool) []textinput.Model {
	styles := inputStyles(isDark)
	inputs := []textinput.Model{textinput.New(), textinput.New(), textinput.New()}
	placeholders := []string{"name", "password", "password again"}
	for i := range inputs {
		inputs[i].Prompt = ""
		inputs[i].Placeholder = placeholders[i]
		inputs[i].SetStyles(styles)
	}
	inputs[1].EchoMode = textinput.EchoPassword
	inputs[2].EchoMode = textinput.EchoPassword
	return inputs
}

type authLayout struct {
	frame   frameSize
	box     frameSize
	content frameSize
}

func authLayoutFor(width, height int, styles authStyles, desiredContentHeight int) authLayout {
	frame := safeFrameSize(width, height)
	box := authBoxSize(frame, styles.box, desiredContentHeight)
	content := frameSize{
		width:  max(1, box.width-styles.box.GetHorizontalFrameSize()),
		height: max(1, box.height-styles.box.GetVerticalFrameSize()),
	}
	return authLayout{frame: frame, box: box, content: content}
}

func authBoxSize(frame frameSize, style lipgloss.Style, desiredContentHeight int) frameSize {
	maxWidth := frame.width
	if frame.width > authFramePaddingX*2+style.GetHorizontalFrameSize() {
		maxWidth -= authFramePaddingX * 2
	}

	maxHeight := frame.height
	if frame.height > authFramePaddingY*2+style.GetVerticalFrameSize() {
		maxHeight -= authFramePaddingY * 2
	}

	desiredBoxHeight := safeDimension(desiredContentHeight) + style.GetVerticalFrameSize()
	return frameSize{
		width:  max(1, min(authTargetBoxWidth, maxWidth)),
		height: max(1, min(desiredBoxHeight, maxHeight)),
	}
}
