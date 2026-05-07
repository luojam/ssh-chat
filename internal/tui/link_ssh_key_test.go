package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestLinkSSHKeyViewRendersFingerprint(t *testing.T) {
	m := newLinkSSHKeyModel(t, Config{Width: 70, Height: 18, SSHKeyFingerprint: "SHA256:abc123"})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("link ssh key view should request alt-screen")
	}
	if !strings.Contains(view.Content, linkSSHKeyQuestionLine) {
		t.Fatalf("view should include question, got %q", view.Content)
	}
	if !strings.Contains(view.Content, "SHA256:abc123") {
		t.Fatalf("view should include fingerprint, got %q", view.Content)
	}
}

func TestLinkSSHKeyArrowKeysSwitchSelection(t *testing.T) {
	m := newLinkSSHKeyModel(t, Config{Width: 70, Height: 18, SSHKeyFingerprint: "SHA256:abc123"})
	if !m.selectedYes {
		t.Fatal("initial selection should be yes")
	}

	next, _ := m.Update(keySpecial(tea.KeyRight))
	m = next.(linkSSHKeyModel)
	if m.selectedYes {
		t.Fatal("right should switch to no")
	}

	next, _ = m.Update(keySpecial(tea.KeyLeft))
	m = next.(linkSSHKeyModel)
	if !m.selectedYes {
		t.Fatal("left should switch back to yes")
	}
}

func TestLinkSSHKeyEnterEmitsSelectedIntent(t *testing.T) {
	m := newLinkSSHKeyModel(t, Config{Width: 70, Height: 18, SSHKeyFingerprint: "SHA256:abc123"})
	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter should emit selected link intent")
	}
	msg, ok := cmd().(LinkSSHKeySelectionRequested)
	if !ok {
		t.Fatalf("enter command = %T, want LinkSSHKeySelectionRequested", cmd())
	}
	if !msg.Link {
		t.Fatal("initial enter should choose yes")
	}
}

func TestLinkSSHKeyEscChoosesNo(t *testing.T) {
	m := newLinkSSHKeyModel(t, Config{Width: 70, Height: 18, SSHKeyFingerprint: "SHA256:abc123"})
	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should emit no intent")
	}
	msg, ok := cmd().(LinkSSHKeySelectionRequested)
	if !ok {
		t.Fatalf("esc command = %T, want LinkSSHKeySelectionRequested", cmd())
	}
	if msg.Link {
		t.Fatal("esc should choose no")
	}
}

func TestLinkSSHKeyCtrlCRequestsQuit(t *testing.T) {
	m := newLinkSSHKeyModel(t, Config{Width: 70, Height: 18, SSHKeyFingerprint: "SHA256:abc123"})
	_, cmd := m.Update(keyCtrl("c"))
	if cmd == nil {
		t.Fatal("ctrl+c should request quit")
	}
	if _, ok := cmd().(QuitRequested); !ok {
		t.Fatalf("ctrl+c command returned %T, want QuitRequested", cmd())
	}
}

func newLinkSSHKeyModel(t *testing.T, config Config) linkSSHKeyModel {
	t.Helper()

	tm := NewLinkSSHKey(config)
	m, ok := tm.(linkSSHKeyModel)
	if !ok {
		t.Fatalf("new link ssh key model has type %T, want linkSSHKeyModel", tm)
	}
	return m
}
