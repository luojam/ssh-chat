package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

type Store interface {
	CreateRoom(ctx context.Context, room StoredRoom, owner UserID, role RoomRole) (RoomSummary, error)
	ListRoomsForUser(ctx context.Context, userID UserID) ([]RoomSummary, error)
	IsRoomMember(ctx context.Context, roomID RoomID, userID UserID) (bool, error)
	StoreMessage(ctx context.Context, message Message) (Message, error)
	RecentMessages(ctx context.Context, roomID RoomID, limit int) ([]Message, error)
}

type Service struct {
	store Store

	mu        sync.Mutex
	liveRooms map[RoomID]*liveRoom
}

func NewService(store Store) *Service {
	return &Service{
		store:     store,
		liveRooms: make(map[RoomID]*liveRoom),
	}
}

func (s *Service) CreateRoom(ctx context.Context, creator UserID, title string) (RoomSummary, error) {
	if creator == "" {
		return RoomSummary{}, ErrNotRoomMember
	}
	title = normalizeRoomTitle(title)
	if !validRoomTitle(title) {
		return RoomSummary{}, ErrInvalidRoomTitle
	}

	id, err := newRoomID()
	if err != nil {
		return RoomSummary{}, err
	}
	room := StoredRoom{
		ID:        id,
		Title:     title,
		CreatedBy: creator,
		CreatedAt: time.Now().UTC(),
	}
	return s.store.CreateRoom(ctx, room, creator, RoomRoleOwner)
}

func (s *Service) ListRoomsForUser(ctx context.Context, userID UserID) ([]RoomSummary, error) {
	if userID == "" {
		return nil, ErrNotRoomMember
	}
	return s.store.ListRoomsForUser(ctx, userID)
}

func (s *Service) JoinRoom(ctx context.Context, roomID RoomID, member Member) (*Subscription, error) {
	if roomID == "" || member.ID == "" {
		return nil, ErrNotRoomMember
	}
	ok, err := s.store.IsRoomMember(ctx, roomID, member.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotRoomMember
	}

	live := s.liveRoom(roomID)
	live.mu.Lock()
	defer live.mu.Unlock()

	history, err := s.store.RecentMessages(ctx, roomID, historyLimit)
	if err != nil {
		return nil, err
	}
	return live.joinLocked(member, history), nil
}

func (s *Service) Post(ctx context.Context, roomID RoomID, author Member, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}
	if roomID == "" || author.ID == "" {
		return Message{}, ErrNotRoomMember
	}
	ok, err := s.store.IsRoomMember(ctx, roomID, author.ID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrNotRoomMember
	}

	live := s.liveRoom(roomID)
	live.mu.Lock()
	defer live.mu.Unlock()

	msg, err := s.store.StoreMessage(ctx, Message{
		RoomID:    roomID,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return Message{}, err
	}
	live.broadcastLocked(Event{Kind: MessagePosted, Message: msg})
	return msg, nil
}

func (s *Service) liveRoom(roomID RoomID) *liveRoom {
	s.mu.Lock()
	defer s.mu.Unlock()

	live, ok := s.liveRooms[roomID]
	if !ok {
		live = newLiveRoom()
		s.liveRooms[roomID] = live
	}
	return live
}

func normalizeRoomTitle(title string) string {
	return strings.TrimSpace(title)
}

func validRoomTitle(title string) bool {
	return title != "" && len([]rune(title)) <= maxRoomTitleRunes
}

func newRoomID() (RoomID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return RoomID("room_" + hex.EncodeToString(b[:])), nil
}
