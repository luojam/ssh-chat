package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	if !strings.Contains(lines[0], "esc/ctrl+c to quit") {
		t.Fatalf("header should include quit hint, got %q", lines[0])
	}
	if strings.Contains(lines[1], "No messages yet.") {
		t.Fatalf("empty state should not start at top of message area, got %q", lines[1])
	}
	if !strings.Contains(lines[len(lines)-2], "No messages yet.") {
		t.Fatalf("line above composer should include empty state, got %q", lines[len(lines)-2])
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

func TestShortHistoryRendersAboveComposer(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	m = updateModel(t, m, keyText("h"))
	m = updateModel(t, m, keyText("i"))
	m = updateModel(t, m, keySpecial(tea.KeyEnter))

	lines := strings.Split(m.View().Content, "\n")
	if strings.Contains(lines[1], "you") || strings.Contains(lines[1], "hi") {
		t.Fatalf("message should not start at top of message area, got %q", lines[1])
	}
	lineAboveComposer := lines[len(lines)-2]
	if !strings.Contains(lineAboveComposer, "you") || !strings.Contains(lineAboveComposer, "hi") {
		t.Fatalf("line above composer should include latest message, got %q", lineAboveComposer)
	}
}

func TestHeaderStylesKeepQuitHintDim(t *testing.T) {
	styles := newStyles(true)

	if _, ok := styles.header.GetForeground().(lipgloss.NoColor); !ok {
		t.Fatal("header bar should not set a foreground color that would override child text styles")
	}
	if _, ok := styles.header.GetBackground().(lipgloss.NoColor); ok {
		t.Fatal("header bar should keep the colorful background")
	}
	if !styles.headerTitle.GetBold() {
		t.Fatal("header title should keep the bold colorful treatment")
	}
	if !styles.headerHint.GetFaint() {
		t.Fatal("header quit hint should stay dim")
	}
}

func TestHeaderKeepsEdgePadding(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	header := m.renderHeader(40)

	if got := ansi.StringWidth(header); got != 40 {
		t.Fatalf("header width = %d, want 40; header = %q", got, header)
	}
	if !strings.HasPrefix(header, " ") {
		t.Fatalf("header should start with unstyled edge padding, got %q", header)
	}
	if !strings.HasSuffix(header, " ") {
		t.Fatalf("header should end with unstyled edge padding, got %q", header)
	}
	stripped := ansi.Strip(header)
	if !strings.HasPrefix(stripped, " "+appName) {
		t.Fatalf("header should keep one left padding cell before title, got %q", stripped)
	}
	if !strings.HasSuffix(stripped, statusConnected+" ") {
		t.Fatalf("header should keep one right padding cell after status, got %q", stripped)
	}
}

func TestQIsNormalInput(t *testing.T) {
	m := New(Config{Width: 40, Height: 8}).(model)

	m = updateModel(t, m, keyText("q"))

	if got := m.input.Value(); got != "q" {
		t.Fatalf("input value = %q, want %q", got, "q")
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
