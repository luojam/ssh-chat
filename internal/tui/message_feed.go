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
)

type MessageAuthorRole int

const (
	MessageAuthorNormal MessageAuthorRole = iota
	MessageAuthorLocal
	MessageAuthorSystem
)

type message struct {
	author string
	body   string
	role   MessageAuthorRole
}

// MessageReceived is terminal-ready display data. Session owns Room-event wording
// and local author labelling; the TUI only lays out and styles the supplied line.
type MessageReceived struct {
	Author string
	Body   string
	Role   MessageAuthorRole
}

func newMessageViewport() viewport.Model {
	vp := viewport.New()
	vp.SoftWrap = false
	vp.FillHeight = true
	vp.MouseWheelEnabled = false
	return vp
}

func (m *roomViewModel) requestSend() tea.Cmd {
	body := strings.TrimSpace(m.input.Value())
	if body == "" {
		return nil
	}

	m.input.Reset()
	return func() tea.Msg {
		return SendRequested{Body: body}
	}
}

func (m *roomViewModel) receiveMessage(msg MessageReceived) {
	m.messages = append(m.messages, message{
		author: msg.Author,
		body:   msg.Body,
		role:   msg.Role,
	})
	m.syncMessageViewport(true)
}

// Keep viewport synchronization in mutation paths, not View. Bubble Tea child
// models often carry state such as scroll position; rendering should observe it.
func (m *roomViewModel) syncMessageViewport(follow bool) {
	wasAtBottom := m.viewport.AtBottom()
	layout := m.layout()

	m.viewport.SetWidth(layout.width)
	m.viewport.SetHeight(layout.messageRows)
	m.viewport.SetContentLines(m.bottomAlignedMessageLines(layout.width, layout.messageRows))
	if follow || wasAtBottom {
		m.viewport.GotoBottom()
	}
}

func (m roomViewModel) renderMessages() string {
	return m.viewport.View()
}

func (m roomViewModel) messageLines(width int) []string {
	if len(m.messages) == 0 {
		return []string{m.styles.empty.Render(ansi.Truncate(emptyStateText, width, ""))}
	}

	lines := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		lines = append(lines, m.renderMessage(msg, width)...)
	}
	return lines
}

func (m roomViewModel) bottomAlignedMessageLines(width, height int) []string {
	lines := m.messageLines(width)
	if height <= 0 || len(lines) >= height {
		return lines
	}

	padding := make([]string, height-len(lines), height)
	return append(padding, lines...)
}

func (m roomViewModel) renderMessage(msg message, width int) []string {
	if width <= authorColumnWidth+1 {
		return []string{m.styles.body.Render(ansi.Truncate(msg.body, width, ""))}
	}

	bodyWidth := max(1, width-authorColumnWidth-1)
	bodyLines := strings.Split(ansi.Wrap(msg.body, bodyWidth, " "), "\n")
	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}

	authorCell := fixedCell(msg.author, authorColumnWidth)
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

func (m roomViewModel) messageAuthorStyle(msg message) lipgloss.Style {
	switch msg.role {
	case MessageAuthorLocal:
		return m.styles.localAuthor
	case MessageAuthorSystem:
		return m.styles.systemAuthor
	default:
		return m.styles.author
	}
}
