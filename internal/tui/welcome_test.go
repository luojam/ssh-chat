package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestWelcomeViewUsesFullFrame(t *testing.T) {
	m := newWelcomeModel(t, Config{Width: 50, Height: 10})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("welcome view should request alt-screen")
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
	if !strings.Contains(view.Content, welcomeContinueLine) {
		t.Fatalf("welcome view should include continue hint, got %q", view.Content)
	}
	if !strings.Contains(view.Content, welcomeQuitLine) {
		t.Fatalf("welcome view should include quit hint, got %q", view.Content)
	}
}

func TestWelcomeShowsFullLogoWhenItFits(t *testing.T) {
	m := newWelcomeModel(t, Config{Width: 70, Height: 15})

	view := m.View()
	if !strings.Contains(view.Content, welcomeLogoFull[0]) {
		t.Fatalf("welcome view should include full logo, got %q", view.Content)
	}
	if strings.Contains(view.Content, welcomeTitleLine) {
		t.Fatalf("welcome view should omit title when logo is visible, got %q", view.Content)
	}
}

func TestWelcomeStacksLogoWhenTooNarrow(t *testing.T) {
	m := newWelcomeModel(t, Config{Width: 45, Height: 22})

	view := m.View()
	if strings.Contains(view.Content, welcomeLogoFull[0]) {
		t.Fatalf("welcome view should not include full logo when narrow, got %q", view.Content)
	}
	if !strings.Contains(view.Content, welcomeLogoStacked[0]) || !strings.Contains(view.Content, welcomeLogoStacked[7]) {
		t.Fatalf("welcome view should include stacked logo, got %q", view.Content)
	}
}

func TestWelcomeEnterRequestsContinue(t *testing.T) {
	m := newWelcomeModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should request continue")
	}
	msg := cmd()
	if _, ok := msg.(ContinueRequested); !ok {
		t.Fatalf("enter command returned %T, want ContinueRequested", msg)
	}
}

func TestWelcomeCtrlCRequestsQuit(t *testing.T) {
	m := newWelcomeModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keyCtrl("c"))
	if cmd == nil {
		t.Fatal("ctrl+c should request quit")
	}
	got := cmd()
	if _, ok := got.(QuitRequested); !ok {
		t.Fatalf("ctrl+c command returned %T, want QuitRequested", got)
	}
}

func newWelcomeModel(t *testing.T, config Config) welcomeModel {
	t.Helper()

	tm := NewWelcome(config)
	m, ok := tm.(welcomeModel)
	if !ok {
		t.Fatalf("new welcome model has type %T, want welcomeModel", tm)
	}
	return m
}
