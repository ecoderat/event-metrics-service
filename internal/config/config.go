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
	DatabaseURL       string
	AppMode           string
	FiberPrefork      bool
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
}

// Load reads configuration from environment variables with sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:          getEnv("HTTP_PORT", ":8080"),
		AppMode:           strings.ToLower(getEnv("APP_MODE", "dev")),
		FiberPrefork:      parseBoolEnv("FIBER_PREFORK", false),
		DBMaxConns:        parseInt32Env("DB_MAX_CONNS", 50),
		DBMinConns:        parseInt32Env("DB_MIN_CONNS", 10),
		DBMaxConnLifetime: parseDurationEnv("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime: parseDurationEnv("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
	}
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
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
