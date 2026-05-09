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
	if len(room.JoinCode) != joinCodeLength {
		t.Fatalf("join code = %q, want %d characters", room.JoinCode, joinCodeLength)
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

func TestServiceJoinRoomByCodeAddsMemberAndRedactsCode(t *testing.T) {
	store := newMemoryChatStore()
	service := NewService(store)

	room, err := service.CreateRoom(context.Background(), "owner", "Room")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	joined, err := service.JoinRoomByCode(context.Background(), " "+FormatJoinCode(room.JoinCode)+" ", Member{ID: "user-2", Name: "bob"})
	if err != nil {
		t.Fatalf("JoinRoomByCode returned error: %v", err)
	}
	if joined.ID != room.ID || joined.Role != RoomRoleMember || joined.JoinCode != "" {
		t.Fatalf("joined summary = %+v, want member without join code", joined)
	}

	subscription, err := service.JoinRoom(context.Background(), room.ID, Member{ID: "user-2", Name: "bob"})
	if err != nil {
		t.Fatalf("joined user should be able to enter room: %v", err)
	}
	subscription.Close()
}

func TestServiceJoinRoomByCodeOwnerIsIdempotent(t *testing.T) {
	service := NewService(newMemoryChatStore())

	room, err := service.CreateRoom(context.Background(), "owner", "Room")
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	joined, err := service.JoinRoomByCode(context.Background(), room.JoinCode, Member{ID: "owner", Name: "alice"})
	if err != nil {
		t.Fatalf("JoinRoomByCode returned error: %v", err)
	}
	if joined.Role != RoomRoleOwner || joined.JoinCode != room.JoinCode {
		t.Fatalf("owner joined summary = %+v, want owner with join code", joined)
	}
}

func TestServiceJoinRoomByCodeValidatesCode(t *testing.T) {
	service := NewService(newMemoryChatStore())

	_, err := service.JoinRoomByCode(context.Background(), "bad", Member{ID: "user-1"})
	if !errors.Is(err, ErrInvalidJoinCode) {
		t.Fatalf("short code error = %v, want ErrInvalidJoinCode", err)
	}
	_, err = service.JoinRoomByCode(context.Background(), "ABCDEFGH", Member{ID: "user-1"})
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("unknown code error = %v, want ErrRoomNotFound", err)
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
	return RoomSummary{ID: room.ID, Title: room.Title, JoinCode: room.JoinCode, Role: role, CreatedAt: room.CreatedAt}, nil
}

func (s *memoryChatStore) ListRoomsForUser(_ context.Context, userID UserID) ([]RoomSummary, error) {
	var rooms []RoomSummary
	for roomID, members := range s.memberships {
		role, ok := members[userID]
		if !ok {
			continue
		}
		room := s.rooms[roomID]
		summary := RoomSummary{ID: room.ID, Title: room.Title, Role: role, CreatedAt: room.CreatedAt}
		if role == RoomRoleOwner {
			summary.JoinCode = room.JoinCode
		}
		rooms = append(rooms, summary)
	}
	return rooms, nil
}

func (s *memoryChatStore) JoinRoomByCode(_ context.Context, joinCode string, userID UserID, role RoomRole) (RoomSummary, error) {
	for roomID, room := range s.rooms {
		if room.JoinCode != joinCode {
			continue
		}
		if s.memberships[roomID] == nil {
			s.memberships[roomID] = map[UserID]RoomRole{}
		}
		if _, ok := s.memberships[roomID][userID]; !ok {
			s.memberships[roomID][userID] = role
		}
		storedRole := s.memberships[roomID][userID]
		summary := RoomSummary{ID: room.ID, Title: room.Title, Role: storedRole, CreatedAt: room.CreatedAt}
		if storedRole == RoomRoleOwner {
			summary.JoinCode = room.JoinCode
		}
		return summary, nil
	}
	return RoomSummary{}, ErrRoomNotFound
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
