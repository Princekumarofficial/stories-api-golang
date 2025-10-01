package websocket

import (
	"log/slog"
	"sync"

	"github.com/princekumarofficial/stories-service/internal/types"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
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
		broadcast:  make(chan *BroadcastMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If user already has a connection, close the old one
			if existingClient, exists := h.clients[client.userID]; exists {
				close(existingClient.send)
				slog.Info("Replaced existing WebSocket connection", slog.String("user_id", client.userID))
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			slog.Info("WebSocket client connected", slog.String("user_id", client.userID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				slog.Info("WebSocket client disconnected", slog.String("user_id", client.userID))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastToUsers(message.UserIDs, message.Event)
		}
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

// broadcastToUsers is the internal method that actually sends messages to users
func (h *Hub) broadcastToUsers(userIDs []string, event *types.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		if client, ok := h.clients[userID]; ok {
			err := client.SendEvent(event)
			if err != nil {
				slog.Error("Failed to send event to client",
					slog.String("user_id", userID),
					slog.String("error", err.Error()))
				// Remove the client if sending fails
				go func(c *Client) {
					h.unregister <- c
				}(client)
			}
		}
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
