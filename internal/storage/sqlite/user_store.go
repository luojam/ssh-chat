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

func (s *UserStore) FindUserBySSHKeyFingerprint(ctx context.Context, fingerprint string) (auth.User, error) {
	var user auth.User
	row := s.db.QueryRowContext(ctx,
		`SELECT users.id, users.username
		FROM ssh_keys
		JOIN users ON users.id = ssh_keys.user_id
		WHERE ssh_keys.fingerprint = ?`,
		fingerprint,
	)
	if err := row.Scan(&user.ID, &user.Username); errors.Is(err, sql.ErrNoRows) {
		return auth.User{}, auth.ErrSSHKeyNotFound
	} else if err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (s *UserStore) LinkSSHKey(ctx context.Context, key auth.SSHKey) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ssh_keys (id, user_id, public_key, fingerprint) VALUES (?, ?, ?, ?)`,
		key.ID, key.UserID, key.PublicKey, key.Fingerprint,
	)
	if err == nil {
		return nil
	}
	if !isUniqueConstraint(err) {
		return err
	}

	var existingUserID string
	row := s.db.QueryRowContext(ctx, `SELECT user_id FROM ssh_keys WHERE fingerprint = ?`, key.Fingerprint)
	if scanErr := row.Scan(&existingUserID); errors.Is(scanErr, sql.ErrNoRows) {
		return auth.ErrSSHKeyNotFound
	} else if scanErr != nil {
		return scanErr
	}
	if existingUserID == key.UserID {
		return nil
	}
	return auth.ErrSSHKeyAlreadyLinked
}

func (s *UserStore) DeleteAccount(ctx context.Context, userID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete owned rooms explicitly instead of relying only on foreign-key
	// cascades; SQLite enforces cascades per connection, and this keeps the
	// account-deletion invariant local to this transaction.
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages WHERE room_id IN (SELECT id FROM rooms WHERE created_by = ?)`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM room_memberships WHERE room_id IN (SELECT id FROM rooms WHERE created_by = ?)`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM rooms WHERE created_by = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE messages SET author_user_id = NULL, author_name = 'deleted user' WHERE author_user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM room_memberships WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ssh_keys WHERE user_id = ?`, userID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return auth.ErrUserNotFound
	}
	return tx.Commit()
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
