package tui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

const (
	headerTitleColor      = "15"
	headerDividerColor    = "240"
	emptyMessageColor     = "8"
	authorColor           = "12"
	systemAuthorColor     = "11"
	darkLocalAuthorColor  = "10"
	lightLocalAuthorColor = "2"
	darkBodyColor         = "15"
	composerAccentColor   = "10"
	composerMutedColor    = "8"
	composerSepColor      = "240"
	darkWelcomeBorder     = "7"
	lightWelcomeBorder    = "8"
	darkWelcomeTitle      = "10"
	lightWelcomeTitle     = "2"
	darkWelcomeLogo       = "10"
	lightWelcomeLogo      = "2"
	darkWelcomePrimary    = "15"
	lightWelcomePrimary   = "0"

	mainMenuBorderColor      = "62"
	mainMenuSelectedButtonBg = "63"
	mainMenuSelectedButtonFg = "15"
	mainMenuInactiveButtonFg = "250"
	mainMenuHeaderColor      = "14"
	lightMainMenuSelectedFg  = "0"
	lightMainMenuSelectedBg  = "159"
	lightMainMenuInactiveFg  = "255"
	myChatsSelectedMarker    = "2"
)

type baseStyles struct {
	headerTitle      lipgloss.Style
	headerDivider    lipgloss.Style
	empty            lipgloss.Style
	author           lipgloss.Style
	systemAuthor     lipgloss.Style
	localAuthor      lipgloss.Style
	body             lipgloss.Style
	composer         lipgloss.Style
	inputSep         lipgloss.Style
	welcomeBox       lipgloss.Style
	welcomeTitle     lipgloss.Style
	welcomeLogo      lipgloss.Style
	welcomePrimary   lipgloss.Style
	welcomeSecondary lipgloss.Style
}

func newBaseStyles(isDark bool) baseStyles {
	s := baseStyles{
		headerTitle:      lipgloss.NewStyle().Foreground(lipgloss.Color(headerTitleColor)).Bold(true).Align(lipgloss.Center),
		headerDivider:    lipgloss.NewStyle().Foreground(lipgloss.Color(headerDividerColor)).Faint(true),
		empty:            lipgloss.NewStyle().Foreground(lipgloss.Color(emptyMessageColor)),
		author:           lipgloss.NewStyle().Foreground(lipgloss.Color(authorColor)),
		systemAuthor:     lipgloss.NewStyle().Foreground(lipgloss.Color(systemAuthorColor)).Bold(true),
		composer:         lipgloss.NewStyle(),
		inputSep:         lipgloss.NewStyle().Foreground(lipgloss.Color(composerSepColor)).Faint(true),
		welcomeBox:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(2, 2).Align(lipgloss.Center),
		welcomeTitle:     lipgloss.NewStyle().Bold(true),
		welcomeLogo:      lipgloss.NewStyle().Bold(true),
		welcomePrimary:   lipgloss.NewStyle().Bold(true),
		welcomeSecondary: lipgloss.NewStyle().Faint(true),
	}

	if isDark {
		s.localAuthor = lipgloss.NewStyle().Foreground(lipgloss.Color(darkLocalAuthorColor))
		s.body = lipgloss.NewStyle().Foreground(lipgloss.Color(darkBodyColor))
		s.welcomeBox = s.welcomeBox.BorderForeground(lipgloss.Color(darkWelcomeBorder))
		s.welcomeTitle = s.welcomeTitle.Foreground(lipgloss.Color(darkWelcomeTitle))
		s.welcomeLogo = s.welcomeLogo.Foreground(lipgloss.Color(darkWelcomeLogo))
		s.welcomePrimary = s.welcomePrimary.Foreground(lipgloss.Color(darkWelcomePrimary))
		return s
	}

	s.localAuthor = lipgloss.NewStyle().Foreground(lipgloss.Color(lightLocalAuthorColor))
	s.body = lipgloss.NewStyle()
	s.welcomeBox = s.welcomeBox.BorderForeground(lipgloss.Color(lightWelcomeBorder))
	s.welcomeTitle = s.welcomeTitle.Foreground(lipgloss.Color(lightWelcomeTitle))
	s.welcomeLogo = s.welcomeLogo.Foreground(lipgloss.Color(lightWelcomeLogo))
	s.welcomePrimary = s.welcomePrimary.Foreground(lipgloss.Color(lightWelcomePrimary))
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

type mainMenuStyles struct {
	box            lipgloss.Style
	heading        lipgloss.Style
	hint           lipgloss.Style
	button         lipgloss.Style
	selectedButton lipgloss.Style
}

func newMainMenuStyles(isDark bool) mainMenuStyles {
	borderColor := lipgloss.Color(mainMenuBorderColor)
	selectedForeground := lipgloss.Color(mainMenuSelectedButtonFg)
	selectedBackground := lipgloss.Color(mainMenuSelectedButtonBg)
	inactiveForeground := lipgloss.Color(mainMenuInactiveButtonFg)
	if !isDark {
		borderColor = lipgloss.Color(lightWelcomeBorder)
		selectedForeground = lipgloss.Color(lightMainMenuSelectedFg)
		selectedBackground = lipgloss.Color(lightMainMenuSelectedBg)
		inactiveForeground = lipgloss.Color(lightMainMenuInactiveFg)
	}

	return mainMenuStyles{
		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(2, 3).
			BorderForeground(borderColor),
		heading: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(mainMenuHeaderColor)),
		hint:    lipgloss.NewStyle().Faint(true).Foreground(inactiveForeground),
		button: lipgloss.NewStyle().
			Foreground(inactiveForeground).
			Padding(0, 2).
			MarginRight(mainMenuButtonGap),
		selectedButton: lipgloss.NewStyle().
			Bold(true).
			Foreground(selectedForeground).
			Background(selectedBackground).
			Padding(0, 2).
			MarginRight(mainMenuButtonGap),
	}
}

type myChatsStyles struct {
	isDark     bool
	listBorder lipgloss.Style
	hint       lipgloss.Style
}

func newMyChatsStyles(isDark bool) myChatsStyles {
	return myChatsStyles{
		isDark:     isDark,
		listBorder: lipgloss.NewStyle(),
		hint: lipgloss.NewStyle().
			Faint(true).
			Foreground(lipgloss.Color(mainMenuInactiveButtonFg)),
	}
}

func (s myChatsStyles) listStyles(width int) list.Styles {
	styles := list.DefaultStyles(s.isDark)
	width = safeDimension(width)
	// The default list title bar has left padding, which makes a centered title
	// look slightly right-shifted inside our bordered list. Center the title bar
	// itself and keep the title style content-sized.
	styles.TitleBar = styles.TitleBar.
		Padding(0, 0, 1, 0).
		Width(width).
		Align(lipgloss.Center)
	styles.Title = styles.Title.
		Foreground(lipgloss.Color(mainMenuHeaderColor)).
		Bold(true).
		UnsetBackground().
		Padding(0, 0)
	return styles
}

func (s myChatsStyles) listDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles = list.NewDefaultItemStyles(s.isDark)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "→"}).
		BorderForeground(lipgloss.Color(myChatsSelectedMarker))
	delegate.ShowDescription = false
	return delegate
}
