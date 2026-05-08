package session

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/auth"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

type viewState int

const (
	viewWelcome viewState = iota
	viewAuth
	viewLinkSSHKey
	viewMainMenu
	viewMyChats
	viewManageRooms
	viewChat
)

type ChatService interface {
	CreateRoom(ctx context.Context, creator chat.UserID, title string) (chat.RoomSummary, error)
	ListRoomsForUser(ctx context.Context, userID chat.UserID) ([]chat.RoomSummary, error)
	JoinRoom(ctx context.Context, roomID chat.RoomID, member chat.Member) (*chat.Subscription, error)
	Post(ctx context.Context, roomID chat.RoomID, author chat.Member, body string) (chat.Message, error)
}

// Config wires one Bubble Tea session to chat/auth services. Member is replaced
// with the authenticated user identity before any room action is allowed.
type Config struct {
	Width             int
	Height            int
	Context           context.Context
	ChatService       ChatService
	AuthService       auth.Service
	SSHPublicKey      string
	SSHKeyFingerprint string
	Member            chat.Member
}

func New(config Config) tea.Model {
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
	}

	m := model{
		ctx:                         ctx,
		chatService:                 config.ChatService,
		authService:                 config.AuthService,
		connectionSSHPublicKey:      config.SSHPublicKey,
		connectionSSHKeyFingerprint: config.SSHKeyFingerprint,
		member:                      config.Member,
		width:                       config.Width,
		height:                      config.Height,
		view:                        viewWelcome,
		ui: tui.NewWelcome(tui.Config{
			Width:  config.Width,
			Height: config.Height,
		}),
	}
	m.authenticateWithConnectionSSHKey()
	return m
}

type model struct {
	ctx                         context.Context
	chatService                 ChatService
	authService                 auth.Service
	authenticatedUser           *auth.User
	connectionSSHPublicKey      string
	connectionSSHKeyFingerprint string
	member                      chat.Member // authenticated participant after login/key auth
	width                       int
	height                      int
	roomList                    []chat.RoomSummary
	activeRoomID                chat.RoomID
	activeRoomTitle             string
	activeRoomJoinCode          string
	subscription                *chat.Subscription
	view                        viewState
	closed                      bool
	ui                          tea.Model
}

type roomEvent struct {
	event chat.Event
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// The session model owns cross-view state: room membership, current terminal
	// size, and which TUI model is active. View-specific messages are handled here
	// only when they affect that shared state; everything else is delegated below.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tui.QuitRequested:
		m.close()
		return m, tea.Quit
	case tui.ContinueRequested, tui.BackRequested, tui.AuthSubmissionRequested, tui.LinkSSHKeySelectionRequested, tui.MainMenuSelectionRequested, tui.RoomSelected, tui.CreateRoomRequested, tui.LeaveRequested:
		return m, m.applyFlowIntent(msg)
	case tui.SendRequested:
		return m, m.postMessage(msg.Body)
	case roomEvent:
		if !m.inRoomView() {
			return m, nil
		}
		display, ok := m.displayMessage(msg.event)
		if !ok {
			return m, m.waitForRoomEvent()
		}
		var cmd tea.Cmd
		m.ui, cmd = m.ui.Update(display)
		return m, tea.Batch(cmd, m.waitForRoomEvent())
	}

	var cmd tea.Cmd
	m.ui, cmd = m.ui.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return m.ui.View()
}

func (m model) postMessage(body string) tea.Cmd {
	return func() tea.Msg {
		if m.closed || !m.inRoomView() || m.chatService == nil || m.activeRoomID == "" || !m.isAuthenticated() {
			return nil
		}
		_, _ = m.chatService.Post(m.ctx, m.activeRoomID, m.member, body)
		return nil
	}
}

func (m model) waitForRoomEvent() tea.Cmd {
	return func() tea.Msg {
		if m.subscription == nil {
			return nil
		}

		select {
		case <-m.ctx.Done():
			m.subscription.Close()
			return nil
		default:
		}

		select {
		case <-m.ctx.Done():
			m.subscription.Close()
			return nil
		case event, ok := <-m.subscription.Events():
			if !ok {
				return nil
			}
			return roomEvent{event: event}
		}
	}
}

func (m *model) close() {
	if m.closed {
		return
	}
	m.closed = true
	m.closeRoomMembership()
}
