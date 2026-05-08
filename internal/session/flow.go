package session

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/auth"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

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
	case tui.LinkSSHKeySelectionRequested:
		return m.answerSSHKeyLink(msg.Link)
	case tui.MainMenuSelectionRequested:
		return m.openMainMenuSelection(msg.Action)
	case tui.RoomSelected:
		return m.enterSelectedRoom(msg.RoomID)
	case tui.CreateRoomRequested:
		return m.createRoom(msg.Title)
	case tui.LeaveRequested:
		return m.leaveRoomView()
	default:
		return nil
	}
}

func (m *model) continueFromCurrentView() tea.Cmd {
	switch m.view {
	case viewWelcome:
		if m.isAuthenticated() {
			return m.showStandaloneView(viewMainMenu)
		}
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
		return m.showStandaloneView(viewWelcome)
	case viewMyChats:
		return m.showStandaloneView(viewMainMenu)
	case viewManageRooms:
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

	m.setAuthenticatedUser(user)
	if m.shouldPromptForSSHKeyLink() {
		return m.showStandaloneView(viewLinkSSHKey)
	}
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

func (m *model) authenticateWithConnectionSSHKey() {
	if m.authService == nil || m.connectionSSHKeyFingerprint == "" {
		return
	}
	user, err := m.authService.FindUserBySSHKeyFingerprint(m.ctx, m.connectionSSHKeyFingerprint)
	if err != nil {
		return
	}
	m.setAuthenticatedUser(user)
}

func (m *model) shouldPromptForSSHKeyLink() bool {
	if m.authService == nil || m.connectionSSHPublicKey == "" || m.connectionSSHKeyFingerprint == "" {
		return false
	}
	_, err := m.authService.FindUserBySSHKeyFingerprint(m.ctx, m.connectionSSHKeyFingerprint)
	return errors.Is(err, auth.ErrSSHKeyNotFound)
}

func (m *model) answerSSHKeyLink(link bool) tea.Cmd {
	if m.view != viewLinkSSHKey {
		return nil
	}
	if link && m.authenticatedUser != nil && m.authService != nil {
		// Duplicate same-user links are handled as a no-op by the auth service. If
		// the key was raced onto another user, continue safely without relinking.
		_ = m.authService.LinkSSHKey(m.ctx, *m.authenticatedUser, m.connectionSSHPublicKey, m.connectionSSHKeyFingerprint)
	}
	return m.showStandaloneView(viewMainMenu)
}

func (m *model) setAuthenticatedUser(user auth.User) {
	m.authenticatedUser = &user
	m.member.ID = chat.UserID(user.ID)
	m.member.Name = user.Username
}

func (m *model) openMainMenuSelection(action tui.MainMenuAction) tea.Cmd {
	if m.view != viewMainMenu {
		return nil
	}

	switch action {
	case tui.MainMenuActionMyChats:
		return m.showMyChats()
	case tui.MainMenuActionManageRooms:
		if !m.isAuthenticated() {
			return nil
		}
		return m.showStandaloneView(viewManageRooms)
	default:
		return nil
	}
}

func (m *model) showStandaloneView(view viewState) tea.Cmd {
	if m.closed {
		return nil
	}
	if view == viewAuth && m.isAuthenticated() {
		view = viewMainMenu
	}

	m.closeRoomMembership()
	m.view = view
	m.ui = m.newUI(view)
	return m.ui.Init()
}

func (m model) isAuthenticated() bool {
	return m.authenticatedUser != nil
}

func (m model) currentChatUserID() (chat.UserID, bool) {
	if m.authenticatedUser == nil {
		return "", false
	}
	return chat.UserID(m.authenticatedUser.ID), true
}

func (m *model) showMyChats() tea.Cmd {
	userID, ok := m.currentChatUserID()
	if !ok || m.chatService == nil {
		return nil
	}
	rooms, err := m.chatService.ListRoomsForUser(m.ctx, userID)
	if err != nil {
		return nil
	}
	m.roomList = rooms
	return m.showStandaloneView(viewMyChats)
}

func (m *model) enterSelectedRoom(roomID string) tea.Cmd {
	if m.view != viewMyChats || m.chatService == nil || !m.isAuthenticated() {
		return nil
	}
	selected, ok := m.findRoomSummary(chat.RoomID(roomID))
	if !ok {
		return nil
	}
	subscription, err := m.chatService.JoinRoom(m.ctx, selected.ID, m.member)
	if err != nil {
		return nil
	}
	return m.enterRoomView(selected, subscription)
}

func (m *model) createRoom(title string) tea.Cmd {
	if m.view != viewManageRooms || m.chatService == nil {
		return nil
	}
	userID, ok := m.currentChatUserID()
	if !ok {
		return nil
	}
	summary, err := m.chatService.CreateRoom(m.ctx, userID, title)
	if err != nil {
		return m.showCreateRoomError(createRoomErrorMessage(err))
	}
	subscription, err := m.chatService.JoinRoom(m.ctx, summary.ID, m.member)
	if err != nil {
		return m.showStandaloneView(viewMainMenu)
	}
	return m.enterRoomView(summary, subscription)
}

func (m *model) showCreateRoomError(message string) tea.Cmd {
	var cmd tea.Cmd
	m.ui, cmd = m.ui.Update(tui.CreateRoomFailed{Message: message})
	return cmd
}

func createRoomErrorMessage(err error) string {
	switch {
	case errors.Is(err, chat.ErrInvalidRoomTitle):
		return "Room title must be 1-64 characters."
	default:
		return "Could not create room."
	}
}

func (m model) findRoomSummary(roomID chat.RoomID) (chat.RoomSummary, bool) {
	for _, room := range m.roomList {
		if room.ID == roomID {
			return room, true
		}
	}
	return chat.RoomSummary{}, false
}

func (m *model) enterRoomView(room chat.RoomSummary, subscription *chat.Subscription) tea.Cmd {
	if m.closed || subscription == nil {
		return nil
	}

	m.closeRoomMembership()
	m.activeRoomID = room.ID
	m.activeRoomTitle = room.Title
	m.activeRoomJoinCode = room.JoinCode
	m.subscription = subscription
	m.view = viewChat
	m.ui = m.newUI(viewChat)

	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m *model) leaveRoomView() tea.Cmd {
	if !m.inRoomView() {
		return nil
	}
	return m.showStandaloneView(viewMainMenu)
}

func (m *model) closeRoomMembership() {
	if m.subscription != nil {
		m.subscription.Close()
		m.subscription = nil
	}
	m.activeRoomID = ""
	m.activeRoomTitle = ""
	m.activeRoomJoinCode = ""
}

func (m model) inRoomView() bool {
	return m.view == viewChat && m.subscription != nil && m.activeRoomID != ""
}

func (m model) newUI(view viewState) tea.Model {
	config := tui.Config{Width: m.width, Height: m.height}
	switch view {
	case viewWelcome:
		return tui.NewWelcome(config)
	case viewAuth:
		return tui.NewAuth(config)
	case viewLinkSSHKey:
		config.SSHKeyFingerprint = m.connectionSSHKeyFingerprint
		return tui.NewLinkSSHKey(config)
	case viewMainMenu:
		return tui.NewMainMenu(config)
	case viewMyChats:
		config.Rooms = roomListItems(m.roomList)
		return tui.NewMyChats(config)
	case viewManageRooms:
		return tui.NewManageRooms(config)
	case viewChat:
		config.RoomTitle = m.activeRoomTitle
		config.RoomJoinCode = chat.FormatJoinCode(m.activeRoomJoinCode)
		return tui.NewRoomView(config)
	default:
		return tui.NewWelcome(config)
	}
}

func roomListItems(rooms []chat.RoomSummary) []tui.RoomListItem {
	items := make([]tui.RoomListItem, 0, len(rooms))
	for _, room := range rooms {
		items = append(items, tui.RoomListItem{ID: string(room.ID), Title: room.Title})
	}
	return items
}
