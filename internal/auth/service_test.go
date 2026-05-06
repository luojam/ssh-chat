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

type memoryStore struct {
	users map[string]StoredUser
}

func newMemoryStore() *memoryStore {
	return &memoryStore{users: map[string]StoredUser{}}
}

func (s *memoryStore) CreateUser(_ context.Context, user StoredUser) (User, error) {
	if _, ok := s.users[user.Username]; ok {
		return User{}, ErrUsernameTaken
	}
	s.users[user.Username] = user
	return user.User, nil
}

func (s *memoryStore) FindByUsername(_ context.Context, username string) (StoredUser, error) {
	user, ok := s.users[username]
	if !ok {
		return StoredUser{}, ErrUserNotFound
	}
	return user, nil
}
