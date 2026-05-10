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

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv(envHost, "")
	t.Setenv(envPort, "")
	t.Setenv(envHostKeyPath, "")
	t.Setenv(envSQLitePath, "")

	config := loadConfig()
	if config.Host != defaultHost {
		t.Fatalf("Host = %q, want %q", config.Host, defaultHost)
	}
	if config.Port != defaultPort {
		t.Fatalf("Port = %q, want %q", config.Port, defaultPort)
	}
	if config.HostKeyPath != defaultHostKeyPath {
		t.Fatalf("HostKeyPath = %q, want %q", config.HostKeyPath, defaultHostKeyPath)
	}
	if config.SQLitePath != defaultSQLitePath {
		t.Fatalf("SQLitePath = %q, want %q", config.SQLitePath, defaultSQLitePath)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv(envHost, "127.0.0.1")
	t.Setenv(envPort, "2022")
	t.Setenv(envHostKeyPath, "/tmp/ssh_host_key")
	t.Setenv(envSQLitePath, "/tmp/ssh-chat.sqlite")

	config := loadConfig()
	if config.Host != "127.0.0.1" || config.Port != "2022" || config.HostKeyPath != "/tmp/ssh_host_key" || config.SQLitePath != "/tmp/ssh-chat.sqlite" {
		t.Fatalf("loadConfig() = %+v", config)
	}
}
