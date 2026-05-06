package tui

import tea "charm.land/bubbletea/v2"

// SendRequested is an intent from the interface, not an accepted Room message.
// The session layer decides whether this becomes backend state.
type SendRequested struct {
	Body string
}

// QuitRequested lets the session layer release per-connection resources before
// it asks Bubble Tea to stop the program.
type QuitRequested struct{}

// ContinueRequested is emitted when a full-screen view asks the session to move
// forward in the flow: welcome enters main menu, My Chats enters the Room View.
type ContinueRequested struct{}

// BackRequested asks the session to return to the previous full-screen view.
type BackRequested struct{}

// MainMenuAction identifies which main menu button was selected.
type MainMenuAction int

const (
	MainMenuActionMyChats MainMenuAction = iota
	MainMenuActionManageChats
	MainMenuActionSettings
)

// MainMenuSelectionRequested asks the session to open the selected main menu area.
type MainMenuSelectionRequested struct {
	Action MainMenuAction
}

// LeaveRequested is emitted by the room view when the user chooses to leave the
// room without quitting the SSH session.
type LeaveRequested struct{}

func requestQuit() tea.Msg {
	return QuitRequested{}
}

func requestContinue() tea.Msg {
	return ContinueRequested{}
}

func requestBack() tea.Msg {
	return BackRequested{}
}

func requestMainMenuSelection(action MainMenuAction) tea.Cmd {
	return func() tea.Msg {
		return MainMenuSelectionRequested{Action: action}
	}
}

func requestLeave() tea.Msg {
	return LeaveRequested{}
}
