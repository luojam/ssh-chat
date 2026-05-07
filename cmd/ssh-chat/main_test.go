package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func TestSSHKeyIdentityFromPublicKey(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey returned error: %v", err)
	}

	publicKey, fingerprint := sshKeyIdentityFromPublicKey(sshPub)
	if !strings.HasPrefix(publicKey, "ssh-ed25519 ") {
		t.Fatalf("public key = %q, want authorized_keys ed25519 format", publicKey)
	}
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Fatalf("fingerprint = %q, want SHA256 fingerprint", fingerprint)
	}
}

func TestSSHKeyIdentityFromNilPublicKey(t *testing.T) {
	publicKey, fingerprint := sshKeyIdentityFromPublicKey(nil)
	if publicKey != "" || fingerprint != "" {
		t.Fatalf("nil key identity = %q, %q; want empty", publicKey, fingerprint)
	}
}

func TestNewSessionMemberIDCreatesOpaqueSessionID(t *testing.T) {
	first, err := newSessionMemberID()
	if err != nil {
		t.Fatalf("newSessionMemberID returned error: %v", err)
	}
	second, err := newSessionMemberID()
	if err != nil {
		t.Fatalf("newSessionMemberID returned error: %v", err)
	}

	if first == "" {
		t.Fatal("first ID should not be empty")
	}
	if second == "" {
		t.Fatal("second ID should not be empty")
	}
	if first == second {
		t.Fatalf("IDs should differ, got %q twice", first)
	}
}
