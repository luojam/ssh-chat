package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/session"
)

const (
	host          = "localhost"
	port          = "2222"
	hostKeyPath   = ".ssh/id_ed25519"
	validPassword = "password"
	defaultMember = "anonymous"
)

func main() {
	room := chat.NewRoom()

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(hostKeyPath),

		// Password auth only for dev simplicity.
		// SSH in with:
		//		ssh -o PreferredAuthentications=password -p 2222 localhost
		wish.WithPasswordAuth(func(_ ssh.Context, password string) bool {
			return password == validPassword
		}),

		// Wish builds the handler chain from first to last, so the last middleware
		// listed here runs first for each SSH session. That gives logging the outer
		// view of the whole flow, lets activeterm reject clients without a usable
		// terminal, and only then starts one Bubble Tea program for the session.
		wish.WithMiddleware(
			bubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				return teaHandler(sess, room)
			}),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
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

func teaHandler(sess ssh.Session, room *chat.Room) (tea.Model, []tea.ProgramOption) {
	pty, _, ok := sess.Pty()
	if !ok {
		// Wish's active-terminal middleware should reject this earlier, but keeping
		// the handler defensive makes it safe even if middleware order changes.
		return nil, nil
	}

	id, err := newSessionMemberID()
	if err != nil {
		log.Error("Could not create member ID", "error", err)
		return nil, nil
	}

	return session.New(session.Config{
		Width:   pty.Window.Width,
		Height:  pty.Window.Height,
		Context: sess.Context(),
		Room:    room,
		Member: chat.Member{
			ID:   id,
			Name: memberName(sess),
		},
	}), []tea.ProgramOption{tea.WithContext(sess.Context())}
}

func newSessionMemberID() (chat.MemberID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return chat.MemberID(hex.EncodeToString(b[:])), nil
}

func memberName(sess ssh.Session) string {
	name := sess.User()
	if name == "" {
		return defaultMember
	}
	return name
}
