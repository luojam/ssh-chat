package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSettingsViewShowsUsernameAndOptions(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})
	view := m.View()

	if !view.AltScreen {
		t.Fatal("settings view should request alt-screen")
	}
	for _, want := range []string{settingsUsernameLabel, "alice", settingsTitle, "SSH key settings", "Delete account", settingsHintLine} {
		if !strings.Contains(view.Content, want) {
			t.Fatalf("settings view should include %q, got %q", want, view.Content)
		}
	}
}

func TestSettingsEscRequestsBack(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})

	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should request back")
	}
	if _, ok := cmd().(BackRequested); !ok {
		t.Fatalf("esc command returned %T, want BackRequested", cmd())
	}
}

func TestSettingsEnterDoesNothingForNow(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})

	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd != nil {
		t.Fatal("enter on settings option should not produce a command yet")
	}
}

func newSettingsModel(t *testing.T, config Config) settingsModel {
	t.Helper()
	tm := NewSettings(config)
	m, ok := tm.(settingsModel)
	if !ok {
		t.Fatalf("new settings model has type %T, want settingsModel", tm)
	}
	return m
}
