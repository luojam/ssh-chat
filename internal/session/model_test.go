package session

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

func TestSendRequestedPostsToRoomAndRoomEventUpdatesTUI(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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

func TestRoomEventMarksOtherMemberMessageAsNotMine(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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

func TestRoomEventUsesMemberIDForMine(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
	room := chat.NewRoom()
	user := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "user-1", Name: "user"},
	})
	user = enterChat(t, user)
	sara := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "sara-1", Name: "sara"},
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
	room := chat.NewRoom()
	if _, err := room.Post(chat.Member{ID: "user-1", Name: "user"}, "before join"); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	sara := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "sara-1", Name: "sara"},
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
	room := chat.NewRoom()
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	sara := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "sara-1", Name: "sara"},
	})
	_ = enterChat(t, sara)

	event := nextRoomEvent(t, m.subscription, chat.MemberJoined)
	next, _ := m.Update(roomEvent{event: event})
	m = assertModel(t, next)

	view := m.View()
	if !strings.Contains(view.Content, systemAuthor) || !strings.Contains(view.Content, "sara joined") {
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
	if !strings.Contains(view.Content, systemAuthor) || !strings.Contains(view.Content, "sara left") {
		t.Fatalf("leave should render as system message, got %q", view.Content)
	}
}

func TestWelcomeContinueShowsMainMenu(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
	})

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("ContinueRequested from welcome should initialize main menu")
	}
	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	if m.subscription != nil {
		t.Fatal("main menu should not join the room")
	}
}

func TestBackFromMainMenuReturnsToWelcome(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 12,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
	if m.subscription != nil {
		t.Fatal("welcome should not join the room")
	}
}

func TestContinueFromMainMenuDoesNotJoinRoom(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 12,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
		Width:  40,
		Height: 12,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
		Width:  40,
		Height: 12,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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

func TestContinueFromMyChatsStartsChat(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 12,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterMainMenu(t, m)
	next, _ := m.Update(tui.MainMenuSelectionRequested{Action: tui.MainMenuActionMyChats})
	m = assertModel(t, next)

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("ContinueRequested from My Chats should start the Room View")
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
	room := chat.NewRoom()
	user := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "user-1", Name: "user"},
	})
	user = enterChat(t, user)
	oldSubscription := user.subscription

	sara := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{ID: "sara-1", Name: "sara"},
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
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{ID: "user-1", Name: "user"},
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
		Width:   40,
		Height:  8,
		Context: ctx,
		Room:    chat.NewRoom(),
		Member:  chat.Member{ID: "user-1", Name: "user"},
	})
	m = enterChat(t, m)

	cancel()
	if msg := m.waitForRoomEvent()(); msg != nil {
		t.Fatalf("wait command returned %T, want nil", msg)
	}
	assertSubscriptionClosed(t, m.subscription)
}

func newModel(t *testing.T, config Config) model {
	t.Helper()

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
	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("continue from My Chats should return Room View startup command")
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

func enterMainMenu(t *testing.T, m model) model {
	t.Helper()

	next, cmd := m.Update(tui.ContinueRequested{})
	if cmd == nil {
		t.Fatal("continue from welcome should return main menu startup command")
	}

	m = assertModel(t, next)
	if m.view != viewMainMenu {
		t.Fatalf("view = %d, want viewMainMenu", m.view)
	}
	return m
}
