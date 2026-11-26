package websocket

import (
	"testing"
	"time"

	"github.com/princekumarofficial/stories-service/internal/types"
)

// newTestClient builds a client with no network connection; the hub only uses
// the user ID and the send channel.
func newTestClient(h *Hub, userID string, buffer int) *Client {
	return &Client{send: make(chan []byte, buffer), userID: userID, hub: h}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func testEvent() *types.Event {
	return types.NewEvent(types.EventStoryViewed, &types.StoryViewedEvent{StoryID: "1", ViewerID: "2"})
}

// Reconnecting as the same user used to crash the server: the old socket's
// unregister deleted the new socket and closed the old channel a second time.
func TestReconnectSameUserKeepsNewConnection(t *testing.T) {
	h := NewHub()
	go h.Run()

	old := newTestClient(h, "u1", 256)
	h.RegisterClient(old)
	newer := newTestClient(h, "u1", 256)
	h.RegisterClient(newer)

	select {
	case _, ok := <-old.send:
		if ok {
			t.Fatal("old connection received data; expected its channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("old connection's channel was not closed on reconnect")
	}

	// What the old connection's readPump does when its socket closes.
	h.UnregisterClient(old)
	h.UnregisterClient(old)

	h.BroadcastToUser("u1", testEvent())
	select {
	case msg, ok := <-newer.send:
		if !ok || len(msg) == 0 {
			t.Fatal("new connection's channel was closed by the old connection's unregister")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event was not delivered to the new connection")
	}
	if !h.IsUserConnected("u1") {
		t.Fatal("new connection was removed by the old connection's unregister")
	}

	h.UnregisterClient(newer)
	waitFor(t, "u1 to disconnect", func() bool { return !h.IsUserConnected("u1") })
}

// A client whose buffer is full is dropped; its later unregister must not
// close the channel again.
func TestSlowClientIsDroppedOnce(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "slow", 1)
	h.RegisterClient(c)

	h.BroadcastToUser("slow", testEvent()) // fills the 1-slot buffer
	h.BroadcastToUser("slow", testEvent()) // buffer full: client is dropped
	waitFor(t, "slow client to be dropped", func() bool { return !h.IsUserConnected("slow") })

	h.UnregisterClient(c) // readPump's deferred unregister arrives afterwards

	if _, ok := <-c.send; !ok {
		t.Fatal("expected the first event to still be buffered")
	}
	if _, ok := <-c.send; ok {
		t.Fatal("expected the channel to be closed after the buffered event")
	}
}

// Events sent in a burst used to be dropped whenever the hub goroutine was not
// already waiting on the unbuffered broadcast channel.
func TestBurstOfEventsIsNotDropped(t *testing.T) {
	h := NewHub()
	go h.Run()

	const n = 500
	c := newTestClient(h, "author", n)
	h.RegisterClient(c)

	for i := 0; i < n; i++ {
		h.BroadcastToUser("author", testEvent())
	}
	waitFor(t, "all events to be delivered", func() bool { return len(c.send) == n })
}
