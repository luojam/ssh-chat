package session

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

func TestSendRequestedPostsToRoomAndRoomEventUpdatesTUI(t *testing.T) {
	m := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   chat.NewRoom(),
		Member: chat.Member{Name: "jami"},
	})

	next, cmd := m.Update(tui.SendRequested{Body: "hello"})
	if cmd == nil {
		t.Fatal("SendRequested should return a command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("post command returned %T, want nil", msg)
	}

	m = assertModel(t, next)
	event := <-m.subscription.Events()
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
		Member: chat.Member{Name: "jami"},
	})

	next, _ := m.Update(roomEvent{
		event: chat.Event{
			Message: chat.Message{
				Author: chat.Member{Name: "sara"},
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

func TestRoomBroadcastReachesMultipleSessions(t *testing.T) {
	room := chat.NewRoom()
	jami := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{Name: "jami"},
	})
	sara := newModel(t, Config{
		Width:  40,
		Height: 8,
		Room:   room,
		Member: chat.Member{Name: "sara"},
	})

	_, cmd := jami.Update(tui.SendRequested{Body: "hello"})
	if cmd == nil {
		t.Fatal("SendRequested should return a command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("post command returned %T, want nil", msg)
	}

	jamiEvent := <-jami.subscription.Events()
	saraEvent := <-sara.subscription.Events()

	next, _ := jami.Update(roomEvent{event: jamiEvent})
	jami = assertModel(t, next)
	next, _ = sara.Update(roomEvent{event: saraEvent})
	sara = assertModel(t, next)

	jamiView := jami.View()
	if !strings.Contains(jamiView.Content, "you") || !strings.Contains(jamiView.Content, "hello") {
		t.Fatalf("sender view should render local message, got %q", jamiView.Content)
	}

	saraView := sara.View()
	if !strings.Contains(saraView.Content, "jami") || !strings.Contains(saraView.Content, "hello") {
		t.Fatalf("receiver view should render sender name, got %q", saraView.Content)
	}
	if strings.Contains(saraView.Content, "you") {
		t.Fatalf("receiver view should not mark sender message as local, got %q", saraView.Content)
	}
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
