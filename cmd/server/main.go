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

	"github.com/luytbq/personal-notification-service/internal/api"
	"github.com/luytbq/personal-notification-service/internal/channels"
	"github.com/luytbq/personal-notification-service/internal/config"
	"github.com/luytbq/personal-notification-service/internal/queue"
	"github.com/luytbq/personal-notification-service/internal/ratelimit"
)

func main() {
	logger := setupLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		slog.Int("port", cfg.Server.Port),
		slog.Int("rate_limit_per_minute", cfg.RateLimitPerMinute),
		slog.Int("worker_concurrency", cfg.Worker.Concurrency),
		slog.Int("max_retries", cfg.Worker.MaxRetries),
	)

	registry := channels.NewRegistry()
	registry.Register(channels.NewTelegramChannel(cfg.Telegram.BotToken, cfg.Telegram.ChatID))
	// Email channel is scaffolded but not implemented
	// registry.Register(channels.NewEmailChannel())

	for _, wc := range cfg.Webhooks {
		ch := channels.NewWebhookChannel(wc.Name, wc.URL, wc.Secret)
		registry.Register(ch)
		logger.Info("registered webhook channel", slog.String("name", string(ch.Name())))
	}

	limiter := ratelimit.NewLimiter(cfg.RateLimitPerMinute)

	queueNames := queue.NewQueueNames(cfg.Redis.KeyPrefix)
	logger.Info("queue names configured",
		slog.String("prefix", cfg.Redis.KeyPrefix),
		slog.String("notifications_queue", queueNames.Notifications),
		slog.String("task_type", queueNames.TaskType),
	)

	queueClient := queue.NewClient(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Worker.MaxRetries,
		logger,
		queueNames,
	)
	defer queueClient.Close()

	worker := queue.NewWorker(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Worker.Concurrency,
		registry,
		logger,
		queueNames,
	)

	router := api.NewRouter(cfg, limiter, queueClient, logger)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting worker", slog.Int("concurrency", cfg.Worker.Concurrency))
		if err := worker.Start(); err != nil {
			logger.Error("worker failed to start", slog.String("error", err.Error()))
		}
	}()

	go func() {
		logger.Info("starting HTTP server", slog.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	} else {
		logger.Info("HTTP server stopped")
	}

	worker.Shutdown()
	logger.Info("worker stopped")
	logger.Info("shutdown complete")
}

func setupLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
