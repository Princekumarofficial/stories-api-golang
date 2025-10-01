package websocket

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/princekumarofficial/stories-service/internal/utils/jwt"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
	wsClient "github.com/princekumarofficial/stories-service/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, you should check the origin properly
		return true
	},
}

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(hub *wsClient.Hub, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get JWT token from query parameter
		token := r.URL.Query().Get("token")
		if token == "" {
			slog.Warn("WebSocket connection attempted without token")
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("token required")))
			return
		}

		// Validate JWT token and extract user ID
		userID, err := jwt.ExtractUserIDFromToken(token, jwtSecret)
		if err != nil {
			slog.Warn("WebSocket connection attempted with invalid token", slog.String("error", err.Error()))
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("invalid token")))
			return
		}

		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Failed to upgrade WebSocket connection", slog.String("error", err.Error()))
			return
		}

		// Create new client and register with hub
		client := wsClient.NewClient(conn, userID, hub)
		hub.RegisterClient(client)

		// Start client goroutines
		client.Start()

		slog.Info("WebSocket connection established", slog.String("user_id", userID))
	}
}
