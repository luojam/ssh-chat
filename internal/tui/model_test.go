package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestInitialLayoutUsesFullFrame(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

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
	if !strings.Contains(lines[0], appName) {
		t.Fatalf("header should include app name, got %q", lines[0])
	}
	if strings.Contains(lines[0], "connected") || strings.Contains(lines[0], composerQuitHint) {
		t.Fatalf("header should be title only, got %q", lines[0])
	}
	if !strings.Contains(lines[1], string(inputSepRune)) {
		t.Fatalf("header divider row should be a separator, got %q", lines[1])
	}
	if !strings.Contains(lines[6], composerQuitHint) {
		t.Fatalf("composer placeholder should include quit hint, got %q", lines[6])
	}
	if strings.Contains(lines[2], emptyStateText) {
		t.Fatalf("empty state should not start at top of message area, got %q", lines[2])
	}
	// With an input frame (height ≥ 6): messages, top rule, composer, bottom rule.
	lastMsgRow := 4
	if !strings.Contains(lines[lastMsgRow], emptyStateText) {
		t.Fatalf("last message row should include empty state, got %q", lines[lastMsgRow])
	}
	if !strings.Contains(lines[5], string(inputSepRune)) {
		t.Fatalf("line above composer should be a separator, got %q", lines[5])
	}
	if !strings.Contains(lines[6], composerPrompt) {
		t.Fatalf("composer row should include prompt, got %q", lines[6])
	}
	if !strings.Contains(lines[7], string(inputSepRune)) {
		t.Fatalf("last line should be bottom separator, got %q", lines[7])
	}
}

func TestEnterRequestsSendAndClearsInput(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

	m = updateModel(t, m, keyText("h"))
	m = updateModel(t, m, keyText("i"))
	m, cmd := updateModelWithCmd(t, m, keySpecial(tea.KeyEnter))

	if cmd == nil {
		t.Fatal("enter should request send, got nil command")
	}
	msg, ok := cmd().(SendRequested)
	if !ok {
		t.Fatalf("enter command returned %T, want SendRequested", msg)
	}
	if got := msg.Body; got != "hi" {
		t.Fatalf("send body = %q, want %q", got, "hi")
	}
	if got := m.input.Value(); got != "" {
		t.Fatalf("input value = %q, want empty", got)
	}
	if got := len(m.messages); got != 0 {
		t.Fatalf("message count = %d, want 0 before display event", got)
	}

	m = updateModel(t, m, MessageReceived{Body: msg.Body, Mine: true})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("view should request alt-screen")
	}
	if !strings.Contains(view.Content, localAuthor) || !strings.Contains(view.Content, "hi") {
		t.Fatalf("view content should render local message, got %q", view.Content)
	}
}

func TestShortHistoryRendersAboveComposer(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

	m = updateModel(t, m, MessageReceived{Body: "hi", Mine: true})

	lines := strings.Split(m.View().Content, "\n")
	if strings.Contains(lines[2], localAuthor) || strings.Contains(lines[2], "hi") {
		t.Fatalf("message should not start at top of message area, got %q", lines[2])
	}
	lastMsgRow := lines[4]
	if !strings.Contains(lastMsgRow, localAuthor) || !strings.Contains(lastMsgRow, "hi") {
		t.Fatalf("bottom message row should include latest message, got %q", lastMsgRow)
	}
}

func TestHeaderStylesTitle(t *testing.T) {
	styles := newStyles(true)

	if !styles.headerTitle.GetBold() {
		t.Fatal("header title should be bold")
	}
	if styles.headerTitle.GetAlign() != lipgloss.Center {
		t.Fatal("header title should be center-aligned")
	}
}

func TestHeaderRendersFullWidth(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

	header := m.renderHeader(40)

	if got := ansi.StringWidth(header); got != 40 {
		t.Fatalf("header width = %d, want 40; header = %q", got, header)
	}
	if !strings.Contains(ansi.Strip(header), appName) {
		t.Fatalf("header should contain app name, got %q", header)
	}
}

func TestSystemAuthorStyleIsDistinct(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

	style := m.messageAuthorStyle(message{author: systemAuthor})

	if !style.GetBold() {
		t.Fatal("system author style should be bold for visual distinction")
	}
}

func TestComposerPlaceholderRightAligned(t *testing.T) {
	// At width 40, the prompt is 2 chars, leaving 38 for input.
	// The hint should appear at the trailing end of the placeholder.
	placeholder := buildPlaceholder(inputWidth(40))
	if !strings.HasSuffix(placeholder, composerQuitHint) {
		t.Fatalf("placeholder should end with quit hint, got %q", placeholder)
	}
	if !strings.HasPrefix(placeholder, "Message...") {
		t.Fatalf("placeholder should start with Message..., got %q", placeholder)
	}
}

func TestQIsNormalInput(t *testing.T) {
	m := newModel(t, Config{Width: 40, Height: 8})

	m = updateModel(t, m, keyText("q"))

	if got := m.input.Value(); got != "q" {
		t.Fatalf("input value = %q, want %q", got, "q")
	}
	if got := len(m.messages); got != 0 {
		t.Fatalf("message count = %d, want 0", got)
	}
}

func TestCtrlCAndEscRequestQuit(t *testing.T) {
	for _, msg := range []tea.KeyPressMsg{
		keyCtrl("c"),
		keySpecial(tea.KeyEsc),
	} {
		m := newModel(t, Config{Width: 40, Height: 8})
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatal("expected quit command, got nil")
		}
		msg := cmd()
		if _, ok := msg.(QuitRequested); !ok {
			t.Fatalf("expected QuitRequested, got %T", msg)
		}
	}
}

func newModel(t *testing.T, config Config) model {
	t.Helper()

	tm := New(config)
	m, ok := tm.(model)
	if !ok {
		t.Fatalf("new model has type %T, want model", tm)
	}
	return m
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()

	next, _ := updateModelWithCmd(t, m, msg)
	return next
}

func updateModelWithCmd(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)
	updated, ok := next.(model)
	if !ok {
		t.Fatalf("updated model has type %T, want model", next)
	}
	return updated, cmd
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
