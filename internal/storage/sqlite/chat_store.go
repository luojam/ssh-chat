package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luojam/ssh-chat/internal/chat"
)

const sqliteTimeFormat = time.RFC3339Nano

type ChatStore struct {
	db *sql.DB
}

func NewChatStore(db *sql.DB) *ChatStore {
	return &ChatStore{db: db}
}

func (s *ChatStore) CreateRoom(ctx context.Context, room chat.StoredRoom, owner chat.UserID, role chat.RoomRole) (chat.RoomSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return chat.RoomSummary{}, err
	}
	defer tx.Rollback()

	createdAt := formatTime(room.CreatedAt)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rooms (id, title, join_code, created_by, created_at) VALUES (?, ?, ?, ?, ?)`,
		room.ID, room.Title, room.JoinCode, room.CreatedBy, createdAt,
	); err != nil {
		return chat.RoomSummary{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO room_memberships (room_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		room.ID, owner, role, createdAt,
	); err != nil {
		return chat.RoomSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return chat.RoomSummary{}, err
	}

	return chat.RoomSummary{
		ID:        room.ID,
		Title:     room.Title,
		JoinCode:  room.JoinCode,
		Role:      role,
		CreatedAt: room.CreatedAt,
	}, nil
}

func (s *ChatStore) ListRoomsForUser(ctx context.Context, userID chat.UserID) ([]chat.RoomSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT rooms.id, rooms.title, rooms.join_code, room_memberships.role, rooms.created_at
		FROM room_memberships
		JOIN rooms ON rooms.id = room_memberships.room_id
		WHERE room_memberships.user_id = ?
		ORDER BY rooms.created_at DESC, rooms.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []chat.RoomSummary
	for rows.Next() {
		var room chat.RoomSummary
		var role string
		var createdAt string
		var joinCode string
		if err := rows.Scan(&room.ID, &room.Title, &joinCode, &role, &createdAt); err != nil {
			return nil, err
		}
		parsedRole, err := chat.ParseRoomRole(role)
		if err != nil {
			return nil, err
		}
		room.Role = parsedRole
		if parsedRole == chat.RoomRoleOwner {
			room.JoinCode = joinCode
		}
		room.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rooms, nil
}

func (s *ChatStore) JoinRoomByCode(ctx context.Context, joinCode string, userID chat.UserID, role chat.RoomRole) (chat.RoomSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return chat.RoomSummary{}, err
	}
	defer tx.Rollback()

	var room chat.StoredRoom
	var createdAt string
	row := tx.QueryRowContext(ctx,
		`SELECT id, title, join_code, created_by, created_at FROM rooms WHERE join_code = ?`,
		joinCode,
	)
	if err := row.Scan(&room.ID, &room.Title, &room.JoinCode, &room.CreatedBy, &createdAt); errors.Is(err, sql.ErrNoRows) {
		return chat.RoomSummary{}, chat.ErrRoomNotFound
	} else if err != nil {
		return chat.RoomSummary{}, err
	}
	room.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return chat.RoomSummary{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO room_memberships (room_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		room.ID, userID, role, formatTime(time.Now().UTC()),
	); err != nil {
		return chat.RoomSummary{}, err
	}

	var storedRole string
	row = tx.QueryRowContext(ctx,
		`SELECT role FROM room_memberships WHERE room_id = ? AND user_id = ?`,
		room.ID, userID,
	)
	if err := row.Scan(&storedRole); err != nil {
		return chat.RoomSummary{}, err
	}
	parsedRole, err := chat.ParseRoomRole(storedRole)
	if err != nil {
		return chat.RoomSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return chat.RoomSummary{}, err
	}

	summary := chat.RoomSummary{ID: room.ID, Title: room.Title, Role: parsedRole, CreatedAt: room.CreatedAt}
	if parsedRole == chat.RoomRoleOwner {
		summary.JoinCode = room.JoinCode
	}
	return summary, nil
}

func (s *ChatStore) DeleteRoom(ctx context.Context, roomID chat.RoomID, requester chat.UserID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	row := tx.QueryRowContext(ctx, `SELECT 1 FROM rooms WHERE id = ?`, roomID)
	if err := row.Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return chat.ErrRoomNotFound
	} else if err != nil {
		return err
	}

	var role string
	row = tx.QueryRowContext(ctx,
		`SELECT role FROM room_memberships WHERE room_id = ? AND user_id = ?`,
		roomID, requester,
	)
	if err := row.Scan(&role); errors.Is(err, sql.ErrNoRows) {
		return chat.ErrNotRoomOwner
	} else if err != nil {
		return err
	}
	parsedRole, err := chat.ParseRoomRole(role)
	if err != nil {
		return err
	}
	if parsedRole != chat.RoomRoleOwner {
		return chat.ErrNotRoomOwner
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM rooms WHERE id = ?`, roomID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ChatStore) IsRoomMember(ctx context.Context, roomID chat.RoomID, userID chat.UserID) (bool, error) {
	var exists int
	row := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM room_memberships WHERE room_id = ? AND user_id = ?`,
		roomID, userID,
	)
	if err := row.Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (s *ChatStore) StoreMessage(ctx context.Context, message chat.Message) (chat.Message, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (room_id, author_user_id, author_name, body, created_at) VALUES (?, ?, ?, ?, ?)`,
		message.RoomID, message.Author.ID, message.Author.Name, message.Body, formatTime(message.CreatedAt),
	)
	if err != nil {
		return chat.Message{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return chat.Message{}, err
	}
	message.ID = chat.MessageID(id)
	return message, nil
}

func (s *ChatStore) RecentMessages(ctx context.Context, roomID chat.RoomID, limit int) ([]chat.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, room_id, author_user_id, author_name, body, created_at
		FROM (
			SELECT id, room_id, author_user_id, author_name, body, created_at
			FROM messages
			WHERE room_id = ?
			ORDER BY id DESC
			LIMIT ?
		)
		ORDER BY id ASC`,
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []chat.Message
	for rows.Next() {
		var msg chat.Message
		var createdAt string
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.Author.ID, &msg.Author.Name, &msg.Body, &createdAt); err != nil {
			return nil, err
		}
		parsed, err := parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		msg.CreatedAt = parsed
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeFormat)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(sqliteTimeFormat, value)
}
