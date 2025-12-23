package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/luytbq/personal-notification-service/internal/api"
	"github.com/luytbq/personal-notification-service/internal/channels"
	"github.com/luytbq/personal-notification-service/internal/config"
	"github.com/luytbq/personal-notification-service/internal/queue"
	"github.com/luytbq/personal-notification-service/internal/ratelimit"
)

func main() {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Setup logger
	logger := setupLogger()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		slog.Int("port", cfg.Port),
		slog.Int("rate_limit_per_minute", cfg.RateLimitPerMinute),
		slog.Int("worker_concurrency", cfg.WorkerConcurrency),
		slog.Int("max_retries", cfg.MaxRetries),
	)

	// Setup channel registry
	registry := channels.NewRegistry()
	registry.Register(channels.NewTelegramChannel(cfg.TelegramBotToken, cfg.TelegramChatID))
	// Email channel is scaffolded but not implemented
	// registry.Register(channels.NewEmailChannel())

	// Setup rate limiter
	limiter := ratelimit.NewLimiter(cfg.RateLimitPerMinute)

	// Setup queue names with prefix
	queueNames := queue.NewQueueNames(cfg.RedisKeyPrefix)
	logger.Info("queue names configured",
		slog.String("prefix", cfg.RedisKeyPrefix),
		slog.String("notifications_queue", queueNames.Notifications),
		slog.String("task_type", queueNames.TaskType),
	)

	// Setup queue client
	queueClient := queue.NewClient(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.MaxRetries,
		logger,
		queueNames,
	)
	defer queueClient.Close()

	// Setup worker
	worker := queue.NewWorker(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.WorkerConcurrency,
		registry,
		logger,
		queueNames,
	)

	// Setup HTTP router
	router := api.NewRouter(cfg, limiter, queueClient, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start worker in a goroutine
	go func() {
		logger.Info("starting worker",
			slog.Int("concurrency", cfg.WorkerConcurrency),
		)
		if err := worker.Start(); err != nil {
			logger.Error("worker failed to start", slog.String("error", err.Error()))
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("starting HTTP server",
			slog.Int("port", cfg.Port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	// Shutdown HTTP server first (stop accepting new requests)
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	} else {
		logger.Info("HTTP server stopped")
	}

	// Shutdown worker (wait for in-flight tasks)
	worker.Shutdown()
	logger.Info("worker stopped")

	logger.Info("shutdown complete")
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
