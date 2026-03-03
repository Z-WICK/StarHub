package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wick/github-star-manager/services/api/internal/api"
	"github.com/wick/github-star-manager/services/api/internal/auth"
	"github.com/wick/github-star-manager/services/api/internal/config"
	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/github"
	"github.com/wick/github-star-manager/services/api/internal/security"
	starspkg "github.com/wick/github-star-manager/services/api/internal/stars"
	syncpkg "github.com/wick/github-star-manager/services/api/internal/sync"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := db.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer store.Close()

	migrationsPath := filepath.Join("services", "api", "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = filepath.Join("migrations")
	}
	if err := store.RunMigrations(ctx, migrationsPath); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := store.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	repository := db.NewRepository(store)
	ghClient := github.NewClient()
	tokenCipher, err := security.NewTokenCipher(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("init token cipher: %v", err)
	}
	var sessionCache auth.SessionCache
	redisSessionCache, err := auth.NewRedisSessionCache(cfg.RedisURL)
	if err != nil {
		log.Printf("init redis session cache failed, continue without cache: %v", err)
	} else {
		defer func() {
			if closeErr := redisSessionCache.Close(); closeErr != nil {
				log.Printf("close redis session cache error: %v", closeErr)
			}
		}()
		if err := redisSessionCache.Ping(ctx); err != nil {
			log.Printf("redis ping failed, continue without cache: %v", err)
		} else {
			sessionCache = redisSessionCache
		}
	}

	authService := auth.NewService(repository, ghClient, tokenCipher, sessionCache)
	starsService := starspkg.NewService(repository, authService, ghClient)
	syncService := syncpkg.NewService(
		repository,
		authService,
		ghClient,
		time.Duration(cfg.SchedulerTickSec)*time.Second,
		cfg.SchedulerMaxWorkers,
	)

	authHandler := auth.NewHandler(authService)
	starsHandler := starspkg.NewHandler(starsService)
	syncHandler := syncpkg.NewHandler(syncService)

	router := api.NewRouter(cfg, authService, authHandler, starsHandler, syncHandler)

	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	defer schedulerCancel()
	go syncService.StartScheduler(schedulerCtx)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("API listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
