package tui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

const (
	headerBackgroundColor = "62"
	headerTitleColor      = "15"
	headerStatusColor     = "252"
	headerHintColor       = "245"
	emptyMessageColor     = "8"
	authorColor           = "12"
	darkMineAuthorColor   = "10"
	lightMineAuthorColor  = "2"
	darkBodyColor         = "15"
	composerAccentColor   = "10"
	composerMutedColor    = "8"
)

type styles struct {
	header       lipgloss.Style
	headerTitle  lipgloss.Style
	headerStatus lipgloss.Style
	headerHint   lipgloss.Style
	empty        lipgloss.Style
	author       lipgloss.Style
	mineAuthor   lipgloss.Style
	body         lipgloss.Style
	composer     lipgloss.Style
}

func newStyles(isDark bool) styles {
	headerBackground := lipgloss.Color(headerBackgroundColor)

	s := styles{
		header:       lipgloss.NewStyle().Background(headerBackground),
		headerTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color(headerTitleColor)).Background(headerBackground).Bold(true),
		headerStatus: lipgloss.NewStyle().Foreground(lipgloss.Color(headerStatusColor)).Background(headerBackground),
		headerHint:   lipgloss.NewStyle().Foreground(lipgloss.Color(headerHintColor)).Background(headerBackground).Faint(true),
		empty:        lipgloss.NewStyle().Foreground(lipgloss.Color(emptyMessageColor)),
		author:       lipgloss.NewStyle().Foreground(lipgloss.Color(authorColor)),
		composer:     lipgloss.NewStyle(),
	}

	if isDark {
		s.mineAuthor = lipgloss.NewStyle().Foreground(lipgloss.Color(darkMineAuthorColor))
		s.body = lipgloss.NewStyle().Foreground(lipgloss.Color(darkBodyColor))
		return s
	}

	s.mineAuthor = lipgloss.NewStyle().Foreground(lipgloss.Color(lightMineAuthorColor))
	s.body = lipgloss.NewStyle()
	return s
}

func inputStyles(isDark bool) textinput.Styles {
	s := textinput.DefaultStyles(isDark)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(composerAccentColor))
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(composerMutedColor))
	s.Focused.Text = lipgloss.NewStyle()
	s.Blurred.Prompt = s.Focused.Prompt
	s.Blurred.Placeholder = s.Focused.Placeholder
	s.Blurred.Text = s.Focused.Text
	return s
}
