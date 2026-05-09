package chat

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrEmptyMessage     = errors.New("empty message")
	ErrInvalidRoomTitle = errors.New("invalid room title")
	ErrRoomNotFound     = errors.New("room not found")
	ErrNotRoomMember    = errors.New("not room member")
	ErrInvalidRoomRole  = errors.New("invalid room role")
	ErrInvalidJoinCode  = errors.New("invalid join code")
	ErrNotRoomOwner     = errors.New("not room owner")
)

const (
	historyLimit       = 16
	subscriptionBuffer = 16
	maxRoomTitleRunes  = 64
	joinCodeLength     = 8
)

type UserID string

type RoomID string

type MessageID int64

type RoomRole string

const (
	RoomRoleOwner  RoomRole = "owner"
	RoomRoleMember RoomRole = "member"
)

func ParseRoomRole(role string) (RoomRole, error) {
	switch RoomRole(role) {
	case RoomRoleOwner:
		return RoomRoleOwner, nil
	case RoomRoleMember:
		return RoomRoleMember, nil
	default:
		return "", ErrInvalidRoomRole
	}
}

type RoomSummary struct {
	ID        RoomID
	Title     string
	JoinCode  string
	Role      RoomRole
	CreatedAt time.Time
}

type StoredRoom struct {
	ID        RoomID
	Title     string
	JoinCode  string
	CreatedBy UserID
	CreatedAt time.Time
}

// Member is an authenticated user actively participating in a room through a session.
type Member struct {
	ID   UserID
	Name string
}

type Message struct {
	ID        MessageID
	RoomID    RoomID
	Author    Member
	Body      string
	CreatedAt time.Time
}

type EventKind int

const (
	MessagePosted EventKind = iota + 1
	MemberJoined
	MemberLeft
	RoomDeleted
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
	if s == nil || s.cancel == nil {
		return
	}
	s.cancel()
}

type liveRoom struct {
	mu          sync.Mutex
	subscribers map[chan Event]subscriber
}

type subscriber struct {
	member Member
}

func newLiveRoom() *liveRoom {
	return &liveRoom{subscribers: make(map[chan Event]subscriber)}
}

// joinLocked registers member in the live room and returns their event stream.
// The caller must hold r.mu. The joining member receives recent history but not
// their own MemberJoined event.
func (r *liveRoom) joinLocked(member Member, history []Message) *Subscription {
	events := make(chan Event, historyLimit+subscriptionBuffer)
	for _, msg := range history {
		events <- Event{Kind: MessagePosted, Message: msg}
	}

	r.broadcastLocked(Event{Kind: MemberJoined, Member: member})
	r.subscribers[events] = subscriber{member: member}

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
			r.broadcastLocked(Event{Kind: MemberLeft, Member: subscriber.member})
		},
	}
}

func (r *liveRoom) deleteLocked() {
	for sub := range r.subscribers {
		select {
		case sub <- Event{Kind: RoomDeleted}:
		default:
		}
		delete(r.subscribers, sub)
		close(sub)
	}
}

func (r *liveRoom) broadcastLocked(event Event) {
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
