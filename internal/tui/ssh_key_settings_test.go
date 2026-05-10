package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSSHKeySettingsViewRendersCurrentAndLinkedKeys(t *testing.T) {
	m := newSSHKeySettingsModel(t, Config{Width: 80, Height: 20, SSHKeyFingerprint: "SHA256:current", LinkedSSHKeyFingerprint: "SHA256:linked"})
	view := m.View()

	for _, want := range []string{sshKeySettingsTitle, sshKeySettingsCurrentLabel, "SHA256:current", sshKeySettingsLinkedLabel, "SHA256:linked", sshKeySettingsLinkButton, sshKeySettingsDeleteButton, sshKeySettingsHintLine} {
		if !strings.Contains(view.Content, want) {
			t.Fatalf("ssh key settings view should include %q, got %q", want, view.Content)
		}
	}
}

func TestSSHKeySettingsEnterEmitsSelectedIntent(t *testing.T) {
	m := newSSHKeySettingsModel(t, Config{Width: 80, Height: 20, SSHKeyFingerprint: "SHA256:current"})

	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on link current key should produce command")
	}
	if _, ok := cmd().(LinkCurrentSSHKeyRequested); !ok {
		t.Fatalf("command returned %T, want LinkCurrentSSHKeyRequested", cmd())
	}

	m = updateSSHKeySettingsModel(t, m, keySpecial(tea.KeyRight))
	_, cmd = m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on delete linked key should produce command")
	}
	if _, ok := cmd().(DeleteLinkedSSHKeyRequested); !ok {
		t.Fatalf("command returned %T, want DeleteLinkedSSHKeyRequested", cmd())
	}
}

func TestSSHKeySettingsEscRequestsBack(t *testing.T) {
	m := newSSHKeySettingsModel(t, Config{Width: 80, Height: 20})

	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should request back")
	}
	if _, ok := cmd().(BackRequested); !ok {
		t.Fatalf("command returned %T, want BackRequested", cmd())
	}
}

func updateSSHKeySettingsModel(t *testing.T, m sshKeySettingsModel, msg tea.Msg) sshKeySettingsModel {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(sshKeySettingsModel)
	if !ok {
		t.Fatalf("updated ssh key settings model has type %T, want sshKeySettingsModel", next)
	}
	return updated
}

func newSSHKeySettingsModel(t *testing.T, config Config) sshKeySettingsModel {
	t.Helper()
	tm := NewSSHKeySettings(config)
	m, ok := tm.(sshKeySettingsModel)
	if !ok {
		t.Fatalf("new ssh key settings model has type %T, want sshKeySettingsModel", tm)
	}
	return m
}
