package session

import (
	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/tui"
)

// applyFlowIntent is the Session navigation table. Keeping transitions here
// concentrates the important invariant: only the Room View may hold a Room
// Subscription, and leaving that View closes the Subscription before navigation.
func (m *model) applyFlowIntent(msg tea.Msg) tea.Cmd {
	if m.closed {
		return nil
	}

	switch msg := msg.(type) {
	case tui.ContinueRequested:
		return m.continueFromCurrentView()
	case tui.BackRequested:
		return m.backFromCurrentView()
	case tui.MainMenuSelectionRequested:
		return m.openMainMenuSelection(msg.Action)
	case tui.LeaveRequested:
		return m.leaveRoomView()
	default:
		return nil
	}
}

func (m *model) continueFromCurrentView() tea.Cmd {
	switch m.view {
	case viewWelcome:
		return m.showStandaloneView(viewMainMenu)
	case viewMyChats:
		return m.enterRoomView()
	default:
		return nil
	}
}

func (m *model) backFromCurrentView() tea.Cmd {
	switch m.view {
	case viewMainMenu:
		return m.showStandaloneView(viewWelcome)
	case viewMyChats:
		return m.showStandaloneView(viewMainMenu)
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
		return m.showStandaloneView(viewMyChats)
	default:
		return nil
	}
}

func (m *model) showStandaloneView(view viewState) tea.Cmd {
	if m.closed {
		return nil
	}

	m.closeRoomMembership()
	m.view = view
	m.ui = m.newUI(view)
	return m.ui.Init()
}

func (m *model) enterRoomView() tea.Cmd {
	if m.closed || m.inRoomView() {
		return nil
	}

	m.closeRoomMembership()
	m.view = viewChat
	m.ui = m.newUI(viewChat)
	m.subscription = m.room.Join(m.member)

	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m *model) leaveRoomView() tea.Cmd {
	if !m.inRoomView() {
		return nil
	}
	return m.showStandaloneView(viewMainMenu)
}

func (m *model) closeRoomMembership() {
	if m.subscription == nil {
		return
	}
	m.subscription.Close()
	m.subscription = nil
}

func (m model) inRoomView() bool {
	return m.view == viewChat && m.subscription != nil
}

func (m model) newUI(view viewState) tea.Model {
	config := tui.Config{Width: m.width, Height: m.height}
	switch view {
	case viewWelcome:
		return tui.NewWelcome(config)
	case viewMainMenu:
		return tui.NewMainMenu(config)
	case viewMyChats:
		return tui.NewMyChats(config)
	case viewChat:
		return tui.NewRoomView(config)
	default:
		return tui.NewWelcome(config)
	}
}
