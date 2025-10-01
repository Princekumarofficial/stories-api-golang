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

	"github.com/go-redis/redis/v8"
	_ "github.com/princekumarofficial/stories-service/docs"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/princekumarofficial/stories-service/internal/cache"
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

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	slog.Info("Connected to Redis")

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

	// Initialize rate limiting
	rateLimitConfig := middleware.NewRateLimitConfig(redisClient)

	// Initialize caching layer
	cacheService := cache.NewCacheService(storage, redisClient)
	optimizedQuery := cache.NewOptimizedFeedQuery(storage.GetDB())
	slog.Info("Cache service initialized")

	// setup server
	router := http.NewServeMux()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// WebSocket route
	router.HandleFunc("GET /ws", wsHandler.WebSocketHandler(hub, cfg.JWTSecret))

	// Protected routes with rate limiting
	router.Handle("POST /stories", authMiddleware(rateLimitConfig.RateLimitedHandler("stories", stories.PostStory(cacheService))))
	router.Handle("GET /stories/{id}", authMiddleware(http.HandlerFunc(stories.GetStory(cacheService))))
	router.Handle("GET /feed", authMiddleware(http.HandlerFunc(stories.CachedFeed(cacheService))))
	router.Handle("GET /feed/optimized", authMiddleware(http.HandlerFunc(stories.OptimizedFeed(cacheService, optimizedQuery))))
	router.Handle("POST /stories/{id}/view", authMiddleware(http.HandlerFunc(stories.ViewStoryWithEvents(cacheService, eventPublisher))))
	router.Handle("POST /stories/{id}/reactions", authMiddleware(rateLimitConfig.RateLimitedHandler("reactions", stories.AddReactionWithEvents(cacheService, eventPublisher))))
	router.Handle("GET /me/stats", authMiddleware(http.HandlerFunc(users.GetStats(cacheService))))

	// Follow/Unfollow routes
	router.Handle("POST /follow/{user_id}", authMiddleware(http.HandlerFunc(users.FollowUser(cacheService))))
	router.Handle("DELETE /follow/{user_id}", authMiddleware(http.HandlerFunc(users.UnfollowUser(cacheService))))

	// Media routes (protected)
	router.Handle("POST /media/upload-url", authMiddleware(http.HandlerFunc(mediaHandlers.GenerateUploadURL())))
	router.Handle("GET /media", authMiddleware(http.HandlerFunc(mediaHandlers.ListUserMedia())))
	router.Handle("GET /media/{object_key}/info", authMiddleware(http.HandlerFunc(mediaHandlers.GetMediaInfo())))
	router.Handle("GET /media/{object_key}/download-url", authMiddleware(http.HandlerFunc(mediaHandlers.GenerateDownloadURL())))
	router.Handle("DELETE /media/{object_key}", authMiddleware(http.HandlerFunc(mediaHandlers.DeleteMedia())))

	// Public routes
	router.Handle("POST /signup", http.HandlerFunc(users.SignUp(storage)))
	router.Handle("POST /login", http.HandlerFunc(users.Login(storage, cfg.JWTSecret)))

	// Cache monitoring endpoints (for development/admin)
	router.Handle("GET /cache/stats", http.HandlerFunc(cache.GetCacheStats(redisClient)))
	router.Handle("DELETE /cache/clear", http.HandlerFunc(cache.ClearCache(redisClient)))

	// Documentation
	router.Handle("GET /docs/", httpSwagger.WrapHandler)

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

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		slog.Error("failed to close Redis connection", slog.String("error", err.Error()))
	}

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error("failed to gracefully shutdown server", slog.String("error", err.Error()))
		return
	}

	slog.Info("Server stopped")
}
