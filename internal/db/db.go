package db

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"event-metrics-service/internal/config"
)

// NewPool creates a PostgreSQL connection pool configured with sane defaults.
func NewPool(ctx context.Context, dbURL string, cfg *config.Config) (*pgxpool.Pool, error) {
	pgxCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	pgxCfg.MinConns = cfg.DBMinConns
	pgxCfg.MaxConns = cfg.DBMaxConns
	pgxCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	pgxCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	pgxCfg.HealthCheckPeriod = 30 * time.Second

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if strings.ToLower(cfg.AppMode) == "benchmark" {
		log.Printf("db pool configured: max_conns=%d min_conns=%d max_conn_lifetime=%s max_conn_idle=%s", pgxCfg.MaxConns, pgxCfg.MinConns, pgxCfg.MaxConnLifetime, pgxCfg.MaxConnIdleTime)
	}

	return pool, nil
}
