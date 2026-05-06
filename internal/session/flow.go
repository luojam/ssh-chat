package session

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/auth"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

const townSquareRoomID = "town-square"

// applyFlowIntent is the Session navigation table. Keeping transitions here
// concentrates the important invariant: only the Room View may hold a Room
// Subscription, and leaving that View closes the Subscription before navigation.
func (m *model) applyFlowIntent(msg tea.Msg) tea.Cmd {
	if m.closed {
		return nil
	}

	switch msg := msg.(type) {
	case tui.ContinueRequested:
		return m.continueFromCurrentView()
	case tui.BackRequested:
		return m.backFromCurrentView()
	case tui.AuthSubmissionRequested:
		return m.submitAuth(msg)
	case tui.MainMenuSelectionRequested:
		return m.openMainMenuSelection(msg.Action)
	case tui.RoomSelected:
		return m.enterSelectedRoom(msg.RoomID)
	case tui.LeaveRequested:
		return m.leaveRoomView()
	default:
		return nil
	}
}

func (m *model) continueFromCurrentView() tea.Cmd {
	switch m.view {
	case viewWelcome:
		return m.showStandaloneView(viewAuth)
	default:
		return nil
	}
}

func (m *model) backFromCurrentView() tea.Cmd {
	switch m.view {
	case viewAuth:
		return m.showStandaloneView(viewWelcome)
	case viewMainMenu:
		return m.showStandaloneView(viewAuth)
	case viewMyChats:
		return m.showStandaloneView(viewMainMenu)
	default:
		return nil
	}
}

func (m *model) submitAuth(submission tui.AuthSubmissionRequested) tea.Cmd {
	if m.view != viewAuth || m.authService == nil {
		return nil
	}

	user, err := m.authenticate(submission)
	if err != nil {
		return m.showAuthError(authErrorMessage(err))
	}

	m.authenticatedUser = &user
	m.member.ID = chatMemberID(user.ID)
	m.member.Name = user.Username
	return m.showStandaloneView(viewMainMenu)
}

func (m *model) authenticate(submission tui.AuthSubmissionRequested) (auth.User, error) {
	switch submission.Mode {
	case tui.AuthModeSignup:
		return m.authService.Signup(m.ctx, submission.Username, submission.Password, submission.ConfirmPassword)
	default:
		return m.authService.Login(m.ctx, submission.Username, submission.Password)
	}
}

func (m *model) showAuthError(message string) tea.Cmd {
	var cmd tea.Cmd
	m.ui, cmd = m.ui.Update(tui.AuthFailed{Message: message})
	return cmd
}

func authErrorMessage(err error) string {
	switch {
	case errors.Is(err, auth.ErrUsernameTaken):
		return "Username is already taken."
	case errors.Is(err, auth.ErrPasswordMismatch):
		return "Passwords do not match."
	case errors.Is(err, auth.ErrInvalidCredentials):
		return "Invalid username or password."
	case errors.Is(err, auth.ErrInvalidInput):
		return "Username and password are required."
	default:
		return "Authentication failed."
	}
}

func chatMemberID(userID string) chat.MemberID {
	return chat.MemberID(userID)
}

func (m *model) openMainMenuSelection(action tui.MainMenuAction) tea.Cmd {
	if m.view != viewMainMenu {
		return nil
	}

	switch action {
	case tui.MainMenuActionMyChats:
		return m.showStandaloneView(viewMyChats)
	default:
		return nil
	}
}

func (m *model) showStandaloneView(view viewState) tea.Cmd {
	if m.closed {
		return nil
	}

	m.closeRoomMembership()
	m.view = view
	m.ui = m.newUI(view)
	return m.ui.Init()
}

func (m *model) enterSelectedRoom(roomID string) tea.Cmd {
	if m.view != viewMyChats || roomID != townSquareRoomID {
		return nil
	}
	return m.enterRoomView()
}

func (m *model) enterRoomView() tea.Cmd {
	if m.closed || m.inRoomView() {
		return nil
	}

	m.closeRoomMembership()
	m.view = viewChat
	m.ui = m.newUI(viewChat)
	m.subscription = m.room.Join(m.member)

	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m *model) leaveRoomView() tea.Cmd {
	if !m.inRoomView() {
		return nil
	}
	return m.showStandaloneView(viewMainMenu)
}

func (m *model) closeRoomMembership() {
	if m.subscription == nil {
		return
	}
	m.subscription.Close()
	m.subscription = nil
}

func (m model) inRoomView() bool {
	return m.view == viewChat && m.subscription != nil
}

func (m model) newUI(view viewState) tea.Model {
	config := tui.Config{Width: m.width, Height: m.height}
	switch view {
	case viewWelcome:
		return tui.NewWelcome(config)
	case viewAuth:
		return tui.NewAuth(config)
	case viewMainMenu:
		return tui.NewMainMenu(config)
	case viewMyChats:
		config.Rooms = []tui.RoomListItem{{ID: townSquareRoomID, Title: "Town Square"}}
		return tui.NewMyChats(config)
	case viewChat:
		return tui.NewRoomView(config)
	default:
		return tui.NewWelcome(config)
	}
}
