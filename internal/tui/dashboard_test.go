package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestDashboardViewUsesFullFrame(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 50, Height: 10})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("dashboard view should request alt-screen")
	}

	lines := strings.Split(view.Content, "\n")
	if got := len(lines); got != 10 {
		t.Fatalf("line count = %d, want 10; content = %q", got, view.Content)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != 50 {
			t.Fatalf("line %d width = %d, want 50", i, got)
		}
	}
	if !strings.Contains(view.Content, dashboardMenuTitleLine) {
		t.Fatalf("dashboard view should include title, got %q", view.Content)
	}
}

func TestDashboardViewIncludesNavItems(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 50, Height: 20})
	view := m.View()

	items := m.list.Items()
	if got, want := len(items), 4; got != want {
		t.Fatalf("dashboard item count = %d, want %d", got, want)
	}

	first, ok := items[0].(dashboardItem)
	if !ok {
		t.Fatalf("first dashboard item has type %T, want dashboardItem", items[0])
	}
	if first.section != dashboardSectionMyChats {
		t.Fatalf("first dashboard item section = %d, want %d", first.section, dashboardSectionMyChats)
	}
	if first.title != dashboardMyChatsTitle {
		t.Fatalf("first dashboard item title = %q, want %q", first.title, dashboardMyChatsTitle)
	}
	if !strings.Contains(view.Content, first.title) {
		t.Fatalf("dashboard view should include first item, got %q", view.Content)
	}
}

func TestDashboardBoxAlmostFillsSmallFrame(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 20, Height: 8})
	box := dashboardBoxSize(safeFrameSize(m.width, m.height), m.styles.box)

	if got, want := box.width, 16; got != want {
		t.Fatalf("box width = %d, want %d", got, want)
	}
	if got, want := box.height, 6; got != want {
		t.Fatalf("box height = %d, want %d", got, want)
	}
}

func TestDashboardLayoutSplitsNarrowNavFromPanel(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 80, Height: 20})
	layout := dashboardLayoutFor(m.width, m.height, m.styles)

	if layout.nav.width >= layout.panel.width {
		t.Fatalf("nav width should be narrower than panel: nav=%d panel=%d", layout.nav.width, layout.panel.width)
	}
	if got, want := layout.nav.width, dashboardNavTargetWidth; got != want {
		t.Fatalf("nav width = %d, want %d", got, want)
	}

	navContent := dashboardPaneContentSize(layout.nav, m.styles.navPane)
	if got := m.list.Width(); got != navContent.width {
		t.Fatalf("list width = %d, want nav content width %d", got, navContent.width)
	}
}

func TestDashboardEnterDoesNotStartChat(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("enter command = %T, want nil", cmd())
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
