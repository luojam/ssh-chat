package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestDashboardViewUsesFullFrame(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 50, Height: 14})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("dashboard view should request alt-screen")
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
	if !strings.Contains(view.Content, dashboardHeaderLines[0]) {
		t.Fatalf("dashboard view should include heading, got %q", view.Content)
	}
	if !strings.Contains(view.Content, dashboardHintLine) {
		t.Fatalf("dashboard view should include key hints, got %q", view.Content)
	}
}

func TestDashboardViewIncludesButtons(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 70, Height: 20})
	view := m.View()

	if got, want := len(dashboardItems), 3; got != want {
		t.Fatalf("dashboard item count = %d, want %d", got, want)
	}

	first := dashboardItems[0]
	if first.section != dashboardSectionMyChats {
		t.Fatalf("first dashboard item section = %d, want %d", first.section, dashboardSectionMyChats)
	}
	if first.title != dashboardMyChatsTitle {
		t.Fatalf("first dashboard item title = %q, want %q", first.title, dashboardMyChatsTitle)
	}

	for _, item := range dashboardItems {
		if !strings.Contains(view.Content, item.title) {
			t.Fatalf("dashboard view should include %q, got %q", item.title, view.Content)
		}
	}
}

func TestDashboardBoxAlmostFillsSmallFrame(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 20, Height: 8})
	box := dashboardBoxSize(m.screen.frame(), m.styles.box)

	if got, want := box.width, 16; got != want {
		t.Fatalf("box width = %d, want %d", got, want)
	}
	if got, want := box.height, 8; got != want {
		t.Fatalf("box height = %d, want %d", got, want)
	}
}

func TestDashboardBoxIsCenteredContainerOnLargeFrame(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 100, Height: 30})
	layout := dashboardLayoutFor(m.screen.width, m.screen.height, m.styles)

	if got, want := layout.box.width, dashboardTargetBoxWidth; got != want {
		t.Fatalf("box width = %d, want %d", got, want)
	}
	if got, want := layout.box.height, dashboardTargetBoxHeight; got != want {
		t.Fatalf("box height = %d, want %d", got, want)
	}
}

func TestDashboardArrowKeysMoveSelection(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 70, Height: 20})

	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(dashboardModel)
	if got, want := m.selectedIndex, 1; got != want {
		t.Fatalf("selected index after right = %d, want %d", got, want)
	}

	next, _ = m.Update(keySpecial(tea.KeyLeft))
	m = next.(dashboardModel)
	if got, want := m.selectedIndex, 0; got != want {
		t.Fatalf("selected index after left = %d, want %d", got, want)
	}
}

func TestDashboardSelectionWraps(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 70, Height: 20})

	next, _ := m.Update(keySpecial(tea.KeyLeft))
	m = next.(dashboardModel)
	if got, want := m.selectedIndex, len(dashboardItems)-1; got != want {
		t.Fatalf("selected index after wrapping left = %d, want %d", got, want)
	}
}

func TestDashboardEnterRequestsSelectedAction(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should request selected dashboard action")
	}
	msg := cmd()
	selection, ok := msg.(DashboardSelectionRequested)
	if !ok {
		t.Fatalf("enter command = %T, want DashboardSelectionRequested", msg)
	}
	if selection.Action != DashboardActionMyChats {
		t.Fatalf("selected action = %d, want %d", selection.Action, DashboardActionMyChats)
	}
}

func TestDashboardCtrlLDoesNothing(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keyCtrl("l"))
	if cmd != nil {
		t.Fatalf("ctrl+l command = %T, want nil", cmd())
	}
}

func newDashboardModel(t *testing.T, config Config) dashboardModel {
	t.Helper()

	tm := NewDashboard(config)
	m, ok := tm.(dashboardModel)
	if !ok {
		t.Fatalf("new dashboard model has type %T, want dashboardModel", tm)
	}
	return m
}
