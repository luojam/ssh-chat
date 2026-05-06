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
// forward in the flow without selecting a domain object, such as Welcome entering
// the Main Menu.
type ContinueRequested struct{}

// BackRequested asks the session to return to the previous full-screen view.
type BackRequested struct{}

// AuthSubmissionRequested asks the session layer to authenticate or create a user.
// The TUI only collects form data; storage and password validation live outside
// the view.
type AuthSubmissionRequested struct {
	Mode            AuthMode
	Username        string
	Password        string
	ConfirmPassword string
}

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

// RoomSelected asks the session to enter the selected Room. The TUI supplies
// identity only; Session still owns Room membership lifecycle.
type RoomSelected struct {
	RoomID string
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

func requestAuthSubmission(submission AuthSubmissionRequested) tea.Cmd {
	return func() tea.Msg {
		return submission
	}
}

func requestMainMenuSelection(action MainMenuAction) tea.Cmd {
	return func() tea.Msg {
		return MainMenuSelectionRequested{Action: action}
	}
}

func requestRoomSelection(roomID string) tea.Cmd {
	return func() tea.Msg {
		return RoomSelected{RoomID: roomID}
	}
}

func requestLeave() tea.Msg {
	return LeaveRequested{}
}
