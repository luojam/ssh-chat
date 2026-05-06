package chat

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrEmptyMessage = errors.New("empty message")

const (
	historyLimit       = 16
	subscriptionBuffer = 16
)

// MemberID is an opaque room participant identifier. The SSH server assigns a
// new ID per connection; it is not a durable user account ID unless you add that layer.
type MemberID string

// Member is the chat package's name for "someone in the room": messages and join/leave
// events reference it. Session code chooses which Member value applies to each SSH client.
type Member struct {
	ID   MemberID
	Name string
}

type MessageID uint64

type Message struct {
	ID        MessageID
	Author    Member
	Body      string
	CreatedAt time.Time
}

type EventKind int

const (
	MessagePosted EventKind = iota + 1
	MemberJoined
	MemberLeft
)

type Event struct {
	Kind    EventKind
	Message Message
	Member  Member
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
	nextID      MessageID
	messages    []Message
	subscribers map[chan Event]subscriber
}

type subscriber struct {
	member Member
}

func NewRoom() *Room {
	return &Room{
		nextID:      1,
		subscribers: make(map[chan Event]subscriber),
	}
}

// Join registers member in the room and returns their event stream. The stream
// starts with recent message history; existing members are notified that member
// joined, but the joining member does not receive their own MemberJoined event.
func (r *Room) Join(member Member) *Subscription {
	// Capacity includes replay plus live slack so a full history replay never makes
	// a new member look slow before their session can read its first event.
	events := make(chan Event, historyLimit+subscriptionBuffer)

	r.mu.Lock()
	history := r.history()
	for _, msg := range history {
		events <- Event{
			Kind:    MessagePosted,
			Message: msg,
		}
	}
	r.broadcast(Event{
		Kind:   MemberJoined,
		Member: member,
	})
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
			r.broadcast(Event{
				Kind:   MemberLeft,
				Member: subscriber.member,
			})
		},
	}
}

func (r *Room) Post(author Member, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}

	// A room is shared by many SSH sessions. The mutex keeps message storage and
	// broadcast order together so subscribers see accepted messages in room order.
	r.mu.Lock()
	defer r.mu.Unlock()

	msg := Message{
		ID:        r.nextID,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now(),
	}
	r.nextID++

	event := Event{
		Kind:    MessagePosted,
		Message: msg,
	}
	r.messages = append(r.messages, msg)
	r.broadcast(event)

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
			// A full subscriber is not keeping up with room traffic. Closing it
			// keeps one slow SSH session from blocking every other participant.
			delete(r.subscribers, sub)
			close(sub)
		}
	}
}
