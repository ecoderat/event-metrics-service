package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"

	"event-metrics-service/internal/config"
	"event-metrics-service/internal/controller"
	"event-metrics-service/internal/db"
	httpserver "event-metrics-service/internal/http"
	"event-metrics-service/internal/repository"
	"event-metrics-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	repo := repository.NewEventRepository(pool)
	worker := service.NewbatchEventWorker(repo, 10000, 1000, 1*time.Second)
	eventService := service.NewEventService(repo, worker)
	eventController := controller.NewEventController(eventService)

	server := httpserver.NewServer(cfg, eventController)

	go logRuntimeStats(ctx, cfg, pool)

	log.Printf("starting server on %s", cfg.HTTPPort)
	if err := server.Listen(cfg.HTTPPort); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func logRuntimeStats(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	logPoolStats(cfg, pool)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logPoolStats(cfg, pool)
		}
	}
}

func logPoolStats(cfg *config.Config, pool *pgxpool.Pool) {
	stats := pool.Stat()
	log.Printf("mode=%s prefork=%t port=%s db_conns[total=%d idle=%d acquired=%d]", cfg.AppMode, cfg.FiberPrefork, cfg.HTTPPort, stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())
}
