package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
