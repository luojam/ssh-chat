package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestInitialLayoutUsesFullFrame(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})

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
	if !strings.Contains(lines[0], defaultRoomViewTitle) {
		t.Fatalf("header should include app name, got %q", lines[0])
	}
	if strings.Contains(lines[0], "connected") || strings.Contains(lines[0], composerQuitHint) {
		t.Fatalf("header should be title only, got %q", lines[0])
	}
	if !strings.Contains(lines[1], string(inputSepRune)) {
		t.Fatalf("header divider row should be a separator, got %q", lines[1])
	}
	// if !strings.Contains(lines[6], composerQuitHint) {
	// t.Fatalf("composer placeholder should include quit hint, got %q", lines[6])
	// }
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

func TestTinyLayoutsUseFullFrameHeight(t *testing.T) {
	for height := 1; height <= 8; height++ {
		m := newRoomViewModel(t, Config{Width: 40, Height: height})

		lines := strings.Split(m.View().Content, "\n")
		if got := len(lines); got != height {
			t.Fatalf("height %d rendered %d rows; content = %q", height, got, m.View().Content)
		}
	}
}

func TestEnterRequestsSendAndClearsInput(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})

	m = updateRoomViewModel(t, m, keyText("h"))
	m = updateRoomViewModel(t, m, keyText("i"))
	m, cmd := updateRoomViewModelWithCmd(t, m, keySpecial(tea.KeyEnter))

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

	m = updateRoomViewModel(t, m, MessageReceived{Author: "you", Body: msg.Body, Role: MessageAuthorLocal})

	view := m.View()
	if !view.AltScreen {
		t.Fatal("view should request alt-screen")
	}
	if !strings.Contains(view.Content, "you") || !strings.Contains(view.Content, "hi") {
		t.Fatalf("view content should render local message, got %q", view.Content)
	}
}

func TestShortHistoryRendersAboveComposer(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})

	m = updateRoomViewModel(t, m, MessageReceived{Author: "you", Body: "hi", Role: MessageAuthorLocal})

	lines := strings.Split(m.View().Content, "\n")
	if strings.Contains(lines[2], "you") || strings.Contains(lines[2], "hi") {
		t.Fatalf("message should not start at top of message area, got %q", lines[2])
	}
	lastMsgRow := lines[4]
	if !strings.Contains(lastMsgRow, "you") || !strings.Contains(lastMsgRow, "hi") {
		t.Fatalf("bottom message row should include latest message, got %q", lastMsgRow)
	}
}

func TestViewportSyncPreservesScrollOnResizeAndThemeChange(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})
	for i := 0; i < 10; i++ {
		m = updateRoomViewModel(t, m, MessageReceived{Author: "you", Body: "message", Role: MessageAuthorLocal})
	}
	if !m.viewport.AtBottom() {
		t.Fatal("received messages should follow to bottom")
	}
	bottomOffset := m.viewport.YOffset()
	if bottomOffset == 0 {
		t.Fatal("test setup needs scrollable message content")
	}

	scrolledOffset := bottomOffset - 1
	m.viewport.SetYOffset(scrolledOffset)
	m.resize(50, 8)
	if got := m.viewport.YOffset(); got != scrolledOffset {
		t.Fatalf("resize y offset = %d, want preserved offset %d", got, scrolledOffset)
	}

	m.setDark(false)
	if got := m.viewport.YOffset(); got != scrolledOffset {
		t.Fatalf("theme change y offset = %d, want preserved offset %d", got, scrolledOffset)
	}

	m = updateRoomViewModel(t, m, MessageReceived{Author: "you", Body: "new", Role: MessageAuthorLocal})
	if !m.viewport.AtBottom() {
		t.Fatalf("new message should follow to bottom; offset = %d", m.viewport.YOffset())
	}
}

func TestEscRequestsLeaveRoomView(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})

	_, cmd := m.Update(keySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc should request leaving room view")
	}
	msg := cmd()
	if _, ok := msg.(LeaveRequested); !ok {
		t.Fatalf("esc command returned %T, want LeaveRequested", msg)
	}
}

func TestCtrlCRequestsQuit(t *testing.T) {
	m := newRoomViewModel(t, Config{Width: 40, Height: 8})
	_, cmd := m.Update(keyCtrl("c"))
	if cmd == nil {
		t.Fatal("ctrl+c should request quit")
	}
	msg := cmd()
	if _, ok := msg.(QuitRequested); !ok {
		t.Fatalf("ctrl+c command returned %T, want QuitRequested", msg)
	}
}

func newRoomViewModel(t *testing.T, config Config) roomViewModel {
	t.Helper()

	tm := NewRoomView(config)
	m, ok := tm.(roomViewModel)
	if !ok {
		t.Fatalf("new room view model has type %T, want roomViewModel", tm)
	}
	return m
}

func updateRoomViewModel(t *testing.T, m roomViewModel, msg tea.Msg) roomViewModel {
	t.Helper()

	next, _ := updateRoomViewModelWithCmd(t, m, msg)
	return next
}

func updateRoomViewModelWithCmd(t *testing.T, m roomViewModel, msg tea.Msg) (roomViewModel, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)
	updated, ok := next.(roomViewModel)
	if !ok {
		t.Fatalf("updated room view model has type %T, want roomViewModel", next)
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
