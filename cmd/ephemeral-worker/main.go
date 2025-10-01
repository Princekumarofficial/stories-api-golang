package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/princekumarofficial/stories-service/internal/config"
	"github.com/princekumarofficial/stories-service/internal/storage/postgres"
)

type EphemeralWorker struct {
	storage  *postgres.Postgres
	interval time.Duration
	logger   *slog.Logger
}

func NewEphemeralWorker(storage *postgres.Postgres, interval time.Duration) *EphemeralWorker {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &EphemeralWorker{
		storage:  storage,
		interval: interval,
		logger:   logger,
	}
}

func (ew *EphemeralWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(ew.interval)
	defer ticker.Stop()

	ew.logger.Info("Ephemeral worker started",
		"interval", ew.interval.String())

	// Run once immediately on startup
	ew.processExpiredStories(ctx)

	for {
		select {
		case <-ctx.Done():
			ew.logger.Info("Ephemeral worker shutting down")
			return
		case <-ticker.C:
			ew.processExpiredStories(ctx)
		}
	}
}

func (ew *EphemeralWorker) processExpiredStories(ctx context.Context) {
	startTime := time.Now()
	
	ew.logger.Info("Starting expired stories cleanup")

	count, err := ew.storage.SoftDeleteExpiredStories()
	if err != nil {
		ew.logger.Error("Failed to process expired stories",
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds())
		return
	}

	duration := time.Since(startTime)
	
	ew.logger.Info("Completed expired stories cleanup",
		"stories_deleted", count,
		"duration_ms", duration.Milliseconds(),
		"duration", duration.String())
}

func main() {
	// Load config
	cfg := config.MustLoad()

	// Initialize database connection
	storage, err := postgres.NewPostgres(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	slog.Info("Connected to Postgres database")

	// Create worker with 1-minute interval
	worker := NewEphemeralWorker(storage, time.Minute)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	// Start the worker
	worker.Start(ctx)
	
	slog.Info("Ephemeral worker stopped")
}
