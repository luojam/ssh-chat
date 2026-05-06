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
