package chat

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrEmptyMessage = errors.New("empty message")

type Member struct {
	Name string
}

type Message struct {
	Author    Member
	Body      string
	CreatedAt time.Time
}

type Event struct {
	Message Message
}

type Subscription struct {
	events <-chan Event
	cancel func()
}

func (s *Subscription) Events() <-chan Event {
	return s.events
}

func (s *Subscription) Close() {
	s.cancel()
}

type Room struct {
	mu          sync.Mutex
	messages    []Message
	subscribers map[chan Event]struct{}
}

func NewRoom() *Room {
	return &Room{
		subscribers: make(map[chan Event]struct{}),
	}
}

func (r *Room) Subscribe() *Subscription {
	events := make(chan Event, 16)

	r.mu.Lock()
	r.subscribers[events] = struct{}{}
	r.mu.Unlock()

	return &Subscription{
		events: events,
		cancel: func() {
			r.mu.Lock()
			defer r.mu.Unlock()

			if _, ok := r.subscribers[events]; !ok {
				return
			}
			delete(r.subscribers, events)
			close(events)
		},
	}
}

func (r *Room) Post(author Member, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}

	msg := Message{
		Author:    author,
		Body:      body,
		CreatedAt: time.Now(),
	}

	event := Event{Message: msg}

	// A room is shared by many SSH sessions. The mutex keeps message storage and
	// broadcast order together so subscribers see accepted messages in room order.
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, msg)
	for subscriber := range r.subscribers {
		subscriber <- event
	}

	return msg, nil
}
