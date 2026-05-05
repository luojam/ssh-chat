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
	if !strings.Contains(view.Content, dashboardTitleLine) {
		t.Fatalf("dashboard view should include title, got %q", view.Content)
	}
	if !strings.Contains(view.Content, dashboardJoinLine) {
		t.Fatalf("dashboard view should include join hint, got %q", view.Content)
	}
}

func TestDashboardEnterRequestsContinue(t *testing.T) {
	m := newDashboardModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should request continue")
	}
	msg := cmd()
	if _, ok := msg.(ContinueRequested); !ok {
		t.Fatalf("enter command returned %T, want ContinueRequested", msg)
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
