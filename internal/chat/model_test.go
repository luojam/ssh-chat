package chat

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRoomPostStoresTrimmedMessage(t *testing.T) {
	room := NewRoom()
	author := Member{ID: "user-1", Name: "user"}

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
	if got := msg.ID; got != 1 {
		t.Fatalf("id = %d, want 1", got)
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

	msg, err := room.Post(Member{ID: "user-1", Name: "user"}, "hello")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	event := <-subscription.Events()
	if got := event.Kind; got != MessagePosted {
		t.Fatalf("event kind = %d, want MessagePosted", got)
	}
	if got := event.Message.Body; got != msg.Body {
		t.Fatalf("event body = %q, want %q", got, msg.Body)
	}
	if got := event.Message.Author.Name; got != msg.Author.Name {
		t.Fatalf("event author = %q, want %q", got, msg.Author.Name)
	}
	if got := event.Message.ID; got != msg.ID {
		t.Fatalf("event id = %d, want %d", got, msg.ID)
	}
}

func TestRoomJoinBroadcastsMemberJoined(t *testing.T) {
	room := NewRoom()
	first := room.Join(Member{ID: "user-1", Name: "user"})
	defer first.Close()

	event := assertNextEvent(t, first)
	if got := event.Kind; got != MemberJoined {
		t.Fatalf("event kind = %d, want MemberJoined", got)
	}
	if got := event.Member.Name; got != "user" {
		t.Fatalf("event member = %q, want %q", got, "user")
	}
}

func TestJoinedSubscriptionCloseBroadcastsMemberLeft(t *testing.T) {
	room := NewRoom()
	first := room.Join(Member{ID: "user-1", Name: "user"})
	defer first.Close()
	_ = assertNextEvent(t, first)

	second := room.Join(Member{ID: "sara-1", Name: "sara"})
	defer second.Close()
	_ = assertNextEvent(t, first)
	_ = assertNextEvent(t, second)

	second.Close()

	event := assertNextEvent(t, first)
	if got := event.Kind; got != MemberLeft {
		t.Fatalf("event kind = %d, want MemberLeft", got)
	}
	if got := event.Member.Name; got != "sara" {
		t.Fatalf("event member = %q, want %q", got, "sara")
	}
}

func TestRoomPostAssignsSequentialMessageIDs(t *testing.T) {
	room := NewRoom()

	first, err := room.Post(Member{ID: "user-1", Name: "user"}, "first")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	second, err := room.Post(Member{ID: "user-1", Name: "user"}, "second")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	if first.ID == 0 {
		t.Fatal("first id should be non-zero")
	}
	if got, want := second.ID, first.ID+1; got != want {
		t.Fatalf("second id = %d, want %d", got, want)
	}
}

func TestRoomSubscribeReplaysHistory(t *testing.T) {
	room := NewRoom()
	first, err := room.Post(Member{ID: "user-1", Name: "user"}, "first")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	second, err := room.Post(Member{ID: "sara-1", Name: "sara"}, "second")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	subscription := room.Subscribe()
	defer subscription.Close()

	assertEvent(t, subscription, first.ID, "first")
	assertEvent(t, subscription, second.ID, "second")
}

func TestRoomSubscribeReplaysRecentHistoryOnly(t *testing.T) {
	room := NewRoom()
	for i := 0; i < historyLimit+1; i++ {
		if _, err := room.Post(Member{ID: "user-1", Name: "user"}, fmt.Sprintf("message-%02d", i)); err != nil {
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
		if _, err := room.Post(Member{ID: "user-1", Name: "user"}, "hello"); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		<-fast.Events()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = room.Post(Member{ID: "user-1", Name: "user"}, "overflow")
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

	_, err := room.Post(Member{ID: "user-1", Name: "user"}, "   ")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("error = %v, want ErrEmptyMessage", err)
	}
	if got := len(room.messages); got != 0 {
		t.Fatalf("stored messages = %d, want 0", got)
	}
}

func assertEventBody(t *testing.T, subscription *Subscription, want string) {
	t.Helper()

	event := assertNextEvent(t, subscription)
	if got := event.Message.Body; got != want {
		t.Fatalf("event body = %q, want %q", got, want)
	}
}

func assertEvent(t *testing.T, subscription *Subscription, wantID MessageID, wantBody string) {
	t.Helper()

	event := assertNextEvent(t, subscription)
	if got := event.Kind; got != MessagePosted {
		t.Fatalf("event kind = %d, want MessagePosted", got)
	}
	if got := event.Message.ID; got != wantID {
		t.Fatalf("event id = %d, want %d", got, wantID)
	}
	if got := event.Message.Body; got != wantBody {
		t.Fatalf("event body = %q, want %q", got, wantBody)
	}
}

func assertNextEvent(t *testing.T, subscription *Subscription) Event {
	t.Helper()

	select {
	case event := <-subscription.Events():
		return event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}
