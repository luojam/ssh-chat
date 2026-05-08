package chat

import (
	"strings"
	"sync"
	"time"
)

// Room is kept only for existing package/session tests while application code
// migrates to Service. New application code should use Service instead.
type Room struct {
	mu          sync.Mutex
	nextID      MessageID
	messages    []Message
	subscribers map[chan Event]subscriber
}

func NewRoom() *Room {
	return &Room{
		nextID:      1,
		subscribers: make(map[chan Event]subscriber),
	}
}

func (r *Room) Join(member Member) *Subscription {
	events := make(chan Event, historyLimit+subscriptionBuffer)

	r.mu.Lock()
	history := r.history()
	for _, msg := range history {
		events <- Event{Kind: MessagePosted, Message: msg}
	}
	r.broadcast(Event{Kind: MemberJoined, Member: member})
	r.subscribers[events] = subscriber{member: member}
	r.mu.Unlock()

	return &Subscription{
		events: events,
		cancel: func() {
			r.mu.Lock()
			defer r.mu.Unlock()

			subscriber, ok := r.subscribers[events]
			if !ok {
				return
			}
			delete(r.subscribers, events)
			close(events)
			r.broadcast(Event{Kind: MemberLeft, Member: subscriber.member})
		},
	}
}

func (r *Room) Post(author Member, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	msg := Message{
		ID:        r.nextID,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now(),
	}
	r.nextID++

	r.messages = append(r.messages, msg)
	r.broadcast(Event{Kind: MessagePosted, Message: msg})
	return msg, nil
}

func (r *Room) history() []Message {
	start := max(0, len(r.messages)-historyLimit)
	history := make([]Message, len(r.messages[start:]))
	copy(history, r.messages[start:])
	return history
}

func (r *Room) broadcast(event Event) {
	for sub := range r.subscribers {
		select {
		case sub <- event:
		default:
			delete(r.subscribers, sub)
			close(sub)
		}
	}
}
