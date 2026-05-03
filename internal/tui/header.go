package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	appName            = "ssh-chat"
	statusConnected    = "connected"
	quitHint           = "esc/ctrl+c to quit"
	headerLeftPadding  = 1
	headerRightPadding = 1
)

func (m model) renderHeader(width int) string {
	title := appName
	titleWidth := lipgloss.Width(title)
	leftPadding := min(headerLeftPadding, max(0, width))
	if width <= leftPadding+titleWidth {
		content := renderHeaderEdgePadding(leftPadding) +
			m.styles.headerTitle.Render(ansi.Truncate(title, width-leftPadding, ""))
		return m.renderHeaderLine(content, width)
	}

	statusText := statusConnected
	statusWidth := lipgloss.Width(statusText)
	rightPadding := min(headerRightPadding, max(0, width-leftPadding-titleWidth))
	contentWidth := width - leftPadding - rightPadding
	availableBetween := contentWidth - titleWidth - statusWidth - 2
	if availableBetween < 1 {
		statusWidth = max(0, contentWidth-titleWidth-1)
		status := ansi.Truncate(statusText, statusWidth, "")
		gap := max(0, contentWidth-titleWidth-lipgloss.Width(status))
		content := renderHeaderEdgePadding(leftPadding) +
			m.styles.headerTitle.Render(title) +
			m.renderHeaderGap(gap) +
			m.styles.headerStatus.Render(status) +
			renderHeaderEdgePadding(rightPadding)
		return m.renderHeaderLine(content, width)
	}

	hintText := ansi.Truncate(quitHint, availableBetween, "")
	hintWidth := lipgloss.Width(hintText)
	hintStart := leftPadding + titleWidth + 1 + (availableBetween-hintWidth)/2
	statusStart := width - rightPadding - statusWidth
	gap1 := hintStart - (leftPadding + titleWidth)
	gap2 := statusStart - (hintStart + hintWidth)
	content := renderHeaderEdgePadding(leftPadding) +
		m.styles.headerTitle.Render(title) +
		m.renderHeaderGap(gap1) +
		m.styles.headerHint.Render(hintText) +
		m.renderHeaderGap(gap2) +
		m.styles.headerStatus.Render(statusText) +
		renderHeaderEdgePadding(rightPadding)

	return m.renderHeaderLine(content, width)
}

func (m model) renderHeaderGap(width int) string {
	if width <= 0 {
		return ""
	}
	return m.styles.header.Inline(true).Render(strings.Repeat(" ", width))
}

func renderHeaderEdgePadding(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(" ", width)
}

func (m model) renderHeaderLine(content string, width int) string {
	line := ansi.Truncate(content, width, "")
	if padding := width - lipgloss.Width(line); padding > 0 {
		line += m.renderHeaderGap(padding)
	}
	return line
}
