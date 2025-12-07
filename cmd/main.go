package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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

	conn, err := db.NewConnection(ctx, cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer conn.Close()

	if err := db.RunMigrations(ctx, conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := repository.NewEventRepository(conn)
	worker := service.NewbatchEventWorker(repo, cfg.WorkerBufferSize, cfg.WorkerBatchSize, cfg.WorkerFlushEvery)
	eventService := service.NewEventService(repo, worker, cfg.FutureTolerance)
	eventController := controller.NewEventController(eventService)

	server := httpserver.NewServer(cfg, eventController)

	log.Printf("starting server on %s", cfg.HTTPPort)
	if err := server.Listen(cfg.HTTPPort); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
