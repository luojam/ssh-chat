package chat

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrEmptyMessage = errors.New("empty message")

const subscriptionBuffer = 16
const historyLimit = subscriptionBuffer

type Member struct {
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
	joined bool
}

func NewRoom() *Room {
	return &Room{
		nextID:      1,
		subscribers: make(map[chan Event]subscriber),
	}
}

func (r *Room) Subscribe() *Subscription {
	return r.subscribe(Member{}, false)
}

func (r *Room) Join(member Member) *Subscription {
	return r.subscribe(member, true)
}

func (r *Room) subscribe(member Member, joined bool) *Subscription {
	events := make(chan Event, subscriptionBuffer)

	r.mu.Lock()
	history := r.history()
	for _, msg := range history {
		events <- Event{
			Kind:    MessagePosted,
			Message: msg,
		}
	}
	r.subscribers[events] = subscriber{
		member: member,
		joined: joined,
	}
	if joined {
		r.broadcast(Event{
			Kind:   MemberJoined,
			Member: member,
		})
	}
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
			if subscriber.joined {
				r.broadcast(Event{
					Kind:   MemberLeft,
					Member: subscriber.member,
				})
			}
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
	for events := range r.subscribers {
		select {
		case events <- event:
		default:
			// A full subscriber is not keeping up with room traffic. Closing it
			// keeps one slow SSH session from blocking every other participant.
			delete(r.subscribers, events)
			close(events)
		}
	}
}
