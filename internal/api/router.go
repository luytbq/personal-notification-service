package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/luytbq/personal-notification-service/internal/config"
	"github.com/luytbq/personal-notification-service/internal/notification"
	"github.com/luytbq/personal-notification-service/internal/queue"
	"github.com/luytbq/personal-notification-service/internal/ratelimit"
)

// NewRouter creates and configures the HTTP router
func NewRouter(cfg *config.Config, limiter *ratelimit.Limiter, client *queue.Client, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RecoveryMiddleware(logger))
	r.Use(LoggingMiddleware(logger))

	// Create handler
	validator := notification.NewValidator()
	handler := NewHandler(validator, limiter, client, logger)

	// Public routes (no auth required)
	r.Get("/notify/health", handler.HandleHealth)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(cfg, logger))
		r.Post("/notify", handler.HandleNotify)
	})

	return r
}
