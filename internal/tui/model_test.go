package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestInitialLayoutUsesFullFrame(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	if got := len(lines); got != 8 {
		t.Fatalf("line count = %d, want 8; content = %q", got, view.Content)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != 40 {
			t.Fatalf("line %d width = %d, want 40; line = %q", i, got, line)
		}
	}
	if !strings.Contains(lines[0], appName) || !strings.Contains(lines[0], statusConnected) {
		t.Fatalf("header should include app name and status, got %q", lines[0])
	}
	if !strings.Contains(view.Content, "No messages yet.") {
		t.Fatalf("view should include empty state, got %q", view.Content)
	}
	if !strings.Contains(lines[len(lines)-1], composerPrompt) {
		t.Fatalf("last line should include composer prompt, got %q", lines[len(lines)-1])
	}
}

func TestEnterAppendsLocalMessageAndClearsInput(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	m = updateModel(t, m, keyText("h"))
	m = updateModel(t, m, keyText("i"))
	m = updateModel(t, m, keySpecial(tea.KeyEnter))

	if got := len(m.messages); got != 1 {
		t.Fatalf("message count = %d, want 1", got)
	}
	if got := m.messages[0].body; got != "hi" {
		t.Fatalf("message body = %q, want %q", got, "hi")
	}
	if got := m.input.Value(); got != "" {
		t.Fatalf("input value = %q, want empty", got)
	}

	view := m.View()
	if !view.AltScreen {
		t.Fatal("view should request alt-screen")
	}
	if !strings.Contains(view.Content, "you") || !strings.Contains(view.Content, "hi") {
		t.Fatalf("view content should render local message, got %q", view.Content)
	}
}

func TestQIsNormalInput(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	m = updateModel(t, m, keyText("q"))

	if got := m.input.Value(); got != "q" {
		t.Fatalf("input value = %q, want q", got)
	}
	if got := len(m.messages); got != 0 {
		t.Fatalf("message count = %d, want 0", got)
	}
}

func TestCtrlCAndEscQuit(t *testing.T) {
	for _, msg := range []tea.KeyPressMsg{
		keyCtrl("c"),
		keySpecial(tea.KeyEsc),
	} {
		m := New(Config{Width: 40, Height: 8}).(model)
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatal("expected quit command, got nil")
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("expected tea.QuitMsg, got %T", msg)
		}
	}
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()

	next, _ := m.Update(msg)
	updated, ok := next.(model)
	if !ok {
		t.Fatalf("updated model has type %T, want model", next)
	}
	return updated
}

func keyText(s string) tea.KeyPressMsg {
	r := []rune(s)
	return tea.KeyPressMsg(tea.Key{
		Text: s,
		Code: r[0],
	})
}

func keyCtrl(s string) tea.KeyPressMsg {
	r := []rune(s)
	return tea.KeyPressMsg(tea.Key{
		Code: r[0],
		Mod:  tea.ModCtrl,
	})
}

func keySpecial(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}
