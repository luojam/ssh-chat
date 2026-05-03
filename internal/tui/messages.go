package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	authorColumnWidth = 8
	emptyStateText    = "No messages yet."
	localAuthor       = "you"
	unknownAuthor     = "unknown"
)

type message struct {
	author string
	body   string
	mine   bool
}

func newMessageViewport() viewport.Model {
	vp := viewport.New()
	vp.SoftWrap = false
	vp.FillHeight = true
	vp.MouseWheelEnabled = false
	return vp
}

// Local echo is the first message source. Network delivery can use the same
// message type later without coupling the renderer to SSH session details.
func (m *model) sendLocalMessage() {
	body := strings.TrimSpace(m.input.Value())
	if body == "" {
		return
	}

	m.messages = append(m.messages, message{
		author: localAuthor,
		body:   body,
		mine:   true,
	})
	m.input.Reset()
	m.syncMessageViewport()
}

// Keep viewport synchronization in mutation paths, not View. Bubble Tea child
// models often carry state such as scroll position; rendering should observe it.
func (m *model) syncMessageViewport() {
	width := m.frameWidth()
	height := m.messageAreaHeight()

	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
	m.viewport.SetContentLines(m.bottomAlignedMessageLines(width, height))
	m.viewport.GotoBottom()
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

func fixedCell(s string, width int) string {
	if lipgloss.Width(s) > width {
		return ansi.Truncate(s, width, "")
	}
	return fmt.Sprintf("%-*s", width, s)
}
