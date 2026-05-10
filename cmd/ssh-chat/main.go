package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
	"github.com/luojam/ssh-chat/internal/auth"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/session"
	"github.com/luojam/ssh-chat/internal/storage/sqlite"
	"github.com/luojam/ssh-chat/internal/tui"
	gossh "golang.org/x/crypto/ssh"
)

const (
	envHost        = "SSH_CHAT_HOST"
	envPort        = "SSH_CHAT_PORT"
	envHostKeyPath = "SSH_CHAT_HOST_KEY_PATH"
	envSQLitePath  = "SSH_CHAT_SQLITE_PATH"

	defaultHost        = "0.0.0.0"
	defaultPort        = "2222"
	defaultHostKeyPath = "/var/lib/ssh-chat/ssh_host_ed25519"
	defaultSQLitePath  = "/var/lib/ssh-chat/ssh-chat.sqlite"
	defaultMember      = "anonymous"
)

type serverConfig struct {
	Host        string
	Port        string
	HostKeyPath string
	SQLitePath  string
}

func loadConfig() serverConfig {
	return serverConfig{
		Host:        envString(envHost, defaultHost),
		Port:        envString(envPort, defaultPort),
		HostKeyPath: envString(envHostKeyPath, defaultHostKeyPath),
		SQLitePath:  envString(envSQLitePath, defaultSQLitePath),
	}
}

func envString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	config := loadConfig()
	if err := ensureParentDir(config.SQLitePath); err != nil {
		log.Error("Could not create SQLite data directory", "path", config.SQLitePath, "error", err)
		return
	}
	if err := ensureParentDir(config.HostKeyPath); err != nil {
		log.Error("Could not create SSH host key directory", "path", config.HostKeyPath, "error", err)
		return
	}

	db, err := sqlite.Open(context.Background(), config.SQLitePath)
	if err != nil {
		log.Error("Could not open SQLite database", "error", err)
		return
	}
	defer db.Close()
	authService := auth.NewPasswordService(sqlite.NewUserStore(db))
	chatService := chat.NewService(sqlite.NewChatStore(db))

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(config.Host, config.Port)),
		wish.WithHostKeyPath(config.HostKeyPath),
		// Accept any client public key as transport identity. App-level username/password
		// auth still happens in the TUI; this only makes Wish expose sess.PublicKey()
		// so the Session can offer explicit key linking after password auth.
		wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool { return true }),

		// Wish builds the handler chain from first to last, so the last middleware
		// listed here runs first for each SSH session. That gives logging the outer
		// view of the whole flow, lets activeterm reject clients without a usable
		// terminal, and only then starts one Bubble Tea program for the session.
		wish.WithMiddleware(
			bubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				return teaHandler(sess, chatService, authService)
			}),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
		return
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", config.Host, "port", config.Port, "sqlite", config.SQLitePath, "host_key", config.HostKeyPath)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}

func teaHandler(sess ssh.Session, chatService session.ChatService, authService auth.Service) (tea.Model, []tea.ProgramOption) {
	pty, _, ok := sess.Pty()
	if !ok {
		// Wish's active-terminal middleware should reject this earlier, but keeping
		// the handler defensive makes it safe even if middleware order changes.
		return nil, nil
	}

	publicKey, fingerprint := sshKeyIdentity(sess)

	app := session.New(session.Config{
		Width:             pty.Window.Width,
		Height:            pty.Window.Height,
		Context:           sess.Context(),
		ChatService:       chatService,
		AuthService:       authService,
		SSHPublicKey:      publicKey,
		SSHKeyFingerprint: fingerprint,
		// Display name from SSH login is used only before app auth; authenticated
		// user ID/name replace this before any room action is allowed.
		Member: chat.Member{
			Name: memberName(sess),
		},
	})
	return tui.WithMinimumSize(app, pty.Window.Width, pty.Window.Height), []tea.ProgramOption{tea.WithContext(sess.Context())}
}

func sshKeyIdentity(sess ssh.Session) (string, string) {
	return sshKeyIdentityFromPublicKey(sess.PublicKey())
}

func sshKeyIdentityFromPublicKey(key ssh.PublicKey) (string, string) {
	if key == nil {
		return "", ""
	}
	publicKey := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))
	fingerprint := gossh.FingerprintSHA256(key)
	return publicKey, fingerprint
}

func memberName(sess ssh.Session) string {
	name := sess.User()
	if name == "" {
		return defaultMember
	}
	return name
}
