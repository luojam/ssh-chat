package session

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

type Config struct {
	Width   int
	Height  int
	Context context.Context
	Room    *chat.Room
	Member  chat.Member
}

func New(config Config) tea.Model {
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return model{
		ctx:          ctx,
		room:         config.Room,
		member:       config.Member,
		subscription: config.Room.Subscribe(),
		ui: tui.New(tui.Config{
			Width:  config.Width,
			Height: config.Height,
		}),
	}
}

type model struct {
	ctx          context.Context
	room         *chat.Room
	member       chat.Member
	subscription *chat.Subscription
	ui           tea.Model
}

type roomEvent struct {
	event chat.Event
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.ui.Init(), m.waitForRoomEvent())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.SendRequested:
		return m, m.postMessage(msg.Body)
	case roomEvent:
		display := tui.MessageReceived{
			Author: msg.event.Message.Author.Name,
			Body:   msg.event.Message.Body,
			Mine:   msg.event.Message.Author.Name == m.member.Name,
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
		_, _ = m.room.Post(m.member, body)
		return nil
	}
}

func (m model) waitForRoomEvent() tea.Cmd {
	return func() tea.Msg {
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
