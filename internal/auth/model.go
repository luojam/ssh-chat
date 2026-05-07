package auth

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid auth input")
	ErrUsernameTaken       = errors.New("username already taken")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrPasswordMismatch    = errors.New("passwords do not match")
	ErrUserNotFound        = errors.New("user not found")
	ErrSSHKeyNotFound      = errors.New("ssh key not found")
	ErrSSHKeyAlreadyLinked = errors.New("ssh key already linked to another user")
)

type User struct {
	ID       string
	Username string
}

type StoredUser struct {
	User
	PasswordHash string
}

type SSHKey struct {
	ID          string
	UserID      string
	PublicKey   string
	Fingerprint string
}
