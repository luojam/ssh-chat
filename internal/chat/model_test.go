package chat

import (
	"errors"
	"fmt"
	"testing"
	"time"
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

func TestRoomSubscribeReplaysHistory(t *testing.T) {
	room := NewRoom()
	if _, err := room.Post(Member{Name: "jami"}, "first"); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if _, err := room.Post(Member{Name: "sara"}, "second"); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	subscription := room.Subscribe()
	defer subscription.Close()

	assertEventBody(t, subscription, "first")
	assertEventBody(t, subscription, "second")
}

func TestRoomSubscribeReplaysRecentHistoryOnly(t *testing.T) {
	room := NewRoom()
	for i := 0; i < historyLimit+1; i++ {
		if _, err := room.Post(Member{Name: "jami"}, fmt.Sprintf("message-%02d", i)); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
	}

	subscription := room.Subscribe()
	defer subscription.Close()

	for i := 1; i < historyLimit+1; i++ {
		assertEventBody(t, subscription, fmt.Sprintf("message-%02d", i))
	}

	select {
	case event := <-subscription.Events():
		t.Fatalf("unexpected extra history event: %+v", event)
	default:
	}
}

func TestRoomPostDropsSlowSubscriber(t *testing.T) {
	room := NewRoom()
	slow := room.Subscribe()
	fast := room.Subscribe()
	defer fast.Close()

	for i := 0; i < subscriptionBuffer; i++ {
		if _, err := room.Post(Member{Name: "jami"}, "hello"); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		<-fast.Events()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = room.Post(Member{Name: "jami"}, "overflow")
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Post blocked on slow subscriber")
	}

	for i := 0; i < subscriptionBuffer; i++ {
		select {
		case _, ok := <-slow.Events():
			if !ok {
				t.Fatal("slow subscriber should keep buffered events before close")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out draining slow subscriber")
		}
	}

	select {
	case _, ok := <-slow.Events():
		if ok {
			t.Fatal("slow subscriber should not receive overflow event")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("slow subscriber channel should be closed")
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

func assertEventBody(t *testing.T, subscription *Subscription, want string) {
	t.Helper()

	select {
	case event := <-subscription.Events():
		if got := event.Message.Body; got != want {
			t.Fatalf("event body = %q, want %q", got, want)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timed out waiting for history event %q", want)
	}
}
