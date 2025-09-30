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
	"github.com/princekumarofficial/stories-service/internal/http/handlers/stories"
	"github.com/princekumarofficial/stories-service/internal/http/handlers/users"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	"github.com/princekumarofficial/stories-service/internal/storage/postgres"
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

	// setup server
	router := http.NewServeMux()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})
	router.HandleFunc("GET /feed", stories.Feed())
	
	// Protected routes
	router.Handle("POST /stories", authMiddleware(http.HandlerFunc(stories.PostStory(storage))))
	
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
