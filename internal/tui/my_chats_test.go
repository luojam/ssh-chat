package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestMyChatsViewShowsRoomList(t *testing.T) {
	m := newMyChatsModel(t, Config{Width: 50, Height: 12})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("my chats view should request alt-screen")
	}
	lines := strings.Split(view.Content, "\n")
	if got := len(lines); got != 12 {
		t.Fatalf("line count = %d, want 12; content = %q", got, view.Content)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != 50 {
			t.Fatalf("line %d width = %d, want 50", i, got)
		}
	}
	if !strings.Contains(view.Content, myChatsTitle) {
		t.Fatalf("my chats view should include title, got %q", view.Content)
	}
	if !strings.Contains(view.Content, "Town Square") {
		t.Fatalf("my chats view should include Town Square room, got %q", view.Content)
	}
	if !strings.Contains(view.Content, myChatsHintLine) {
		t.Fatalf("my chats view should include key hints, got %q", view.Content)
	}
}

func TestMyChatsContainerIsCenteredAndConstrained(t *testing.T) {
	m := newMyChatsModel(t, Config{Width: 100, Height: 30})
	layout := myChatsLayoutFor(m.screen.width, m.screen.height, m.styles)

	if got, want := layout.container.width, myChatsTargetWidth; got != want {
		t.Fatalf("container width = %d, want %d", got, want)
	}
	wantHeight := min(myChatsTargetHeight, m.screen.height-myChatsFramePaddingY*2)
	if got := layout.container.height; got != wantHeight {
		t.Fatalf("container height = %d, want %d", got, wantHeight)
	}
}

func TestMyChatsEnterRequestsContinue(t *testing.T) {
	m := newMyChatsModel(t, Config{Width: 50, Height: 12})

	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should request continue")
	}
	if _, ok := cmd().(ContinueRequested); !ok {
		t.Fatalf("enter command returned %T, want ContinueRequested", cmd())
	}
}

func TestMyChatsEscRequestsBack(t *testing.T) {
	m := newMyChatsModel(t, Config{Width: 50, Height: 12})

	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should request back")
	}
	if _, ok := cmd().(BackRequested); !ok {
		t.Fatalf("esc command returned %T, want BackRequested", cmd())
	}
}

func newMyChatsModel(t *testing.T, config Config) myChatsModel {
	t.Helper()

	tm := NewMyChats(config)
	m, ok := tm.(myChatsModel)
	if !ok {
		t.Fatalf("new my chats model has type %T, want myChatsModel", tm)
	}
	return m
}
