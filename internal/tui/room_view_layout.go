package tui

import "charm.land/lipgloss/v2"

// roomViewLayout is the small contract between sizing and rendering.
// Keeping every row decision here prevents render paths and viewport sizing from
// drifting apart as the TUI grows.
type roomViewLayout struct {
	width          int
	height         int
	messageRows    int
	showHeader     bool
	showInputFrame bool
}

func (m roomViewModel) frameWidth() int {
	return safeDimension(m.screen.width)
}

func (m roomViewModel) layout() roomViewLayout {
	return roomViewLayoutFor(m.screen.width, m.screen.height)
}

func roomViewLayoutFor(width, height int) roomViewLayout {
	frame := safeFrameSize(width, height)
	layout := roomViewLayout{
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

func inputWidth(width int) int {
	return max(1, width-lipgloss.Width(composerPrompt))
}

// inputFrameVisible returns true if horizontal input separators are shown—i.e.,
// header, header divider, separators, and at least one message row fit.
func inputFrameVisible(frameHeight int) bool {
	return frameHeight >= 6
}
