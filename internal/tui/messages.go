package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	authorColumnWidth = 8
	emptyStateText    = "No messages yet."
	localAuthor       = "you"
	systemAuthor      = "system"
	unknownAuthor     = "unknown"
)

type message struct {
	author string
	body   string
	mine   bool
}

// SendRequested is an intent from the interface, not an accepted chat message.
// The session layer decides whether this becomes backend state.
type SendRequested struct {
	Body string
}

// QuitRequested lets the session layer release per-connection resources before
// it asks Bubble Tea to stop the program.
type QuitRequested struct{}

// ContinueRequested is emitted by the welcome view when the user chooses to
// enter the chat view.
type ContinueRequested struct{}

// MessageReceived is display data for the terminal. Backend message IDs,
// delivery rules, and storage details stay outside the TUI.
type MessageReceived struct {
	Author string
	Body   string
	Mine   bool
}

func newMessageViewport() viewport.Model {
	vp := viewport.New()
	vp.SoftWrap = false
	vp.FillHeight = true
	vp.MouseWheelEnabled = false
	return vp
}

func (m *model) requestSend() tea.Cmd {
	body := strings.TrimSpace(m.input.Value())
	if body == "" {
		return nil
	}

	m.input.Reset()
	return func() tea.Msg {
		return SendRequested{Body: body}
	}
}

func requestQuit() tea.Msg {
	return QuitRequested{}
}

func requestContinue() tea.Msg {
	return ContinueRequested{}
}

func (m *model) receiveMessage(msg MessageReceived) {
	m.messages = append(m.messages, message{
		author: msg.Author,
		body:   msg.Body,
		mine:   msg.Mine,
	})
	m.syncMessageViewport(true)
}

// Keep viewport synchronization in mutation paths, not View. Bubble Tea child
// models often carry state such as scroll position; rendering should observe it.
func (m *model) syncMessageViewport(follow bool) {
	wasAtBottom := m.viewport.AtBottom()
	layout := m.layout()

	m.viewport.SetWidth(layout.width)
	m.viewport.SetHeight(layout.messageRows)
	m.viewport.SetContentLines(m.bottomAlignedMessageLines(layout.width, layout.messageRows))
	if follow || wasAtBottom {
		m.viewport.GotoBottom()
	}
}

func (m model) renderMessages() string {
	return m.viewport.View()
}

func (m model) messageLines(width int) []string {
	if len(m.messages) == 0 {
		return []string{m.styles.empty.Render(ansi.Truncate(emptyStateText, width, ""))}
	}

	lines := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		lines = append(lines, m.renderMessage(msg, width)...)
	}
	return lines
}

func (m model) bottomAlignedMessageLines(width, height int) []string {
	lines := m.messageLines(width)
	if height <= 0 || len(lines) >= height {
		return lines
	}

	padding := make([]string, height-len(lines), height)
	return append(padding, lines...)
}

func (m model) renderMessage(msg message, width int) []string {
	if width <= authorColumnWidth+1 {
		return []string{m.styles.body.Render(ansi.Truncate(msg.body, width, ""))}
	}

	bodyWidth := max(1, width-authorColumnWidth-1)
	bodyLines := strings.Split(ansi.Wrap(msg.body, bodyWidth, " "), "\n")
	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}

	authorCell := fixedCell(msg.displayAuthor(), authorColumnWidth)
	authorStyle := m.messageAuthorStyle(msg)

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

func (m model) messageAuthorStyle(msg message) lipgloss.Style {
	if msg.mine {
		return m.styles.mineAuthor
	}
	if msg.displayAuthor() == systemAuthor {
		return m.styles.systemAuthor
	}
	return m.styles.author
}

func (msg message) displayAuthor() string {
	if msg.mine {
		return localAuthor
	}
	if msg.author == "" {
		return unknownAuthor
	}
	return msg.author
}
