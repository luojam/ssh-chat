package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	composerPrompt   = "> "
	composerQuitHint = "ctrl+l to go back • Esc to quit"
	inputSepRune     = '─'
)

// "Message..." anchors the left; spaces push the hint close to the right end.
// Falls back to the base text alone when the box is too narrow to fit both.
func buildPlaceholder(inputW int) string {
	const base = "Message..."
	gap := inputW - len(base) - len(composerQuitHint) - 1
	if gap < 1 {
		return base
	}
	return base + strings.Repeat(" ", gap) + composerQuitHint
}

func newComposer(isDark bool) (textinput.Model, tea.Cmd) {
	input := textinput.New()
	input.Prompt = composerPrompt
	input.SetStyles(inputStyles(isDark))
	return input, input.Focus()
}

func (m model) renderComposer(width int) string {
	return renderFullWidth(m.styles.composer.Inline(true), width, m.input.View())
}

// renderComposerSection draws optional separator lines above and below the input row.
func (m model) renderComposerSection(width int, showFrame bool) string {
	line := m.renderComposer(width)
	if !showFrame {
		return line
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderInputSeparator(width),
		line,
		m.renderInputSeparator(width),
	)
}

func (m model) renderInputSeparator(width int) string {
	line := strings.Repeat(string(inputSepRune), max(1, width))
	return renderFullWidth(m.styles.inputSep, width, line)
}
