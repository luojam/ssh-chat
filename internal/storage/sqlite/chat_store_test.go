package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/luojam/ssh-chat/internal/chat"
)

func TestChatStoreCreateListAndMessages(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir()+"/chat.sqlite")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()
	insertTestUser(t, db, "user-1", "alice")
	insertTestUser(t, db, "user-2", "bob")

	store := NewChatStore(db)
	createdAt := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	room, err := store.CreateRoom(ctx, chat.StoredRoom{
		ID:        "room-1",
		Title:     "Room One",
		JoinCode:  "7KQ9M2XP",
		CreatedBy: "user-1",
		CreatedAt: createdAt,
	}, "user-1", chat.RoomRoleOwner)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if room.Role != chat.RoomRoleOwner || room.Title != "Room One" || room.JoinCode != "7KQ9M2XP" {
		t.Fatalf("room summary = %+v, want owner Room One with join code", room)
	}

	rooms, err := store.ListRoomsForUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListRoomsForUser returned error: %v", err)
	}
	if len(rooms) != 1 || rooms[0].ID != "room-1" || rooms[0].JoinCode != "7KQ9M2XP" {
		t.Fatalf("rooms = %+v, want room-1 with owner join code", rooms)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO room_memberships (room_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		"room-1", "user-2", chat.RoomRoleMember, formatTime(createdAt),
	)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	otherRooms, err := store.ListRoomsForUser(ctx, "user-2")
	if err != nil {
		t.Fatalf("ListRoomsForUser other returned error: %v", err)
	}
	if len(otherRooms) != 1 || otherRooms[0].JoinCode != "" {
		t.Fatalf("other user rooms = %+v, want redacted member join code", otherRooms)
	}

	joined, err := store.JoinRoomByCode(ctx, "7KQ9M2XP", "user-2", chat.RoomRoleMember)
	if err != nil {
		t.Fatalf("JoinRoomByCode existing member returned error: %v", err)
	}
	if joined.ID != "room-1" || joined.Role != chat.RoomRoleMember || joined.JoinCode != "" {
		t.Fatalf("joined summary = %+v, want member without join code", joined)
	}
	ownerJoined, err := store.JoinRoomByCode(ctx, "7KQ9M2XP", "user-1", chat.RoomRoleMember)
	if err != nil {
		t.Fatalf("JoinRoomByCode owner returned error: %v", err)
	}
	if ownerJoined.Role != chat.RoomRoleOwner || ownerJoined.JoinCode != "7KQ9M2XP" {
		t.Fatalf("owner joined summary = %+v, want owner with join code", ownerJoined)
	}

	isMember, err := store.IsRoomMember(ctx, "room-1", "user-1")
	if err != nil {
		t.Fatalf("IsRoomMember returned error: %v", err)
	}
	if !isMember {
		t.Fatal("creator should be room member")
	}

	first, err := store.StoreMessage(ctx, chat.Message{
		RoomID:    "room-1",
		Author:    chat.Member{ID: "user-1", Name: "alice"},
		Body:      "first",
		CreatedAt: createdAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("StoreMessage first returned error: %v", err)
	}
	second, err := store.StoreMessage(ctx, chat.Message{
		RoomID:    "room-1",
		Author:    chat.Member{ID: "user-1", Name: "alice"},
		Body:      "second",
		CreatedAt: createdAt.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StoreMessage second returned error: %v", err)
	}
	if first.ID == 0 || second.ID <= first.ID {
		t.Fatalf("message IDs = %d, %d; want increasing storage IDs", first.ID, second.ID)
	}

	recent, err := store.RecentMessages(ctx, "room-1", 1)
	if err != nil {
		t.Fatalf("RecentMessages returned error: %v", err)
	}
	if len(recent) != 1 || recent[0].Body != "second" {
		t.Fatalf("recent = %+v, want newest limited message", recent)
	}
}

func insertTestUser(t *testing.T, execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, id, username string) {
	t.Helper()
	_, err := execer.ExecContext(context.Background(),
		`INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
		id, username, "hash",
	)
	if err != nil {
		t.Fatalf("insert user %s: %v", id, err)
	}
}
