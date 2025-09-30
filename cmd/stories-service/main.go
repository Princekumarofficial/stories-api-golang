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

	"github.com/princekumarofficial/stories-service/internal/config"
	"github.com/princekumarofficial/stories-service/internal/http/handlers/stories"
	"github.com/princekumarofficial/stories-service/internal/storage/postgres"
)

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

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})
	router.HandleFunc("GET /feed", stories.Feed())
	router.HandleFunc("POST /stories", stories.PostStory(storage))

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
