package auth

import (
	"context"
	"errors"
	"testing"
)

func TestPasswordServiceSignupAndLogin(t *testing.T) {
	service := NewPasswordService(newMemoryStore())

	user, err := service.Signup(context.Background(), " alice ", "secret", "secret")
	if err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}
	if user.ID == "" || user.Username != "alice" {
		t.Fatalf("created user = %+v, want normalized username and generated id", user)
	}

	loggedIn, err := service.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if loggedIn != user {
		t.Fatalf("logged in user = %+v, want %+v", loggedIn, user)
	}
}

func TestPasswordServiceSignupValidation(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		confirm  string
		wantErr  error
	}{
		{name: "empty username", username: " ", password: "secret", confirm: "secret", wantErr: ErrInvalidInput},
		{name: "empty password", username: "alice", password: "", confirm: "", wantErr: ErrInvalidInput},
		{name: "password mismatch", username: "alice", password: "secret", confirm: "different", wantErr: ErrPasswordMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPasswordService(newMemoryStore())
			_, err := service.Signup(context.Background(), tt.username, tt.password, tt.confirm)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Signup error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordServiceRejectsDuplicateUsername(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	if _, err := service.Signup(context.Background(), "alice", "secret", "secret"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}
	_, err := service.Signup(context.Background(), "alice", "secret", "secret")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate Signup error = %v, want %v", err, ErrUsernameTaken)
	}
}

func TestPasswordServiceLoginValidation(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	if _, err := service.Signup(context.Background(), "alice", "secret", "secret"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "unknown username", username: "unknown", password: "secret"},
		{name: "wrong password", username: "alice", password: "wrong"},
		{name: "empty username", username: "", password: "secret"},
		{name: "empty password", username: "alice", password: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Login(context.Background(), tt.username, tt.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login error = %v, want %v", err, ErrInvalidCredentials)
			}
		})
	}
}

func TestPasswordServiceFindUserBySSHKeyFingerprint(t *testing.T) {
	store := newMemoryStore()
	service := NewPasswordService(store)
	user := StoredUser{User: User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"}
	if _, err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if err := service.LinkSSHKey(context.Background(), user.User, "ssh-ed25519 AAAA", " SHA256:abc "); err != nil {
		t.Fatalf("LinkSSHKey returned error: %v", err)
	}

	found, err := service.FindUserBySSHKeyFingerprint(context.Background(), "SHA256:abc")
	if err != nil {
		t.Fatalf("FindUserBySSHKeyFingerprint returned error: %v", err)
	}
	if found != user.User {
		t.Fatalf("found user = %+v, want %+v", found, user.User)
	}
}

func TestPasswordServiceSSHKeyLookupMissesCleanly(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	_, err := service.FindUserBySSHKeyFingerprint(context.Background(), "missing")
	if !errors.Is(err, ErrSSHKeyNotFound) {
		t.Fatalf("FindUserBySSHKeyFingerprint error = %v, want %v", err, ErrSSHKeyNotFound)
	}

	_, err = service.FindUserBySSHKeyFingerprint(context.Background(), " ")
	if !errors.Is(err, ErrSSHKeyNotFound) {
		t.Fatalf("empty fingerprint error = %v, want %v", err, ErrSSHKeyNotFound)
	}
}

func TestPasswordServiceLinkSSHKeyValidation(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	tests := []struct {
		name        string
		user        User
		publicKey   string
		fingerprint string
	}{
		{name: "missing user", publicKey: "ssh-ed25519 AAAA", fingerprint: "SHA256:abc"},
		{name: "missing public key", user: User{ID: "user_1"}, fingerprint: "SHA256:abc"},
		{name: "missing fingerprint", user: User{ID: "user_1"}, publicKey: "ssh-ed25519 AAAA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.LinkSSHKey(context.Background(), tt.user, tt.publicKey, tt.fingerprint)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("LinkSSHKey error = %v, want %v", err, ErrInvalidInput)
			}
		})
	}
}

func TestPasswordServiceDuplicateSameUserSSHKeyIsHarmless(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	user := User{ID: "user_1", Username: "alice"}
	if err := service.LinkSSHKey(context.Background(), user, "ssh-ed25519 AAAA", "SHA256:abc"); err != nil {
		t.Fatalf("first LinkSSHKey returned error: %v", err)
	}
	if err := service.LinkSSHKey(context.Background(), user, "ssh-ed25519 AAAA", "SHA256:abc"); err != nil {
		t.Fatalf("duplicate same-user LinkSSHKey returned error: %v", err)
	}
}

func TestPasswordServiceRejectsSSHKeyLinkedToAnotherUser(t *testing.T) {
	service := NewPasswordService(newMemoryStore())
	if err := service.LinkSSHKey(context.Background(), User{ID: "user_1"}, "ssh-ed25519 AAAA", "SHA256:abc"); err != nil {
		t.Fatalf("first LinkSSHKey returned error: %v", err)
	}
	err := service.LinkSSHKey(context.Background(), User{ID: "user_2"}, "ssh-ed25519 AAAA", "SHA256:abc")
	if !errors.Is(err, ErrSSHKeyAlreadyLinked) {
		t.Fatalf("cross-user LinkSSHKey error = %v, want %v", err, ErrSSHKeyAlreadyLinked)
	}
}

func TestPasswordServiceDeleteAccountRemovesUserAndSSHKeys(t *testing.T) {
	store := newMemoryStore()
	service := NewPasswordService(store)
	user, err := store.CreateUser(context.Background(), StoredUser{User: User{ID: "user_1", Username: "alice"}, PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if err := service.LinkSSHKey(context.Background(), user, "ssh-ed25519 AAAA", "SHA256:abc"); err != nil {
		t.Fatalf("LinkSSHKey returned error: %v", err)
	}

	if err := service.DeleteAccount(context.Background(), user); err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
	if _, err := store.FindByUsername(context.Background(), "alice"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("FindByUsername after delete error = %v, want %v", err, ErrUserNotFound)
	}
	if _, err := store.FindUserBySSHKeyFingerprint(context.Background(), "SHA256:abc"); !errors.Is(err, ErrSSHKeyNotFound) {
		t.Fatalf("FindUserBySSHKeyFingerprint after delete error = %v, want %v", err, ErrSSHKeyNotFound)
	}
}

type memoryStore struct {
	usersByUsername map[string]StoredUser
	usersByID       map[string]StoredUser
	keys            map[string]SSHKey
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		usersByUsername: map[string]StoredUser{},
		usersByID:       map[string]StoredUser{},
		keys:            map[string]SSHKey{},
	}
}

func (s *memoryStore) CreateUser(_ context.Context, user StoredUser) (User, error) {
	if _, ok := s.usersByUsername[user.Username]; ok {
		return User{}, ErrUsernameTaken
	}
	s.usersByUsername[user.Username] = user
	s.usersByID[user.ID] = user
	return user.User, nil
}

func (s *memoryStore) FindByUsername(_ context.Context, username string) (StoredUser, error) {
	user, ok := s.usersByUsername[username]
	if !ok {
		return StoredUser{}, ErrUserNotFound
	}
	return user, nil
}

func (s *memoryStore) FindUserBySSHKeyFingerprint(_ context.Context, fingerprint string) (User, error) {
	key, ok := s.keys[fingerprint]
	if !ok {
		return User{}, ErrSSHKeyNotFound
	}
	user, ok := s.usersByID[key.UserID]
	if !ok {
		return User{ID: key.UserID}, nil
	}
	return user.User, nil
}

func (s *memoryStore) FindSSHKeyByUserID(_ context.Context, userID string) (SSHKey, error) {
	for _, key := range s.keys {
		if key.UserID == userID {
			return key, nil
		}
	}
	return SSHKey{}, ErrSSHKeyNotFound
}

func (s *memoryStore) LinkSSHKey(_ context.Context, key SSHKey) error {
	existing, ok := s.keys[key.Fingerprint]
	if ok {
		if existing.UserID == key.UserID {
			return nil
		}
		return ErrSSHKeyAlreadyLinked
	}
	for fingerprint, linked := range s.keys {
		if linked.UserID == key.UserID {
			delete(s.keys, fingerprint)
		}
	}
	s.keys[key.Fingerprint] = key
	return nil
}

func (s *memoryStore) DeleteSSHKey(_ context.Context, userID, fingerprint string) error {
	key, ok := s.keys[fingerprint]
	if !ok || key.UserID != userID {
		return ErrSSHKeyNotFound
	}
	delete(s.keys, fingerprint)
	return nil
}

func (s *memoryStore) DeleteAccount(_ context.Context, userID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return ErrUserNotFound
	}
	delete(s.usersByID, userID)
	delete(s.usersByUsername, user.Username)
	for fingerprint, key := range s.keys {
		if key.UserID == userID {
			delete(s.keys, fingerprint)
		}
	}
	return nil
}
