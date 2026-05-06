package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/luojam/ssh-chat/internal/auth"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, user auth.StoredUser) (auth.User, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash,
	)
	if isUniqueConstraint(err) {
		return auth.User{}, auth.ErrUsernameTaken
	}
	if err != nil {
		return auth.User{}, err
	}
	return user.User, nil
}

func (s *UserStore) FindByUsername(ctx context.Context, username string) (auth.StoredUser, error) {
	var user auth.StoredUser
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE username = ?`,
		username,
	)
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); errors.Is(err, sql.ErrNoRows) {
		return auth.StoredUser{}, auth.ErrUserNotFound
	} else if err != nil {
		return auth.StoredUser{}, err
	}
	return user, nil
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
