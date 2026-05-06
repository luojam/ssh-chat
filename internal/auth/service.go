package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

type Store interface {
	CreateUser(ctx context.Context, user StoredUser) (User, error)
	FindByUsername(ctx context.Context, username string) (StoredUser, error)
}

type Service interface {
	Signup(ctx context.Context, username, password, confirmPassword string) (User, error)
	Login(ctx context.Context, username, password string) (User, error)
}

type PasswordService struct {
	store Store
}

func NewPasswordService(store Store) *PasswordService {
	return &PasswordService{store: store}
}

func (s *PasswordService) Signup(ctx context.Context, username, password, confirmPassword string) (User, error) {
	username = normalizeUsername(username)
	if username == "" || password == "" {
		return User{}, ErrInvalidInput
	}
	if password != confirmPassword {
		return User{}, ErrPasswordMismatch
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return User{}, err
	}

	userID, err := newUserID()
	if err != nil {
		return User{}, err
	}

	user := StoredUser{
		User: User{
			ID:       userID,
			Username: username,
		},
		PasswordHash: string(hash),
	}
	created, err := s.store.CreateUser(ctx, user)
	if errors.Is(err, ErrUsernameTaken) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, err
	}
	return created, nil
}

func (s *PasswordService) Login(ctx context.Context, username, password string) (User, error) {
	username = normalizeUsername(username)
	if username == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}

	stored, err := s.store.FindByUsername(ctx, username)
	if errors.Is(err, ErrUserNotFound) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return stored.User, nil
}

func normalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

func newUserID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "user_" + hex.EncodeToString(b[:]), nil
}
