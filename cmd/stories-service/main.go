package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/princekumarofficial/stories-service/docs"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/princekumarofficial/stories-service/internal/config"
	"github.com/princekumarofficial/stories-service/internal/events"
	"github.com/princekumarofficial/stories-service/internal/http/handlers/media"
	"github.com/princekumarofficial/stories-service/internal/http/handlers/stories"
	"github.com/princekumarofficial/stories-service/internal/http/handlers/users"
	wsHandler "github.com/princekumarofficial/stories-service/internal/http/handlers/websocket"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	mediaService "github.com/princekumarofficial/stories-service/internal/services/media"
	"github.com/princekumarofficial/stories-service/internal/storage/postgres"
	"github.com/princekumarofficial/stories-service/internal/websocket"
)

// @title Stories Service API
// @version 1.0
// @description A simple stories service API
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// load config
	cfg := config.MustLoad()
	// database setup

	storage, err := postgres.NewPostgres(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	slog.Info("Connected to Postgres database")

	// Initialize media service
	mediaService, err := mediaService.NewService(cfg)
	if err != nil {
		log.Fatal("Failed to initialize media service:", err)
	}
	slog.Info("Connected to MinIO")

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()
	slog.Info("WebSocket hub started")

	// Initialize event publisher
	eventPublisher := events.NewEventPublisher(hub)

	// Initialize handlers
	mediaHandlers := media.NewMediaHandlers(mediaService)

	// setup server
	router := http.NewServeMux()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// WebSocket route
	router.HandleFunc("GET /ws", wsHandler.WebSocketHandler(hub, cfg.JWTSecret))

	// Protected routes
	router.Handle("POST /stories", authMiddleware(http.HandlerFunc(stories.PostStory(storage))))
	router.Handle("GET /stories/{id}", authMiddleware(http.HandlerFunc(stories.GetStory(storage))))
	router.Handle("GET /feed", authMiddleware(http.HandlerFunc(stories.Feed(storage))))
	router.Handle("POST /stories/{id}/view", authMiddleware(http.HandlerFunc(stories.ViewStoryWithEvents(storage, eventPublisher))))
	router.Handle("POST /stories/{id}/reactions", authMiddleware(http.HandlerFunc(stories.AddReactionWithEvents(storage, eventPublisher))))
	router.Handle("GET /me/stats", authMiddleware(http.HandlerFunc(users.GetStats(storage))))

	// Follow/Unfollow routes
	router.Handle("POST /follow/{user_id}", authMiddleware(http.HandlerFunc(users.FollowUser(storage))))
	router.Handle("DELETE /follow/{user_id}", authMiddleware(http.HandlerFunc(users.UnfollowUser(storage))))

	// Media routes (protected)
	router.Handle("POST /media/upload-url", authMiddleware(http.HandlerFunc(mediaHandlers.GenerateUploadURL())))
	router.Handle("GET /media", authMiddleware(http.HandlerFunc(mediaHandlers.ListUserMedia())))
	router.Handle("GET /media/{object_key}/info", authMiddleware(http.HandlerFunc(mediaHandlers.GetMediaInfo())))
	router.Handle("GET /media/{object_key}/download-url", authMiddleware(http.HandlerFunc(mediaHandlers.GenerateDownloadURL())))
	router.Handle("DELETE /media/{object_key}", authMiddleware(http.HandlerFunc(mediaHandlers.DeleteMedia())))

	// Public routes
	router.HandleFunc("POST /signup", users.SignUp(storage))
	router.HandleFunc("POST /login", users.Login(storage, cfg.JWTSecret))

	// Swagger UI endpoint
	router.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	// setup router

	server := http.Server{
		Addr:    cfg.HTTPServer.Address,
		Handler: router,
	}

	log.Println("server started on", cfg.HTTPServer.Address)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("failed to start server: %s", err)
		}
	}()

	<-done

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error("failed to gracefully shutdown server", slog.String("error", err.Error()))
		return
	}

	slog.Info("Server stopped")
}
