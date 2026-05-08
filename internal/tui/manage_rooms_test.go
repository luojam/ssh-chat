package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestManageRoomsViewIncludesActions(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})
	view := m.View()
	if !view.AltScreen {
		t.Fatal("manage rooms view should request alt-screen")
	}
	for _, want := range []string{manageRoomsHeadingLine, manageRoomsCreateTitle, manageRoomsJoinTitle, manageRoomsMenuHintLine} {
		if !strings.Contains(view.Content, want) {
			t.Fatalf("view should include %q, got %q", want, view.Content)
		}
	}
}

func TestManageRoomsArrowKeysMoveSelectionButHLDoNot(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})

	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(manageRoomsModel)
	if m.selectedIndex != 1 {
		t.Fatalf("selected index after right = %d, want 1", m.selectedIndex)
	}

	next, _ = m.Update(keyText("h"))
	m = next.(manageRoomsModel)
	if m.selectedIndex != 1 {
		t.Fatalf("h should not change selected index, got %d", m.selectedIndex)
	}

	next, _ = m.Update(keySpecial(tea.KeyLeft))
	m = next.(manageRoomsModel)
	if m.selectedIndex != 0 {
		t.Fatalf("selected index after left = %d, want 0", m.selectedIndex)
	}
}

func TestManageRoomsJoinRoomSelectionIsNoOp(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})
	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(manageRoomsModel)

	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)
	if cmd != nil {
		t.Fatal("enter on Join Room placeholder should not return command")
	}
	if m.mode != manageRoomsModeMenu || m.selectedIndex != 1 {
		t.Fatalf("join no-op changed model: mode %d index %d", m.mode, m.selectedIndex)
	}
}

func TestManageRoomsCreateRoomSubmissionAndFailure(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})
	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)
	if cmd == nil {
		t.Fatal("enter on Create Room should focus input")
	}
	if m.mode != manageRoomsModeCreate {
		t.Fatalf("mode = %d, want create", m.mode)
	}

	m = updateManageRoomsModel(t, m, keyText("r"))
	m = updateManageRoomsModel(t, m, keyText("o"))
	m = updateManageRoomsModel(t, m, keyText("o"))
	m = updateManageRoomsModel(t, m, keyText("m"))
	_, cmd = m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter in create mode should request create")
	}
	msg, ok := cmd().(CreateRoomRequested)
	if !ok {
		t.Fatalf("command returned %T, want CreateRoomRequested", cmd())
	}
	if msg.Title != "room" {
		t.Fatalf("title = %q, want room", msg.Title)
	}

	next, _ = m.Update(CreateRoomFailed{Message: "bad title"})
	m = next.(manageRoomsModel)
	if !strings.Contains(m.View().Content, "bad title") {
		t.Fatalf("failure should render error, got %q", m.View().Content)
	}
}

func TestManageRoomsEscBehavior(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})
	next, _ := m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)

	next, cmd := m.Update(keySpecial(tea.KeyEsc))
	m = next.(manageRoomsModel)
	if cmd != nil {
		t.Fatal("esc from create mode should not ask session to go back")
	}
	if m.mode != manageRoomsModeMenu {
		t.Fatalf("mode = %d, want menu", m.mode)
	}

	_, cmd = m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc from menu should request back")
	}
	if _, ok := cmd().(BackRequested); !ok {
		t.Fatalf("esc command returned %T, want BackRequested", cmd())
	}
}

func newManageRoomsModel(t *testing.T, config Config) manageRoomsModel {
	t.Helper()
	tm := NewManageRooms(config)
	m, ok := tm.(manageRoomsModel)
	if !ok {
		t.Fatalf("new manage rooms model has type %T, want manageRoomsModel", tm)
	}
	return m
}

func updateManageRoomsModel(t *testing.T, m manageRoomsModel, msg tea.Msg) manageRoomsModel {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(manageRoomsModel)
	if !ok {
		t.Fatalf("updated manage rooms model has type %T, want manageRoomsModel", next)
	}
	return updated
}
