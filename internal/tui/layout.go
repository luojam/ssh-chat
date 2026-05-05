package tui

import "charm.land/lipgloss/v2"

type frameSize struct {
	width  int
	height int
}

// chatLayout is the small contract between sizing and rendering.
// Keeping every row decision here prevents render paths and viewport sizing from
// drifting apart as the TUI grows.
type chatLayout struct {
	width          int
	height         int
	messageRows    int
	showHeader     bool
	showInputFrame bool
}

func safeFrameSize(width, height int) frameSize {
	return frameSize{
		width:  safeDimension(width),
		height: safeDimension(height),
	}
}

func (m model) frameWidth() int {
	return safeDimension(m.width)
}

func (m model) layout() chatLayout {
	return chatLayoutFor(m.width, m.height)
}

func chatLayoutFor(width, height int) chatLayout {
	frame := safeFrameSize(width, height)
	layout := chatLayout{
		width:          frame.width,
		height:         frame.height,
		showHeader:     frame.height >= 3,
		showInputFrame: inputFrameVisible(frame.height),
	}

	usedRows := 1 // The composer is always visible; it is the primary input affordance.
	if layout.showHeader {
		usedRows += 2 // Header title plus divider.
	}
	if layout.showInputFrame {
		usedRows += 2 // Separators above and below the composer.
	}
	layout.messageRows = max(0, frame.height-usedRows)

	return layout
}

// Bubble Tea can send zero dimensions before the first resize message.
// Rendering still needs at least one cell so string truncation stays valid.
func safeDimension(n int) int {
	return max(1, n)
}

func inputWidth(width int) int {
	return max(1, width-lipgloss.Width(composerPrompt))
}

// inputFrameVisible returns true if horizontal input separators are shown—i.e.,
// header, header divider, separators, and at least one message row fit.
func inputFrameVisible(frameHeight int) bool {
	return frameHeight >= 6
}
