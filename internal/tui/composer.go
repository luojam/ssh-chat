package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	composerPrompt      = "> "
	composerPlaceholder = "Message..."
)

func newComposer(isDark bool) (textinput.Model, tea.Cmd) {
	input := textinput.New()
	input.Prompt = composerPrompt
	input.Placeholder = composerPlaceholder
	input.SetStyles(inputStyles(isDark))
	return input, input.Focus()
}

func (m model) renderComposer(width int) string {
	input := m.input
	input.SetWidth(inputWidth(width))
	line := ansi.Truncate(input.View(), width, "")
	return m.styles.composer.Width(width).MaxWidth(width).Inline(true).Render(line)
}

// renderComposerSection draws optional separator lines above and below the input row.
func (m model) renderComposerSection(width, frameHeight int) string {
	line := m.renderComposer(width)
	if !inputFrameVisible(frameHeight) {
		return line
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderInputSeparator(width),
		line,
		m.renderInputSeparator(width),
	)
}

const inputSepRune = '─'

func (m model) renderInputSeparator(width int) string {
	line := strings.Repeat(string(inputSepRune), max(1, width))
	return m.styles.inputSep.Width(width).MaxWidth(width).Render(line)
}
