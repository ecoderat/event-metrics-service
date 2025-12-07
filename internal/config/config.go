package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	HTTPPort          string
	AppMode           string
	FiberPrefork      bool
	ClickHouseAddrs   []string
	ClickHouseUser    string
	ClickHousePass    string
	ClickHouseDB      string
	UseTLS            bool
	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
	FutureTolerance   time.Duration
	WorkerBufferSize  int
	WorkerBatchSize   int
	WorkerFlushEvery  time.Duration
	HealthPingRetries int
	HealthPingDelay   time.Duration
}

// Load reads configuration from environment variables with sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:          getEnv("HTTP_PORT", ":8080"),
		AppMode:           strings.ToLower(getEnv("APP_MODE", "dev")),
		FiberPrefork:      parseBoolEnv("FIBER_PREFORK", false),
		ClickHouseAddrs:   splitAndTrim(getEnv("CLICKHOUSE_ADDRS", "localhost:9000")),
		ClickHouseUser:    getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePass:    os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseDB:      getEnv("CLICKHOUSE_DB", "default"),
		UseTLS:            parseBoolEnv("CLICKHOUSE_TLS", false),
		DBMaxConns:        parseIntEnv("DB_MAX_CONNS", 50),
		DBMinConns:        parseIntEnv("DB_MIN_CONNS", 10),
		DBMaxConnLifetime: parseDurationEnv("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime: parseDurationEnv("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		FutureTolerance:   parseDurationEnv("FUTURE_TOLERANCE", 0),
		WorkerBufferSize:  parseIntEnv("WORKER_BUFFER_SIZE", 10000),
		WorkerBatchSize:   parseIntEnv("WORKER_BATCH_SIZE", 1000),
		WorkerFlushEvery:  parseDurationEnv("WORKER_FLUSH_EVERY", time.Second),
		HealthPingRetries: parseIntEnv("DB_PING_RETRIES", 20),
		HealthPingDelay:   parseDurationEnv("DB_PING_DELAY", 1500*time.Millisecond),
	}

	if len(cfg.ClickHouseAddrs) == 0 || cfg.ClickHouseAddrs[0] == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDRS is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt32Env(key string, fallback int32) int32 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}

func parseIntEnv(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}
