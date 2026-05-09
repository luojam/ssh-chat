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
	for _, want := range []string{manageRoomsHeadingLine, manageRoomsCreateTitle, manageRoomsJoinTitle, manageRoomsDeleteTitle, manageRoomsMenuHintLine} {
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

func TestManageRoomsJoinRoomSubmissionAndFailure(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 16})
	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(manageRoomsModel)

	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)
	if cmd == nil {
		t.Fatal("enter on Join Room should focus input")
	}
	if m.mode != manageRoomsModeJoin {
		t.Fatalf("mode = %d, want join", m.mode)
	}
	if !strings.Contains(m.View().Content, manageRoomsJoinTitle) || strings.Contains(m.View().Content, manageRoomsHeadingLine) {
		t.Fatalf("join mode should use join title, got %q", m.View().Content)
	}

	for _, r := range "7kq9-m2xp" {
		m = updateManageRoomsModel(t, m, keyText(string(r)))
	}
	_, cmd = m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter in join mode should request join")
	}
	msg, ok := cmd().(JoinRoomRequested)
	if !ok {
		t.Fatalf("command returned %T, want JoinRoomRequested", cmd())
	}
	if msg.JoinCode != "7kq9-m2xp" {
		t.Fatalf("join code = %q, want entered code", msg.JoinCode)
	}

	next, _ = m.Update(JoinRoomFailed{Message: "Invalid join code."})
	m = next.(manageRoomsModel)
	if m.mode != manageRoomsModeJoin || !strings.Contains(m.View().Content, "Invalid join code.") {
		t.Fatalf("failure should preserve join mode and render error, got mode %d view %q", m.mode, m.View().Content)
	}
}

func TestManageRoomsDeleteRoomSelectionAndConfirmation(t *testing.T) {
	m := newManageRoomsModel(t, Config{
		Width:      70,
		Height:     20,
		OwnedRooms: []RoomListItem{{ID: "room-1", Title: "Project Room"}},
	})
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))

	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)
	if cmd != nil {
		t.Fatal("enter on Delete Room should open owned room list without command")
	}
	if m.mode != manageRoomsModeDelete || !strings.Contains(m.View().Content, "Project Room") {
		t.Fatalf("delete mode should render owned rooms, mode %d view %q", m.mode, m.View().Content)
	}

	next, cmd = m.Update(keySpecial(tea.KeyEnter))
	m = next.(manageRoomsModel)
	if cmd != nil {
		t.Fatal("selecting a room should open confirmation without command")
	}
	if m.mode != manageRoomsModeDeleteConfirm || !strings.Contains(m.View().Content, "Delete") || !strings.Contains(m.View().Content, "Project Room") {
		t.Fatalf("confirmation should render room name, mode %d view %q", m.mode, m.View().Content)
	}

	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyLeft))
	_, cmd = m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("confirming Delete should request room deletion")
	}
	msg, ok := cmd().(DeleteRoomRequested)
	if !ok {
		t.Fatalf("command returned %T, want DeleteRoomRequested", cmd())
	}
	if msg.RoomID != "room-1" {
		t.Fatalf("room ID = %q, want room-1", msg.RoomID)
	}
}

func TestManageRoomsDeleteRoomListShowsMultipleRooms(t *testing.T) {
	m := newManageRoomsModel(t, Config{
		Width:  70,
		Height: 20,
		OwnedRooms: []RoomListItem{
			{ID: "room-1", Title: "Project Room"},
			{ID: "room-2", Title: "Design Room"},
			{ID: "room-3", Title: "Ops Room"},
		},
	})
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyEnter))

	view := m.View().Content
	for _, want := range []string{"Project Room", "Design Room", "Ops Room"} {
		if !strings.Contains(view, want) {
			t.Fatalf("delete room list missing %q, got %q", want, view)
		}
	}
}

func TestManageRoomsDeleteRoomEmptyState(t *testing.T) {
	m := newManageRoomsModel(t, Config{Width: 70, Height: 20})
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyRight))
	m = updateManageRoomsModel(t, m, keySpecial(tea.KeyEnter))

	if !strings.Contains(m.View().Content, manageRoomsDeleteEmptyState) {
		t.Fatalf("delete empty state missing, got %q", m.View().Content)
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
	if !strings.Contains(m.View().Content, manageRoomsCreateTitle) || strings.Contains(m.View().Content, manageRoomsHeadingLine) {
		t.Fatalf("create mode should use create title, got %q", m.View().Content)
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
