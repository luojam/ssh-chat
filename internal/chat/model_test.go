package chat

import (
	"errors"
	"testing"
)

func TestRoomPostStoresTrimmedMessage(t *testing.T) {
	room := NewRoom()
	author := Member{Name: "jami"}

	msg, err := room.Post(author, " hello ")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	if got := msg.Author.Name; got != author.Name {
		t.Fatalf("author = %q, want %q", got, author.Name)
	}
	if got := msg.Body; got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
	if msg.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
	if got := len(room.messages); got != 1 {
		t.Fatalf("stored messages = %d, want 1", got)
	}
}

func TestRoomPostBroadcastsToSubscribers(t *testing.T) {
	room := NewRoom()
	subscription := room.Subscribe()
	defer subscription.Close()

	msg, err := room.Post(Member{Name: "jami"}, "hello")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	event := <-subscription.Events()
	if got := event.Message.Body; got != msg.Body {
		t.Fatalf("event body = %q, want %q", got, msg.Body)
	}
	if got := event.Message.Author.Name; got != msg.Author.Name {
		t.Fatalf("event author = %q, want %q", got, msg.Author.Name)
	}
}

func TestRoomPostRejectsEmptyMessage(t *testing.T) {
	room := NewRoom()

	_, err := room.Post(Member{Name: "jami"}, "   ")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("error = %v, want ErrEmptyMessage", err)
	}
	if got := len(room.messages); got != 0 {
		t.Fatalf("stored messages = %d, want 0", got)
	}
}
