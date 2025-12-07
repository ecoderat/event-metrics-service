package db

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"event-metrics-service/internal/config"
)

// NewConnection creates a ClickHouse connection with pool settings.
func NewConnection(ctx context.Context, cfg *config.Config) (clickhouse.Conn, error) {
	options := &clickhouse.Options{
		Addr: cfg.ClickHouseAddrs,
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDB,
			Username: cfg.ClickHouseUser,
			Password: cfg.ClickHousePass,
		},
		TLS: func() *tls.Config {
			if cfg.UseTLS {
				return &tls.Config{InsecureSkipVerify: true} // controlled via env flag
			}
			return nil
		}(),
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		ConnMaxLifetime:  cfg.DBMaxConnLifetime,
		MaxOpenConns:     cfg.DBMaxConns,
		MaxIdleConns:     cfg.DBMinConns,
		DialTimeout:      5 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}

	if err := waitForPing(ctx, conn, 20, 1500*time.Millisecond); err != nil {
		return nil, err
	}

	if cfg.AppMode == "benchmark" {
		log.Printf("clickhouse connected: addrs=%v max_conns=%d min_conns=%d", cfg.ClickHouseAddrs, cfg.DBMaxConns, cfg.DBMinConns)
	}

	return conn, nil
}

// waitForPing retries Ping to allow the DB container to become ready.
func waitForPing(ctx context.Context, conn clickhouse.Conn, attempts int, delay time.Duration) error {
	wait := delay
	for i := 1; i <= attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := conn.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		if i == attempts {
			return fmt.Errorf("ping clickhouse: %w", err)
		}
		time.Sleep(wait)
		if wait < 5*time.Second {
			wait += 500 * time.Millisecond
		}
	}
	return fmt.Errorf("ping clickhouse: exceeded retries")
}
