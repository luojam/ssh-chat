package session

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/auth"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

func TestSendRequestedPostsToRoomAndRoomEventUpdatesTUI(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	next, cmd := m.Update(tui.SendRequested{Body: "hello"})
	if cmd == nil {
		t.Fatal("SendRequested should return a command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("post command returned %T, want nil", msg)
	}

	m = assertModel(t, next)
	event := nextRoomEvent(t, m.subscription, chat.MessagePosted)
	next, _ = m.Update(roomEvent{event: event})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, "you") || !strings.Contains(view.Content, "hello") {
		t.Fatalf("view should render accepted room message, got %q", view.Content)
	}
}

func TestRoomEventMarksOtherMemberMessageAsRemote(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	next, _ := m.Update(roomEvent{
		event: chat.Event{
			Kind: chat.MessagePosted,
			Message: chat.Message{
				Author: chat.Member{ID: "sara-1", Name: "sara"},
				Body:   "hello",
			},
		},
	})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, "sara") || !strings.Contains(view.Content, "hello") {
		t.Fatalf("view should render other member message, got %q", view.Content)
	}
	if strings.Contains(view.Content, "you") {
		t.Fatalf("other member message should not render as local author, got %q", view.Content)
	}
}

func TestRoomEventUsesUnknownAuthorForEmptyMemberName(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	next, _ := m.Update(roomEvent{
		event: chat.Event{
			Kind: chat.MessagePosted,
			Message: chat.Message{
				Author: chat.Member{ID: "anon-1", Name: ""},
				Body:   "nameless",
			},
		},
	})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, displayUnknownAuthor) || !strings.Contains(view.Content, "nameless") {
		t.Fatalf("empty author should render as unknown, got %q", view.Content)
	}
}

func TestRoomEventUsesMemberIDForLocalAuthor(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	next, _ := m.Update(roomEvent{
		event: chat.Event{
			Kind: chat.MessagePosted,
			Message: chat.Message{
				Author: chat.Member{ID: "user-2", Name: "user"},
				Body:   "same display name",
			},
		},
	})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, "user") || !strings.Contains(view.Content, "same display name") {
		t.Fatalf("view should render message from duplicate-name member, got %q", view.Content)
	}
	if strings.Contains(view.Content, "you") {
		t.Fatalf("duplicate display name should not render as local author, got %q", view.Content)
	}
}

func TestRoomBroadcastReachesMultipleSessions(t *testing.T) {
	chatService := newTestChatService()
	user := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	user = enterChat(t, user)
	sara := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "sara-1", Name: "sara"},
	})
	sara = enterChat(t, sara)

	_, cmd := user.Update(tui.SendRequested{Body: "hello"})
	if cmd == nil {
		t.Fatal("SendRequested should return a command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("post command returned %T, want nil", msg)
	}

	userEvent := nextRoomEvent(t, user.subscription, chat.MessagePosted)
	saraEvent := nextRoomEvent(t, sara.subscription, chat.MessagePosted)

	next, _ := user.Update(roomEvent{event: userEvent})
	user = assertModel(t, next)
	next, _ = sara.Update(roomEvent{event: saraEvent})
	sara = assertModel(t, next)

	userView := user.View()
	if !strings.Contains(userView.Content, "you") || !strings.Contains(userView.Content, "hello") {
		t.Fatalf("sender view should render local message, got %q", userView.Content)
	}

	saraView := sara.View()
	if !strings.Contains(saraView.Content, "user") || !strings.Contains(saraView.Content, "hello") {
		t.Fatalf("receiver view should render sender name, got %q", saraView.Content)
	}
	if strings.Contains(saraView.Content, "you") {
		t.Fatalf("receiver view should not mark sender message as local, got %q", saraView.Content)
	}
}

func TestNewSessionCanRenderRoomHistory(t *testing.T) {
	chatService := newTestChatService()
	if _, err := chatService.Post(context.Background(), testRoomID, chat.Member{ID: "user-1", Name: "user"}, "before join"); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	sara := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "sara-1", Name: "sara"},
	})
	sara = enterChat(t, sara)

	event := <-sara.subscription.Events()
	next, _ := sara.Update(roomEvent{event: event})
	sara = assertModel(t, next)

	view := sara.View()
	if !strings.Contains(view.Content, "user") || !strings.Contains(view.Content, "before join") {
		t.Fatalf("new session should render room history, got %q", view.Content)
	}
}

func TestMemberEventsRenderAsSystemMessages(t *testing.T) {
	chatService := newTestChatService()
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	sara := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "sara-1", Name: "sara"},
	})
	_ = enterChat(t, sara)

	event := nextRoomEvent(t, m.subscription, chat.MemberJoined)
	next, _ := m.Update(roomEvent{event: event})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, displaySystemAuthor) || !strings.Contains(view.Content, "sara joined") {
		t.Fatalf("join should render as system message, got %q", view.Content)
	}

	next, _ = m.Update(roomEvent{
		event: chat.Event{
			Kind:   chat.MemberLeft,
			Member: chat.Member{ID: "sara-1", Name: "sara"},
		},
	})
	m = assertModel(t, next)

	view = m.View()
	if !strings.Contains(view.Content, displaySystemAuthor) || !strings.Contains(view.Content, "sara left") {
		t.Fatalf("leave should render as system message, got %q", view.Content)
	}
}

func TestWelcomeContinueShowsAuth(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("ContinueRequested from welcome should initialize auth")
	}
	m = assertModel(t, next)
	if m.view != viewAuth {
		t.Fatalf("view = %d, want viewAuth", m.view)
	}
	if m.subscription != nil {
		t.Fatal("auth should not join the room")
	}
}

func TestAuthSubmissionShowsMainMenu(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterAuth(t, m)

	next, cmd := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "user", Password: "password"})
	if cmd == nil {
		t.Fatal("AuthSubmissionRequested from auth should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	if m.subscription != nil {
		t.Fatal("main menu should not join the room")
	}
}

func TestLinkedSSHKeyAtStartupAuthenticatesUserAndWelcomeContinueSkipsAuth(t *testing.T) {
	authService := newKeyAuthService()
	authService.linked["SHA256:abc"] = auth.User{ID: "user-auth", Username: "alice"}
	m := newModel(t, Config{
		Width:             40,
		Height:            8,
		ChatService:       newTestChatService(),
		AuthService:       authService,
		SSHPublicKey:      "ssh-ed25519 AAAA",
		SSHKeyFingerprint: "SHA256:abc",
		Member:            chat.Member{ID: "session-1", Name: "ssh-user"},
	})

	if m.view != viewWelcome {
		t.Fatalf("initial view = %d, want viewWelcome", m.view)
	}
	if m.authenticatedUser == nil || m.authenticatedUser.Username != "alice" {
		t.Fatalf("authenticated user = %+v, want alice", m.authenticatedUser)
	}
	if m.member.ID != "user-auth" || m.member.Name != "alice" {
		t.Fatalf("member = %+v, want key-authenticated member", m.member)
	}

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("key-authenticated welcome continue should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
}

func TestPasswordAuthWithUnlinkedSSHKeyShowsLinkPromptAndYesLinks(t *testing.T) {
	authService := newKeyAuthService()
	m := newModel(t, Config{
		Width:             40,
		Height:            12,
		ChatService:       newTestChatService(),
		AuthService:       authService,
		SSHPublicKey:      "ssh-ed25519 AAAA",
		SSHKeyFingerprint: "SHA256:abc",
		Member:            chat.Member{ID: "session-1", Name: "ssh-user"},
	})
	m = enterAuth(t, m)

	next, cmd := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "alice", Password: "password"})
	if cmd == nil {
		t.Fatal("auth with unlinked key should initialize link prompt")
	}
	m = assertModel(t, next)
	if m.view != viewLinkSSHKey {
		t.Fatalf("view = %d, want viewLinkSSHKey", m.view)
	}

	next, cmd = m.Update(tui.LinkSSHKeySelectionRequested{Link: true})
	if cmd == nil {
		t.Fatal("yes should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	linked, ok := authService.linked["SHA256:abc"]
	if !ok || linked.Username != "alice" {
		t.Fatalf("linked key user = %+v, present %v; want alice", linked, ok)
	}
}

func TestPasswordAuthWithUnlinkedSSHKeyNoSkipsLinking(t *testing.T) {
	authService := newKeyAuthService()
	m := newModel(t, Config{
		Width:             40,
		Height:            12,
		ChatService:       newTestChatService(),
		AuthService:       authService,
		SSHPublicKey:      "ssh-ed25519 AAAA",
		SSHKeyFingerprint: "SHA256:abc",
		Member:            chat.Member{ID: "session-1", Name: "ssh-user"},
	})
	m = enterAuth(t, m)
	next, _ := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "alice", Password: "password"})
	m = assertModel(t, next)

	next, cmd := m.Update(tui.LinkSSHKeySelectionRequested{Link: false})
	if cmd == nil {
		t.Fatal("no should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	if _, ok := authService.linked["SHA256:abc"]; ok {
		t.Fatal("no should not link key")
	}
}

func TestPasswordAuthWithoutSSHKeyShowsMainMenuDirectly(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		AuthService: newKeyAuthService(),
		Member:      chat.Member{ID: "session-1", Name: "ssh-user"},
	})
	m = enterAuth(t, m)

	next, cmd := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "alice", Password: "password"})
	if cmd == nil {
		t.Fatal("auth without ssh key should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
}

func TestFailedAuthSubmissionStaysOnAuth(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      14,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "session-1", Name: "ssh-user"},
		AuthService: failingAuthService{err: auth.ErrInvalidCredentials},
	})
	m = enterAuth(t, m)

	next, cmd := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "user", Password: "wrong"})
	m = assertModel(t, next)
	if cmd != nil {
		t.Fatal("failed auth should not navigate or return startup command")
	}
	if m.view != viewAuth {
		t.Fatalf("view = %d, want viewAuth", m.view)
	}
	if !strings.Contains(m.View().Content, "Invalid username") {
		t.Fatalf("auth failure should render error, got %q", m.View().Content)
	}
	if m.subscription != nil {
		t.Fatal("failed auth should not join the room")
	}
}

func TestAuthSubmissionStoresAuthenticatedUserAndMember(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "session-1", Name: "ssh-user"},
	})
	m = enterAuth(t, m)

	next, _ := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: "user", Password: "password"})
	m = assertModel(t, next)
	if m.authenticatedUser == nil || m.authenticatedUser.Username != "user" {
		t.Fatalf("authenticated user = %+v, want username user", m.authenticatedUser)
	}
	if m.member.ID != "user-auth" || m.member.Name != "user" {
		t.Fatalf("member = %+v, want authenticated member", m.member)
	}
}

func TestSignupSubmissionUsesConfirmPassword(t *testing.T) {
	authService := recordingAuthService{}
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "session-1", Name: "ssh-user"},
		AuthService: &authService,
	})
	m = enterAuth(t, m)

	_, _ = m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeSignup, Username: "user", Password: "password", ConfirmPassword: "password"})
	if !authService.signupCalled {
		t.Fatal("signup submission should call Signup")
	}
	if authService.confirmPassword != "password" {
		t.Fatalf("confirm password = %q, want password", authService.confirmPassword)
	}
}

func TestBackFromAuthReturnsToWelcome(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterAuth(t, m)

	next, cmd := m.Update(tui.BackRequested{})
	if cmd == nil {
		t.Fatal("BackRequested from auth should initialize welcome")
	}
	m = assertModel(t, next)
	if m.view != viewWelcome {
		t.Fatalf("view = %d, want viewWelcome", m.view)
	}
	if m.subscription != nil {
		t.Fatal("welcome should not join the room")
	}
}

func TestBackFromMainMenuReturnsToWelcomeAndKeepsAuth(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)

	next, cmd := m.Update(tui.BackRequested{})
	if cmd == nil {
		t.Fatal("BackRequested from main menu should initialize welcome")
	}
	m = assertModel(t, next)
	if m.view != viewWelcome {
		t.Fatalf("view = %d, want viewWelcome", m.view)
	}
	if m.authenticatedUser == nil {
		t.Fatal("back to welcome should keep authenticated user")
	}
	if m.subscription != nil {
		t.Fatal("welcome should not join the room")
	}
}

func TestAuthenticatedWelcomeContinueSkipsAuth(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)
	next, _ := m.Update(tui.BackRequested{})
	m = assertModel(t, next)

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("ContinueRequested from authenticated welcome should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
}

func TestAuthenticatedSessionCannotReturnToAuth(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)

	cmd := m.showStandaloneView(viewAuth)
	if cmd == nil {
		t.Fatal("guarded auth navigation should initialize replacement view")
	}
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
}

func TestContinueFromMainMenuDoesNotJoinRoom(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd != nil {
		t.Fatal("ContinueRequested from main menu should not navigate")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	if m.subscription != nil {
		t.Fatal("main menu continue should not join the room")
	}
}

func TestMainMenuMyChatsSelectionShowsMyChats(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)

	next, cmd := m.Update(tui.MainMenuSelectionRequested{Action: tui.MainMenuActionMyChats})
	if cmd == nil {
		t.Fatal("My Chats selection should initialize My Chats view")
	}
	m = assertModel(t, next)
	if m.view != viewMyChats {
		t.Fatalf("view = %d, want viewMyChats", m.view)
	}
	if m.subscription != nil {
		t.Fatal("My Chats view should not join the room")
	}
}

func TestBackFromMyChatsReturnsToMainMenu(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)
	next, _ := m.Update(tui.MainMenuSelectionRequested{Action: tui.MainMenuActionMyChats})
	m = assertModel(t, next)

	next, cmd := m.Update(tui.BackRequested{})
	if cmd == nil {
		t.Fatal("BackRequested from My Chats should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
}

func TestContinueFromMyChatsDoesNotStartChat(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMyChats(t, m)

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd != nil {
		t.Fatal("ContinueRequested from My Chats should not navigate")
	}
	m = assertModel(t, next)
	if m.view != viewMyChats {
		t.Fatalf("view = %d, want viewMyChats", m.view)
	}
	if m.subscription != nil {
		t.Fatal("ContinueRequested from My Chats should not join the room")
	}
}

func TestUnknownSelectedRoomFromMyChatsDoesNotStartChat(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMyChats(t, m)

	next, cmd := m.Update(tui.RoomSelected{RoomID: "unknown-room"})
	if cmd != nil {
		t.Fatal("unknown RoomSelected should not navigate")
	}
	m = assertModel(t, next)
	if m.view != viewMyChats {
		t.Fatalf("view = %d, want viewMyChats", m.view)
	}
	if m.subscription != nil {
		t.Fatal("unknown RoomSelected should not join the room")
	}
}

func TestSelectedRoomFromMyChatsStartsChat(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      12,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMyChats(t, m)

	next, cmd := m.Update(tui.RoomSelected{RoomID: string(testRoomID)})
	if cmd == nil {
		t.Fatal("RoomSelected from My Chats should start the Room View")
	}
	m = assertModel(t, next)
	if m.view != viewChat {
		t.Fatalf("view = %d, want viewChat", m.view)
	}
	if m.subscription == nil {
		t.Fatal("model should create a Room subscription")
	}
}

func TestLeaveRequestedReturnsToMainMenuAndClosesSubscription(t *testing.T) {
	chatService := newTestChatService()
	user := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	user = enterChat(t, user)
	oldSubscription := user.subscription

	sara := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: chatService,
		Member:      chat.Member{ID: "sara-1", Name: "sara"},
	})
	sara = enterChat(t, sara)

	next, cmd := user.Update(tui.LeaveRequested{})
	if cmd == nil {
		t.Fatal("LeaveRequested should initialize the main menu view")
	}
	user = assertModel(t, next)

	if user.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", user.view)
	}
	if user.subscription != nil {
		t.Fatal("subscription should be cleared after leaving chat")
	}
	assertSubscriptionClosed(t, oldSubscription)

	left := nextRoomEvent(t, sara.subscription, chat.MemberLeft)
	if got := left.Member.Name; got != "user" {
		t.Fatalf("left member = %q, want user", got)
	}
}

func TestStaleRoomEventIgnoredAfterLeavingChat(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)
	next, _ := m.Update(tui.LeaveRequested{})
	m = assertModel(t, next)

	next, cmd := m.Update(roomEvent{event: chat.Event{
		Kind: chat.MessagePosted,
		Message: chat.Message{
			Author: chat.Member{ID: "sara-1", Name: "sara"},
			Body:   "stale",
		},
	}})
	m = assertModel(t, next)
	if cmd != nil {
		t.Fatal("stale room event after leaving should not continue room wait loop")
	}
	if strings.Contains(m.View().Content, "stale") {
		t.Fatalf("stale room event should not update main menu view, got %q", m.View().Content)
	}
}

func TestQuitRequestedClosesSubscriptionBeforeQuit(t *testing.T) {
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)
	oldSubscription := m.subscription

	next, cmd := m.Update(tui.QuitRequested{})
	if cmd == nil {
		t.Fatal("QuitRequested should return a quit command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("quit command returned nil")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("quit command returned %T, want tea.QuitMsg", msg)
	}

	m = assertModel(t, next)
	if !m.closed {
		t.Fatal("session should be marked closed")
	}
	if m.subscription != nil {
		t.Fatal("subscription should be cleared after quit")
	}
	assertSubscriptionClosed(t, oldSubscription)
}

func TestWaitForRoomEventClosesSubscriptionOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := newModel(t, Config{
		Width:       40,
		Height:      8,
		Context:     ctx,
		ChatService: newTestChatService(),
		Member:      chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	cancel()
	if msg := m.waitForRoomEvent()(); msg != nil {
		t.Fatalf("wait command returned %T, want nil", msg)
	}
	assertSubscriptionClosed(t, m.subscription)
}

const testRoomID chat.RoomID = "town-square"

func newTestChatService() *chat.Service {
	store := &testChatStore{
		rooms: map[chat.RoomID]chat.StoredRoom{
			testRoomID: {ID: testRoomID, Title: "Town Square", JoinCode: "7KQ9M2XP", CreatedBy: "system"},
		},
		messages:    map[chat.RoomID][]chat.Message{},
		nextMessage: 1,
	}
	return chat.NewService(store)
}

type testChatStore struct {
	mu          sync.Mutex
	rooms       map[chat.RoomID]chat.StoredRoom
	messages    map[chat.RoomID][]chat.Message
	nextMessage chat.MessageID
}

func (s *testChatStore) CreateRoom(_ context.Context, room chat.StoredRoom, owner chat.UserID, role chat.RoomRole) (chat.RoomSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rooms[room.ID] = room
	return chat.RoomSummary{ID: room.ID, Title: room.Title, JoinCode: room.JoinCode, Role: role, CreatedAt: room.CreatedAt}, nil
}

func (s *testChatStore) ListRoomsForUser(_ context.Context, userID chat.UserID) ([]chat.RoomSummary, error) {
	if userID == "" {
		return nil, chat.ErrNotRoomMember
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rooms := make([]chat.RoomSummary, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, chat.RoomSummary{ID: room.ID, Title: room.Title, JoinCode: room.JoinCode, Role: chat.RoomRoleOwner, CreatedAt: room.CreatedAt})
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })
	return rooms, nil
}

func (s *testChatStore) IsRoomMember(_ context.Context, roomID chat.RoomID, userID chat.UserID) (bool, error) {
	if userID == "" {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.rooms[roomID]
	return ok, nil
}

func (s *testChatStore) StoreMessage(_ context.Context, message chat.Message) (chat.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	message.ID = s.nextMessage
	s.nextMessage++
	s.messages[message.RoomID] = append(s.messages[message.RoomID], message)
	return message, nil
}

func (s *testChatStore) RecentMessages(_ context.Context, roomID chat.RoomID, limit int) ([]chat.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages := s.messages[roomID]
	start := len(messages) - limit
	if start < 0 {
		start = 0
	}
	recent := make([]chat.Message, len(messages[start:]))
	copy(recent, messages[start:])
	return recent, nil
}

func newModel(t *testing.T, config Config) model {
	t.Helper()

	if config.AuthService == nil {
		config.AuthService = successfulAuthService{}
	}

	m, ok := New(config).(model)
	if !ok {
		t.Fatalf("New returned %T, want model", m)
	}
	return m
}

func assertModel(t *testing.T, tm tea.Model) model {
	t.Helper()

	m, ok := tm.(model)
	if !ok {
		t.Fatalf("updated model has type %T, want model", tm)
	}
	return m
}

func nextRoomEvent(t *testing.T, subscription *chat.Subscription, kind chat.EventKind) chat.Event {
	t.Helper()

	for {
		select {
		case event, ok := <-subscription.Events():
			if !ok {
				t.Fatalf("subscription closed before event kind %d", kind)
			}
			if event.Kind == kind {
				return event
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timed out waiting for event kind %d", kind)
		}
	}
}

func assertSubscriptionClosed(t *testing.T, subscription *chat.Subscription) {
	t.Helper()

	for {
		select {
		case _, ok := <-subscription.Events():
			if !ok {
				return
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("subscription events should be closed")
		}
	}
}

func enterChat(t *testing.T, m model) model {
	t.Helper()

	m = enterMyChats(t, m)
	next, cmd := m.Update(tui.RoomSelected{RoomID: string(testRoomID)})
	if cmd == nil {
		t.Fatal("RoomSelected from My Chats should return Room View startup command")
	}

	m = assertModel(t, next)
	if m.view != viewChat {
		t.Fatalf("view = %d, want viewChat", m.view)
	}
	if m.subscription == nil {
		t.Fatal("model should set subscription after My Chats continue")
	}

	return m
}

func enterMyChats(t *testing.T, m model) model {
	t.Helper()

	m = enterMainMenu(t, m)
	next, cmd := m.Update(tui.MainMenuSelectionRequested{Action: tui.MainMenuActionMyChats})
	if cmd == nil {
		t.Fatal("My Chats selection should return My Chats startup command")
	}

	m = assertModel(t, next)
	if m.view != viewMyChats {
		t.Fatalf("view = %d, want viewMyChats", m.view)
	}
	return m
}

func enterAuth(t *testing.T, m model) model {
	t.Helper()

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("continue from welcome should return auth startup command")
	}

	m = assertModel(t, next)
	if m.view != viewAuth {
		t.Fatalf("view = %d, want viewAuth", m.view)
	}
	return m
}

func enterMainMenu(t *testing.T, m model) model {
	t.Helper()

	m = enterAuth(t, m)
	next, cmd := m.Update(tui.AuthSubmissionRequested{Mode: tui.AuthModeLogin, Username: m.member.Name, Password: "password"})
	if cmd == nil {
		t.Fatal("auth submission should return main menu startup command")
	}

	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	return m
}

type successfulAuthService struct{}

func (successfulAuthService) Signup(_ context.Context, username, _, _ string) (auth.User, error) {
	return auth.User{ID: username + "-auth", Username: username}, nil
}

func (successfulAuthService) Login(_ context.Context, username, _ string) (auth.User, error) {
	return auth.User{ID: username + "-auth", Username: username}, nil
}

func (successfulAuthService) FindUserBySSHKeyFingerprint(context.Context, string) (auth.User, error) {
	return auth.User{}, auth.ErrSSHKeyNotFound
}

func (successfulAuthService) LinkSSHKey(context.Context, auth.User, string, string) error {
	return nil
}

type failingAuthService struct {
	err error
}

func (s failingAuthService) Signup(context.Context, string, string, string) (auth.User, error) {
	return auth.User{}, s.err
}

func (s failingAuthService) Login(context.Context, string, string) (auth.User, error) {
	return auth.User{}, s.err
}

func (s failingAuthService) FindUserBySSHKeyFingerprint(context.Context, string) (auth.User, error) {
	return auth.User{}, s.err
}

func (s failingAuthService) LinkSSHKey(context.Context, auth.User, string, string) error {
	return s.err
}

type recordingAuthService struct {
	signupCalled    bool
	confirmPassword string
}

func (s *recordingAuthService) Signup(_ context.Context, username, _, confirmPassword string) (auth.User, error) {
	s.signupCalled = true
	s.confirmPassword = confirmPassword
	return auth.User{ID: "auth-user", Username: username}, nil
}

func (s *recordingAuthService) Login(context.Context, string, string) (auth.User, error) {
	return auth.User{}, errors.New("unexpected login")
}

func (s *recordingAuthService) FindUserBySSHKeyFingerprint(context.Context, string) (auth.User, error) {
	return auth.User{}, auth.ErrSSHKeyNotFound
}

func (s *recordingAuthService) LinkSSHKey(context.Context, auth.User, string, string) error {
	return nil
}

type keyAuthService struct {
	linked map[string]auth.User
}

func newKeyAuthService() *keyAuthService {
	return &keyAuthService{linked: map[string]auth.User{}}
}

func (s *keyAuthService) Signup(_ context.Context, username, _, _ string) (auth.User, error) {
	return auth.User{ID: username + "-auth", Username: username}, nil
}

func (s *keyAuthService) Login(_ context.Context, username, _ string) (auth.User, error) {
	return auth.User{ID: username + "-auth", Username: username}, nil
}

func (s *keyAuthService) FindUserBySSHKeyFingerprint(_ context.Context, fingerprint string) (auth.User, error) {
	user, ok := s.linked[fingerprint]
	if !ok {
		return auth.User{}, auth.ErrSSHKeyNotFound
	}
	return user, nil
}

func (s *keyAuthService) LinkSSHKey(_ context.Context, user auth.User, _, fingerprint string) error {
	linked, ok := s.linked[fingerprint]
	if !ok {
		s.linked[fingerprint] = user
		return nil
	}
	if linked.ID == user.ID {
		return nil
	}
	return auth.ErrSSHKeyAlreadyLinked
}
