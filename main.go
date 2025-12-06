package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	DatabaseURL := os.Getenv("DATABASE_URL")
	HTTPPort := os.Getenv("HTTP_PORT")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pgxCfg, err := pgxpool.ParseConfig(DatabaseURL)
	if err != nil {
		log.Fatalf("parse db config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	fiberCfg := fiber.Config{}

	app := fiber.New(fiberCfg)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	log.Printf("starting server on %s", HTTPPort)
	if err := app.Listen(HTTPPort); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
