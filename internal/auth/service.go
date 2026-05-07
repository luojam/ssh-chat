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
	FindUserBySSHKeyFingerprint(ctx context.Context, fingerprint string) (User, error)
	LinkSSHKey(ctx context.Context, key SSHKey) error
}

type Service interface {
	Signup(ctx context.Context, username, password, confirmPassword string) (User, error)
	Login(ctx context.Context, username, password string) (User, error)
	FindUserBySSHKeyFingerprint(ctx context.Context, fingerprint string) (User, error)
	LinkSSHKey(ctx context.Context, user User, publicKey, fingerprint string) error
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

func (s *PasswordService) FindUserBySSHKeyFingerprint(ctx context.Context, fingerprint string) (User, error) {
	fingerprint = normalizeFingerprint(fingerprint)
	if fingerprint == "" {
		return User{}, ErrSSHKeyNotFound
	}
	return s.store.FindUserBySSHKeyFingerprint(ctx, fingerprint)
}

func (s *PasswordService) LinkSSHKey(ctx context.Context, user User, publicKey, fingerprint string) error {
	publicKey = strings.TrimSpace(publicKey)
	fingerprint = normalizeFingerprint(fingerprint)
	if user.ID == "" || publicKey == "" || fingerprint == "" {
		return ErrInvalidInput
	}

	keyID, err := newSSHKeyID()
	if err != nil {
		return err
	}
	return s.store.LinkSSHKey(ctx, SSHKey{
		ID:          keyID,
		UserID:      user.ID,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
	})
}

func normalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

func normalizeFingerprint(fingerprint string) string {
	return strings.TrimSpace(fingerprint)
}

func newUserID() (string, error) {
	return newOpaqueID("user_")
}

func newSSHKeyID() (string, error) {
	return newOpaqueID("sshkey_")
}

func newOpaqueID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}
