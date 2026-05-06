package session

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

const systemAuthor = "[system]"

type viewState int

const (
	viewWelcome viewState = iota
	viewMainMenu
	viewMyChats
	viewChat
)

// Config wires one Bubble Tea session to the shared room. Member is this SSH client's
// identity in the room (same type chat uses for authors and join/leave); the transport
// layer constructs it and passes it in—chat does not infer "current user" by itself.
type Config struct {
	Width   int
	Height  int
	Context context.Context
	Room    *chat.Room
	// Member is who this session joins and posts as; kept on the model to attribute
	// outgoing messages and to mark incoming lines as Mine vs others.
	Member chat.Member
}

func New(config Config) tea.Model {
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return model{
		ctx:    ctx,
		room:   config.Room,
		member: config.Member,
		width:  config.Width,
		height: config.Height,
		view:   viewWelcome,
		ui: tui.NewWelcome(tui.Config{
			Width:  config.Width,
			Height: config.Height,
		}),
	}
}

type model struct {
	ctx          context.Context
	room         *chat.Room
	member       chat.Member // local participant; pairs with Config.Member at construction
	width        int
	height       int
	subscription *chat.Subscription
	view         viewState
	joined       bool
	closed       bool
	ui           tea.Model
}

type roomEvent struct {
	event chat.Event
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// The session model owns cross-view state: room membership, current terminal
	// size, and which TUI model is active. View-specific messages are handled here
	// only when they affect that shared state; everything else is delegated below.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tui.QuitRequested:
		m.close()
		return m, tea.Quit
	case tui.ContinueRequested:
		return m, m.continueFromCurrentView()
	case tui.BackRequested:
		return m, m.backFromCurrentView()
	case tui.MainMenuSelectionRequested:
		return m, m.openMainMenuSelection(msg.Action)
	case tui.LeaveRequested:
		return m, m.leaveChat()
	case tui.SendRequested:
		return m, m.postMessage(msg.Body)
	case roomEvent:
		if m.view != viewChat {
			return m, nil
		}
		display, ok := m.displayMessage(msg.event)
		if !ok {
			return m, m.waitForRoomEvent()
		}
		var cmd tea.Cmd
		m.ui, cmd = m.ui.Update(display)
		return m, tea.Batch(cmd, m.waitForRoomEvent())
	}

	var cmd tea.Cmd
	m.ui, cmd = m.ui.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return m.ui.View()
}

func (m model) postMessage(body string) tea.Cmd {
	return func() tea.Msg {
		if m.closed || !m.joined {
			return nil
		}
		_, _ = m.room.Post(m.member, body)
		return nil
	}
}

func (m model) waitForRoomEvent() tea.Cmd {
	return func() tea.Msg {
		if m.subscription == nil {
			return nil
		}

		select {
		case <-m.ctx.Done():
			m.subscription.Close()
			return nil
		default:
		}

		select {
		case <-m.ctx.Done():
			m.subscription.Close()
			return nil
		case event, ok := <-m.subscription.Events():
			if !ok {
				return nil
			}
			return roomEvent{event: event}
		}
	}
}

func (m *model) continueFromCurrentView() tea.Cmd {
	switch m.view {
	case viewWelcome:
		return m.showMainMenu()
	case viewMainMenu, viewMyChats:
		return m.startChat()
	default:
		return nil
	}
}

func (m *model) backFromCurrentView() tea.Cmd {
	switch m.view {
	case viewMainMenu:
		return m.showWelcome()
	case viewMyChats:
		return m.showMainMenu()
	default:
		return nil
	}
}

func (m *model) openMainMenuSelection(action tui.MainMenuAction) tea.Cmd {
	if m.view != viewMainMenu {
		return nil
	}

	switch action {
	case tui.MainMenuActionMyChats:
		return m.showMyChats()
	default:
		return nil
	}
}

func (m *model) showWelcome() tea.Cmd {
	if m.closed {
		return nil
	}

	m.view = viewWelcome
	m.ui = tui.NewWelcome(tui.Config{
		Width:  m.width,
		Height: m.height,
	})
	return m.ui.Init()
}

func (m *model) showMainMenu() tea.Cmd {
	if m.closed {
		return nil
	}

	m.view = viewMainMenu
	m.ui = tui.NewMainMenu(tui.Config{
		Width:  m.width,
		Height: m.height,
	})
	return m.ui.Init()
}

func (m *model) showMyChats() tea.Cmd {
	if m.closed {
		return nil
	}

	m.view = viewMyChats
	m.ui = tui.NewMyChats(tui.Config{
		Width:  m.width,
		Height: m.height,
	})
	return m.ui.Init()
}

func (m *model) startChat() tea.Cmd {
	if m.closed || m.joined {
		return nil
	}

	m.ui = tui.NewRoomView(tui.Config{
		Width:  m.width,
		Height: m.height,
	})
	m.subscription = m.room.Join(m.member)
	m.view = viewChat
	m.joined = true

	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

// leaveChat is navigation plus membership cleanup. Keeping both together makes
// the invariant explicit: the main menu view never has an active room subscription.
func (m *model) leaveChat() tea.Cmd {
	if m.closed || !m.joined {
		return nil
	}

	if m.subscription != nil {
		m.subscription.Close()
		m.subscription = nil
	}
	m.joined = false
	return m.showMainMenu()
}

func (m model) displayMessage(event chat.Event) (tui.MessageReceived, bool) {
	switch event.Kind {
	case chat.MessagePosted:
		return tui.MessageReceived{
			Author: event.Message.Author.Name,
			Body:   event.Message.Body,
			Mine:   event.Message.Author.ID == m.member.ID,
		}, true
	case chat.MemberJoined:
		return tui.MessageReceived{
			Author: systemAuthor,
			Body:   fmt.Sprintf("%s joined", event.Member.Name),
		}, true
	case chat.MemberLeft:
		return tui.MessageReceived{
			Author: systemAuthor,
			Body:   fmt.Sprintf("%s left", event.Member.Name),
		}, true
	default:
		return tui.MessageReceived{}, false
	}
}

func (m *model) close() {
	if m.closed {
		return
	}
	m.closed = true
	if m.subscription != nil {
		m.subscription.Close()
	}
}
