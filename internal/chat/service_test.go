package chat

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestServiceCreateRoomCreatesOwnerMembershipAndListsForUser(t *testing.T) {
	store := newMemoryChatStore()
	service := NewService(store)

	room, err := service.CreateRoom(context.Background(), "user-1", "  Project Room  ")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if room.ID == "" {
		t.Fatal("room ID should be set")
	}
	if room.Title != "Project Room" {
		t.Fatalf("title = %q, want trimmed title", room.Title)
	}
	if room.Role != RoomRoleOwner {
		t.Fatalf("role = %q, want owner", room.Role)
	}

	rooms, err := service.ListRoomsForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListRoomsForUser returned error: %v", err)
	}
	if len(rooms) != 1 || rooms[0].ID != room.ID {
		t.Fatalf("rooms = %+v, want created room", rooms)
	}

	otherRooms, err := service.ListRoomsForUser(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("ListRoomsForUser other returned error: %v", err)
	}
	if len(otherRooms) != 0 {
		t.Fatalf("other user rooms = %+v, want none", otherRooms)
	}
}

func TestServiceCreateRoomValidatesTitle(t *testing.T) {
	service := NewService(newMemoryChatStore())

	_, err := service.CreateRoom(context.Background(), "user-1", "   ")
	if !errors.Is(err, ErrInvalidRoomTitle) {
		t.Fatalf("blank title error = %v, want ErrInvalidRoomTitle", err)
	}

	_, err = service.CreateRoom(context.Background(), "user-1", string(make([]rune, maxRoomTitleRunes+1)))
	if !errors.Is(err, ErrInvalidRoomTitle) {
		t.Fatalf("long title error = %v, want ErrInvalidRoomTitle", err)
	}
}

func TestServiceJoinAndPostRequireMembership(t *testing.T) {
	service := NewService(newMemoryChatStore())
	room, err := service.CreateRoom(context.Background(), "user-1", "Room")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	if _, err := service.JoinRoom(context.Background(), room.ID, Member{ID: "user-2", Name: "other"}); !errors.Is(err, ErrNotRoomMember) {
		t.Fatalf("JoinRoom error = %v, want ErrNotRoomMember", err)
	}
	if _, err := service.Post(context.Background(), room.ID, Member{ID: "user-2", Name: "other"}, "hi"); !errors.Is(err, ErrNotRoomMember) {
		t.Fatalf("Post error = %v, want ErrNotRoomMember", err)
	}
}

func TestServicePostPersistsThenBroadcasts(t *testing.T) {
	service := NewService(newMemoryChatStore())
	room, err := service.CreateRoom(context.Background(), "user-1", "Room")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	subscription, err := service.JoinRoom(context.Background(), room.ID, Member{ID: "user-1", Name: "user"})
	if err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	defer subscription.Close()

	msg, err := service.Post(context.Background(), room.ID, Member{ID: "user-1", Name: "user"}, " hello ")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if msg.ID == 0 {
		t.Fatal("stored message ID should be set")
	}
	if msg.Body != "hello" {
		t.Fatalf("body = %q, want trimmed body", msg.Body)
	}

	event := assertNextEvent(t, subscription)
	if event.Kind != MessagePosted || event.Message.ID != msg.ID || event.Message.Body != "hello" {
		t.Fatalf("event = %+v, want posted message", event)
	}
}

func TestServiceJoinReplaysPersistedHistoryOldestFirst(t *testing.T) {
	service := NewService(newMemoryChatStore())
	room, err := service.CreateRoom(context.Background(), "user-1", "Room")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	for i := 0; i < historyLimit+1; i++ {
		if _, err := service.Post(context.Background(), room.ID, Member{ID: "user-1", Name: "user"}, fmt.Sprintf("message-%02d", i)); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
	}

	subscription, err := service.JoinRoom(context.Background(), room.ID, Member{ID: "user-1", Name: "user"})
	if err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	defer subscription.Close()

	for i := 1; i < historyLimit+1; i++ {
		event := assertNextEvent(t, subscription)
		if event.Message.Body != fmt.Sprintf("message-%02d", i) {
			t.Fatalf("history body = %q, want message-%02d", event.Message.Body, i)
		}
	}
}

func assertNextEvent(t *testing.T, subscription *Subscription) Event {
	t.Helper()

	select {
	case event := <-subscription.Events():
		return event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

type memoryChatStore struct {
	rooms       map[RoomID]StoredRoom
	memberships map[RoomID]map[UserID]RoomRole
	messages    map[RoomID][]Message
	nextMessage MessageID
}

func newMemoryChatStore() *memoryChatStore {
	return &memoryChatStore{
		rooms:       map[RoomID]StoredRoom{},
		memberships: map[RoomID]map[UserID]RoomRole{},
		messages:    map[RoomID][]Message{},
		nextMessage: 1,
	}
}

func (s *memoryChatStore) CreateRoom(_ context.Context, room StoredRoom, owner UserID, role RoomRole) (RoomSummary, error) {
	s.rooms[room.ID] = room
	if s.memberships[room.ID] == nil {
		s.memberships[room.ID] = map[UserID]RoomRole{}
	}
	s.memberships[room.ID][owner] = role
	return RoomSummary{ID: room.ID, Title: room.Title, Role: role, CreatedAt: room.CreatedAt}, nil
}

func (s *memoryChatStore) ListRoomsForUser(_ context.Context, userID UserID) ([]RoomSummary, error) {
	var rooms []RoomSummary
	for roomID, members := range s.memberships {
		role, ok := members[userID]
		if !ok {
			continue
		}
		room := s.rooms[roomID]
		rooms = append(rooms, RoomSummary{ID: room.ID, Title: room.Title, Role: role, CreatedAt: room.CreatedAt})
	}
	return rooms, nil
}

func (s *memoryChatStore) IsRoomMember(_ context.Context, roomID RoomID, userID UserID) (bool, error) {
	_, ok := s.memberships[roomID][userID]
	return ok, nil
}

func (s *memoryChatStore) StoreMessage(_ context.Context, message Message) (Message, error) {
	message.ID = s.nextMessage
	s.nextMessage++
	s.messages[message.RoomID] = append(s.messages[message.RoomID], message)
	return message, nil
}

func (s *memoryChatStore) RecentMessages(_ context.Context, roomID RoomID, limit int) ([]Message, error) {
	messages := s.messages[roomID]
	start := max(0, len(messages)-limit)
	recent := make([]Message, len(messages[start:]))
	copy(recent, messages[start:])
	return recent, nil
}

var _ Store = (*memoryChatStore)(nil)
