package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/luytbq/personal-notification-service/internal/config"
	"github.com/luytbq/personal-notification-service/internal/notification"
	"github.com/luytbq/personal-notification-service/internal/ratelimit"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// APIKeyContextKey is the context key for the authenticated API key
	APIKeyContextKey contextKey = "api_key"
)

// GetAPIKey extracts the API key from context
func GetAPIKey(ctx context.Context) string {
	if v := ctx.Value(APIKeyContextKey); v != nil {
		return v.(string)
	}
	return ""
}

// AuthMiddleware validates the X-API-Key header
func AuthMiddleware(cfg *config.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if apiKey == "" {
				logger.Warn("missing API key",
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("path", r.URL.Path),
				)
				WriteError(w, http.StatusUnauthorized, "missing X-API-Key header")
				return
			}

			if !cfg.ValidateAPIKey(apiKey) {
				logger.Warn("invalid API key",
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("path", r.URL.Path),
				)
				WriteError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Store API key in context for later use
			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CheckRateLimit checks rate limits for the given API key and channels
// Returns true if allowed, false if rate limited along with the blocked channel
func CheckRateLimit(limiter *ratelimit.Limiter, apiKey string, channels []notification.Channel) (bool, string) {
	for _, ch := range channels {
		if !limiter.Allow(apiKey, string(ch)) {
			return false, string(ch)
		}
	}
	return true, ""
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for health checks to reduce noise
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			logger.Info("request received",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						slog.Any("error", err),
						slog.String("path", r.URL.Path),
					)
					WriteError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
