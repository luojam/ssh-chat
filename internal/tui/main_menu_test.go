package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestMainMenuViewUsesFullFrame(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 50, Height: 14})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("main menu view should request alt-screen")
	}

	lines := strings.Split(view.Content, "\n")
	if got := len(lines); got != 14 {
		t.Fatalf("line count = %d, want 14; content = %q", got, view.Content)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != 50 {
			t.Fatalf("line %d width = %d, want 50", i, got)
		}
	}
	if !strings.Contains(view.Content, mainMenuHeaderLines[0]) {
		t.Fatalf("main menu view should include heading, got %q", view.Content)
	}
	if !strings.Contains(view.Content, mainMenuHintLine) {
		t.Fatalf("main menu view should include key hints, got %q", view.Content)
	}
}

func TestMainMenuViewIncludesButtons(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 70, Height: 20})
	view := m.View()

	if got, want := len(mainMenuItems), 3; got != want {
		t.Fatalf("main menu item count = %d, want %d", got, want)
	}

	first := mainMenuItems[0]
	if first.section != mainMenuSectionMyChats {
		t.Fatalf("first main menu item section = %d, want %d", first.section, mainMenuSectionMyChats)
	}
	if first.title != mainMenuMyChatsTitle {
		t.Fatalf("first main menu item title = %q, want %q", first.title, mainMenuMyChatsTitle)
	}

	for _, item := range mainMenuItems {
		if !strings.Contains(view.Content, item.title) {
			t.Fatalf("main menu view should include %q, got %q", item.title, view.Content)
		}
	}
}

func TestMainMenuBoxAlmostFillsSmallFrame(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 20, Height: 8})
	box := mainMenuBoxSize(m.screen.frame(), m.styles.box)

	if got, want := box.width, 16; got != want {
		t.Fatalf("box width = %d, want %d", got, want)
	}
	if got, want := box.height, 8; got != want {
		t.Fatalf("box height = %d, want %d", got, want)
	}
}

func TestMainMenuBoxIsCenteredContainerOnLargeFrame(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 100, Height: 30})
	layout := mainMenuLayoutFor(m.screen.width, m.screen.height, m.styles)

	if got, want := layout.box.width, mainMenuTargetBoxWidth; got != want {
		t.Fatalf("box width = %d, want %d", got, want)
	}
	if got, want := layout.box.height, mainMenuTargetBoxHeight; got != want {
		t.Fatalf("box height = %d, want %d", got, want)
	}
}

func TestMainMenuArrowKeysMoveSelection(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 70, Height: 20})

	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(mainMenuModel)
	if got, want := m.selectedIndex, 1; got != want {
		t.Fatalf("selected index after right = %d, want %d", got, want)
	}

	next, _ = m.Update(keySpecial(tea.KeyLeft))
	m = next.(mainMenuModel)
	if got, want := m.selectedIndex, 0; got != want {
		t.Fatalf("selected index after left = %d, want %d", got, want)
	}
}

func TestMainMenuSelectionWraps(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 70, Height: 20})

	next, _ := m.Update(keySpecial(tea.KeyLeft))
	m = next.(mainMenuModel)
	if got, want := m.selectedIndex, len(mainMenuItems)-1; got != want {
		t.Fatalf("selected index after wrapping left = %d, want %d", got, want)
	}
}

func TestMainMenuEnterRequestsSelectedAction(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should request selected main menu action")
	}
	msg := cmd()
	selection, ok := msg.(MainMenuSelectionRequested)
	if !ok {
		t.Fatalf("enter command = %T, want MainMenuSelectionRequested", msg)
	}
	if selection.Action != MainMenuActionMyChats {
		t.Fatalf("selected action = %d, want %d", selection.Action, MainMenuActionMyChats)
	}
}

func TestMainMenuEscRequestsBack(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should request back")
	}
	if _, ok := cmd().(BackRequested); !ok {
		t.Fatalf("esc command returned %T, want BackRequested", cmd())
	}
}

func TestMainMenuCtrlCRequestsQuit(t *testing.T) {
	m := newMainMenuModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keyCtrl("c"))
	if cmd == nil {
		t.Fatal("ctrl+c should request quit")
	}
	if _, ok := cmd().(QuitRequested); !ok {
		t.Fatalf("ctrl+c command returned %T, want QuitRequested", cmd())
	}
}

func newMainMenuModel(t *testing.T, config Config) mainMenuModel {
	t.Helper()

	tm := NewMainMenu(config)
	m, ok := tm.(mainMenuModel)
	if !ok {
		t.Fatalf("new main menu model has type %T, want mainMenuModel", tm)
	}
	return m
}
