package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	GitHubClientID      string
	GitHubClientSecret  string
	AppSecret           string
	EncryptionKey       string
	FrontendOrigin      string
	RedisURL            string
	RateLimitPerMin     int
	SchedulerTickSec    int
	SchedulerMaxWorkers int
}

func Load() (Config, error) {
	rateLimitPerMin := 120
	if raw := os.Getenv("RATE_LIMIT_PER_MIN"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid RATE_LIMIT_PER_MIN: %w", err)
		}
		rateLimitPerMin = parsed
	}

	schedulerTickSec := 60
	if raw := os.Getenv("SYNC_SCHEDULER_TICK_SEC"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SYNC_SCHEDULER_TICK_SEC: %w", err)
		}
		schedulerTickSec = parsed
	}

	schedulerMaxWorkers := 3
	if raw := os.Getenv("SYNC_SCHEDULER_MAX_WORKERS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SYNC_SCHEDULER_MAX_WORKERS: %w", err)
		}
		schedulerMaxWorkers = parsed
	}

	cfg := Config{
		Port:                getOrDefault("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		AppSecret:           os.Getenv("APP_SECRET"),
		EncryptionKey:       os.Getenv("ENCRYPTION_KEY"),
		FrontendOrigin:      getOrDefault("FRONTEND_ORIGIN", "http://localhost:1420"),
		RedisURL:            getOrDefault("REDIS_URL", "redis://redis:6379/0"),
		RateLimitPerMin:     rateLimitPerMin,
		SchedulerTickSec:    schedulerTickSec,
		SchedulerMaxWorkers: schedulerMaxWorkers,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.GitHubClientID == "" {
		return Config{}, fmt.Errorf("GITHUB_CLIENT_ID is required")
	}
	if cfg.GitHubClientSecret == "" {
		return Config{}, fmt.Errorf("GITHUB_CLIENT_SECRET is required")
	}
	if cfg.AppSecret == "" {
		return Config{}, fmt.Errorf("APP_SECRET is required")
	}
	if len(cfg.EncryptionKey) < 32 {
		return Config{}, fmt.Errorf("ENCRYPTION_KEY must be at least 32 chars")
	}
	if cfg.SchedulerTickSec <= 0 {
		return Config{}, fmt.Errorf("SYNC_SCHEDULER_TICK_SEC must be > 0")
	}
	if cfg.SchedulerMaxWorkers <= 0 {
		return Config{}, fmt.Errorf("SYNC_SCHEDULER_MAX_WORKERS must be > 0")
	}

	return cfg, nil
}

func getOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
