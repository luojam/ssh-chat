package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/luojam/ssh-chat/internal/auth"
)

func TestUserStoreCreateAndFindByUsername(t *testing.T) {
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()

	store := NewUserStore(db)
	created, err := store.CreateUser(context.Background(), auth.StoredUser{
		User:         auth.User{ID: "user_1", Username: "alice"},
		PasswordHash: "$2a$10$hash",
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if created.ID != "user_1" || created.Username != "alice" {
		t.Fatalf("created user = %+v", created)
	}

	found, err := store.FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("FindByUsername returned error: %v", err)
	}
	if found.ID != "user_1" || found.Username != "alice" || found.PasswordHash != "$2a$10$hash" {
		t.Fatalf("found user = %+v", found)
	}
}

func TestUserStoreRejectsDuplicateUsername(t *testing.T) {
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()

	store := NewUserStore(db)
	user := auth.StoredUser{User: auth.User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"}
	if _, err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	_, err = store.CreateUser(context.Background(), auth.StoredUser{User: auth.User{ID: "user_2", Username: "alice"}, PasswordHash: "hash"})
	if !errors.Is(err, auth.ErrUsernameTaken) {
		t.Fatalf("duplicate CreateUser error = %v, want %v", err, auth.ErrUsernameTaken)
	}
}

func TestUserStoreFindMissingUser(t *testing.T) {
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()

	_, err = NewUserStore(db).FindByUsername(context.Background(), "missing")
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("FindByUsername error = %v, want %v", err, auth.ErrUserNotFound)
	}
}

func TestPasswordServiceDoesNotPersistPlaintextPassword(t *testing.T) {
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()

	service := auth.NewPasswordService(NewUserStore(db))
	if _, err := service.Signup(context.Background(), "alice", "secret", "secret"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	stored, err := NewUserStore(db).FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("FindByUsername returned error: %v", err)
	}
	if stored.PasswordHash == "secret" || strings.Contains(stored.PasswordHash, "secret") {
		t.Fatalf("password hash should not contain plaintext password, got %q", stored.PasswordHash)
	}
}

func TestUserStoreLinkSSHKeyAndFindUserByFingerprint(t *testing.T) {
	store, closeDB := newTestUserStore(t)
	defer closeDB()
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"})

	if err := store.LinkSSHKey(context.Background(), auth.SSHKey{
		ID:          "key_1",
		UserID:      "user_1",
		PublicKey:   "ssh-ed25519 AAAA",
		Fingerprint: "SHA256:abc",
	}); err != nil {
		t.Fatalf("LinkSSHKey returned error: %v", err)
	}

	found, err := store.FindUserBySSHKeyFingerprint(context.Background(), "SHA256:abc")
	if err != nil {
		t.Fatalf("FindUserBySSHKeyFingerprint returned error: %v", err)
	}
	if found.ID != "user_1" || found.Username != "alice" {
		t.Fatalf("found user = %+v, want alice", found)
	}
}

func TestUserStoreDuplicateSameUserSSHKeyIsHarmless(t *testing.T) {
	store, closeDB := newTestUserStore(t)
	defer closeDB()
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"})
	key := auth.SSHKey{ID: "key_1", UserID: "user_1", PublicKey: "ssh-ed25519 AAAA", Fingerprint: "SHA256:abc"}
	if err := store.LinkSSHKey(context.Background(), key); err != nil {
		t.Fatalf("first LinkSSHKey returned error: %v", err)
	}
	key.ID = "key_2"
	if err := store.LinkSSHKey(context.Background(), key); err != nil {
		t.Fatalf("duplicate same-user LinkSSHKey returned error: %v", err)
	}
}

func TestUserStoreRejectsDuplicateSSHKeyForAnotherUser(t *testing.T) {
	store, closeDB := newTestUserStore(t)
	defer closeDB()
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"})
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_2", Username: "bob"}, PasswordHash: "hash"})
	if err := store.LinkSSHKey(context.Background(), auth.SSHKey{ID: "key_1", UserID: "user_1", PublicKey: "ssh-ed25519 AAAA", Fingerprint: "SHA256:abc"}); err != nil {
		t.Fatalf("first LinkSSHKey returned error: %v", err)
	}

	err := store.LinkSSHKey(context.Background(), auth.SSHKey{ID: "key_2", UserID: "user_2", PublicKey: "ssh-ed25519 AAAA", Fingerprint: "SHA256:abc"})
	if !errors.Is(err, auth.ErrSSHKeyAlreadyLinked) {
		t.Fatalf("cross-user LinkSSHKey error = %v, want %v", err, auth.ErrSSHKeyAlreadyLinked)
	}
}

func TestUserStoreFindMissingSSHKey(t *testing.T) {
	store, closeDB := newTestUserStore(t)
	defer closeDB()

	_, err := store.FindUserBySSHKeyFingerprint(context.Background(), "missing")
	if !errors.Is(err, auth.ErrSSHKeyNotFound) {
		t.Fatalf("FindUserBySSHKeyFingerprint error = %v, want %v", err, auth.ErrSSHKeyNotFound)
	}
}

func TestUserStoreDeleteAccountDeletesOwnedRoomsAndAnonymizesRemainingMessages(t *testing.T) {
	store, closeDB := newTestUserStore(t)
	defer closeDB()
	ctx := context.Background()
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"})
	createTestUser(t, store, auth.StoredUser{User: auth.User{ID: "user_2", Username: "bob"}, PasswordHash: "hash"})
	if err := store.LinkSSHKey(ctx, auth.SSHKey{ID: "key_1", UserID: "user_1", PublicKey: "ssh-ed25519 AAAA", Fingerprint: "SHA256:abc"}); err != nil {
		t.Fatalf("LinkSSHKey returned error: %v", err)
	}
	mustExec(t, store, `INSERT INTO rooms (id, title, join_code, created_by, created_at) VALUES ('owned', 'Owned', 'OWNED123', 'user_1', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO rooms (id, title, join_code, created_by, created_at) VALUES ('other', 'Other', 'OTHER123', 'user_2', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO room_memberships (room_id, user_id, role, joined_at) VALUES ('owned', 'user_1', 'owner', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO room_memberships (room_id, user_id, role, joined_at) VALUES ('other', 'user_2', 'owner', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO room_memberships (room_id, user_id, role, joined_at) VALUES ('other', 'user_1', 'member', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO messages (room_id, author_user_id, author_name, body, created_at) VALUES ('owned', 'user_1', 'alice', 'owned room message', '2026-01-01T00:00:00Z')`)
	mustExec(t, store, `INSERT INTO messages (room_id, author_user_id, author_name, body, created_at) VALUES ('other', 'user_1', 'alice', 'other room message', '2026-01-01T00:00:00Z')`)

	if err := store.DeleteAccount(ctx, "user_1"); err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
	assertCount(t, store, `SELECT COUNT(*) FROM users WHERE id = 'user_1'`, 0)
	assertCount(t, store, `SELECT COUNT(*) FROM ssh_keys WHERE user_id = 'user_1'`, 0)
	assertCount(t, store, `SELECT COUNT(*) FROM rooms WHERE id = 'owned'`, 0)
	assertCount(t, store, `SELECT COUNT(*) FROM messages WHERE body = 'owned room message'`, 0)
	assertCount(t, store, `SELECT COUNT(*) FROM room_memberships WHERE user_id = 'user_1'`, 0)
	assertCount(t, store, `SELECT COUNT(*) FROM rooms WHERE id = 'other'`, 1)

	var authorID *string
	var authorName string
	row := store.db.QueryRowContext(ctx, `SELECT author_user_id, author_name FROM messages WHERE body = 'other room message'`)
	if err := row.Scan(&authorID, &authorName); err != nil {
		t.Fatalf("query anonymized message returned error: %v", err)
	}
	if authorID != nil || authorName != "deleted user" {
		t.Fatalf("anonymized author id/name = %v/%q, want nil/deleted user", authorID, authorName)
	}
}

func newTestUserStore(t *testing.T) (*UserStore, func()) {
	t.Helper()
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	return NewUserStore(db), func() { _ = db.Close() }
}

func createTestUser(t *testing.T, store *UserStore, user auth.StoredUser) {
	t.Helper()
	if _, err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
}

func mustExec(t *testing.T, store *UserStore, statement string) {
	t.Helper()
	if _, err := store.db.ExecContext(context.Background(), statement); err != nil {
		t.Fatalf("exec %q returned error: %v", statement, err)
	}
}

func assertCount(t *testing.T, store *UserStore, query string, want int) {
	t.Helper()
	var got int
	if err := store.db.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("count query %q returned error: %v", query, err)
	}
	if got != want {
		t.Fatalf("count query %q = %d, want %d", query, got, want)
	}
}
