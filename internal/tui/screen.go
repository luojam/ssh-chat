package tui

import tea "charm.land/bubbletea/v2"

// screenState owns terminal-wide mechanics shared by full-screen views: frame
// size, dark/light detection, alt-screen wrapping, and background-color init.
type frameSize struct {
	width  int
	height int
}

type screenState struct {
	width  int
	height int
	isDark bool
}

func newScreenState(config Config) screenState {
	return screenState{
		width:  config.Width,
		height: config.Height,
		isDark: true,
	}
}

func fullScreenInit(cmds ...tea.Cmd) tea.Cmd {
	return tea.Batch(append([]tea.Cmd{tea.RequestBackgroundColor}, cmds...)...)
}

func fullScreenView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (s *screenState) resize(width, height int) {
	s.width = width
	s.height = height
}

func (s *screenState) setDark(isDark bool) bool {
	if s.isDark == isDark {
		return false
	}
	s.isDark = isDark
	return true
}

func (s screenState) frame() frameSize {
	return safeFrameSize(s.width, s.height)
}

func safeFrameSize(width, height int) frameSize {
	return frameSize{
		width:  safeDimension(width),
		height: safeDimension(height),
	}
}

// Bubble Tea can send zero dimensions before the first resize message.
// Rendering still needs at least one cell so string truncation stays valid.
func safeDimension(n int) int {
	return max(1, n)
}
