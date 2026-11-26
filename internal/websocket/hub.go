package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/princekumarofficial/stories-service/internal/types"
)

// broadcastBuffer is how many events can queue while the hub goroutine is busy.
// With an unbuffered channel and a non-blocking send, events were silently
// dropped whenever Run was handling a register/unregister.
const broadcastBuffer = 1024

// Hub maintains the set of active clients and broadcasts messages to the clients.
//
// Invariant: only the hub goroutine (Run) sends on or closes a client's send
// channel. Every close goes through remove (or the replace branch in Run), so a
// channel is closed exactly once and never written to after it is closed.
type Hub struct {
	// Registered clients mapped by user ID
	clients map[string]*Client

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex to protect clients map
	mu sync.RWMutex

	// Channel to broadcast events
	broadcast chan *BroadcastMessage
}

// BroadcastMessage represents a message to be broadcast to specific users
type BroadcastMessage struct {
	UserIDs []string     `json:"user_ids"`
	Event   *types.Event `json:"event"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, broadcastBuffer),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If the user already has a connection, close the old one. Its writePump
			// sees the closed channel, sends a close frame and exits; the old
			// readPump's later unregister is then a no-op (see remove).
			if existingClient, exists := h.clients[client.userID]; exists && existingClient != client {
				close(existingClient.send)
				slog.Info("Replaced existing WebSocket connection", slog.String("user_id", client.userID))
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			slog.Info("WebSocket client connected", slog.String("user_id", client.userID))

		case client := <-h.unregister:
			h.remove(client)

		case message := <-h.broadcast:
			h.broadcastToUsers(message.UserIDs, message.Event)
		}
	}
}

// remove drops a client only if it is still the registered connection for its
// user. Before this check, an unregister from a replaced connection deleted the
// NEW connection and closed the old channel a second time, which panicked and
// took down the whole server.
func (h *Hub) remove(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if current, ok := h.clients[client.userID]; ok && current == client {
		delete(h.clients, client.userID)
		close(client.send)
		slog.Info("WebSocket client disconnected", slog.String("user_id", client.userID))
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToUsers sends an event to specific users
func (h *Hub) BroadcastToUsers(userIDs []string, event *types.Event) {
	message := &BroadcastMessage{
		UserIDs: userIDs,
		Event:   event,
	}

	select {
	case h.broadcast <- message:
	default:
		slog.Warn("Broadcast channel is full, dropping message")
	}
}

// BroadcastToUser sends an event to a specific user
func (h *Hub) BroadcastToUser(userID string, event *types.Event) {
	h.BroadcastToUsers([]string{userID}, event)
}

// broadcastToUsers is the internal method that actually sends messages to users.
// It runs on the hub goroutine, so it may close channels via remove.
func (h *Hub) broadcastToUsers(userIDs []string, event *types.Event) {
	data, err := json.Marshal(event) // once, not once per recipient
	if err != nil {
		slog.Error("Failed to marshal event", slog.String("error", err.Error()))
		return
	}

	var slow []*Client
	h.mu.RLock()
	for _, userID := range userIDs {
		if client, ok := h.clients[userID]; ok {
			select {
			case client.send <- data:
			default:
				// The client's buffer is full: it is not reading fast enough.
				slow = append(slow, client)
			}
		}
	}
	h.mu.RUnlock()

	for _, client := range slow {
		slog.Warn("Dropping slow WebSocket client", slog.String("user_id", client.userID))
		h.remove(client)
	}
}

// GetConnectedUsers returns a list of currently connected user IDs
func (h *Hub) GetConnectedUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// IsUserConnected checks if a user is currently connected
func (h *Hub) IsUserConnected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.clients[userID]
	return exists
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}
