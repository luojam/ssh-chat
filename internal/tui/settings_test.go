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

func TestSettingsEnterOnSSHKeySettingsRequestsSSHKeySettings(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})

	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on ssh key settings should request ssh key settings")
	}
	if _, ok := cmd().(SSHKeySettingsRequested); !ok {
		t.Fatalf("command returned %T, want SSHKeySettingsRequested", cmd())
	}
}

func TestSettingsDeleteAccountSelectionAndConfirmation(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})
	m = updateSettingsModel(t, m, keySpecial(tea.KeyDown))

	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(settingsModel)
	if cmd != nil {
		t.Fatal("enter on Delete account should open confirmation without command")
	}
	if m.mode != settingsModeDeleteConfirm || !strings.Contains(m.View().Content, "Delete your account permanently?") || !strings.Contains(m.View().Content, "Cancel") {
		t.Fatalf("confirmation should render destructive prompt, mode %d view %q", m.mode, m.View().Content)
	}

	m = updateSettingsModel(t, m, keySpecial(tea.KeyLeft))
	_, cmd = m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("confirming Delete should request account deletion")
	}
	if _, ok := cmd().(DeleteAccountRequested); !ok {
		t.Fatalf("command returned %T, want DeleteAccountRequested", cmd())
	}
}

func TestSettingsDeleteAccountCancelReturnsToMenu(t *testing.T) {
	m := newSettingsModel(t, Config{Width: 70, Height: 16, Username: "alice"})
	m = updateSettingsModel(t, m, keySpecial(tea.KeyDown))
	m = updateSettingsModel(t, m, keySpecial(tea.KeyEnter))

	next, cmd := m.Update(keySpecial(tea.KeyEnter))
	m = next.(settingsModel)
	if cmd != nil {
		t.Fatal("canceling account deletion should not produce a command")
	}
	if m.mode != settingsModeMenu || !strings.Contains(m.View().Content, "SSH key settings") {
		t.Fatalf("cancel should return to settings menu, mode %d view %q", m.mode, m.View().Content)
	}
}

func updateSettingsModel(t *testing.T, m settingsModel, msg tea.Msg) settingsModel {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(settingsModel)
	if !ok {
		t.Fatalf("updated settings model has type %T, want settingsModel", next)
	}
	return updated
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
