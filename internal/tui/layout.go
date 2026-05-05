package tui

import "charm.land/lipgloss/v2"

type frameSize struct {
	width  int
	height int
}

func (m model) frame() frameSize {
	return frameSize{
		width:  safeDimension(m.width),
		height: safeDimension(m.height),
	}
}

func (m model) frameWidth() int {
	return safeDimension(m.width)
}

func (m model) messageAreaHeight() int {
	return messageAreaHeight(safeDimension(m.height))
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

func messageAreaHeight(frameHeight int) int {
	switch {
	case frameHeight <= 1:
		return 0
	case frameHeight == 2:
		return 1
	default:
		footerRows := 1
		if inputFrameVisible(frameHeight) {
			footerRows = 3 // separator + composer + separator
		}
		// Two rows for title and divider.
		return max(1, frameHeight-2-footerRows)
	}
}
