package session

import (
	"context"

	"github.com/luojam/ssh-chat/internal/chat"
)

const townSquareRoomID = "town-square"

type legacyRoomService struct {
	room *chat.Room
}

func (s legacyRoomService) CreateRoom(context.Context, chat.UserID, string) (chat.RoomSummary, error) {
	return chat.RoomSummary{}, chat.ErrInvalidRoomTitle
}

func (s legacyRoomService) ListRoomsForUser(context.Context, chat.UserID) ([]chat.RoomSummary, error) {
	return []chat.RoomSummary{{ID: townSquareRoomID, Title: "Town Square", Role: chat.RoomRoleOwner}}, nil
}

func (s legacyRoomService) JoinRoom(_ context.Context, roomID chat.RoomID, member chat.Member) (*chat.Subscription, error) {
	if s.room == nil || roomID != townSquareRoomID {
		return nil, chat.ErrRoomNotFound
	}
	return s.room.Join(member), nil
}

func (s legacyRoomService) Post(_ context.Context, roomID chat.RoomID, author chat.Member, body string) (chat.Message, error) {
	if s.room == nil || roomID != townSquareRoomID {
		return chat.Message{}, chat.ErrRoomNotFound
	}
	return s.room.Post(author, body)
}
