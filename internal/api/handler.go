package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/luytbq/personal-notification-service/internal/notification"
	"github.com/luytbq/personal-notification-service/internal/queue"
	"github.com/luytbq/personal-notification-service/internal/ratelimit"
)

// Handler handles HTTP requests
type Handler struct {
	validator *notification.Validator
	limiter   *ratelimit.Limiter
	client    *queue.Client
	logger    *slog.Logger
}

// NewHandler creates a new Handler
func NewHandler(validator *notification.Validator, limiter *ratelimit.Limiter, client *queue.Client, logger *slog.Logger) *Handler {
	return &Handler{
		validator: validator,
		limiter:   limiter,
		client:    client,
		logger:    logger,
	}
}

// HandleNotify handles POST /notify requests
func (h *Handler) HandleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get API key from context
	apiKey := GetAPIKey(r.Context())
	if apiKey == "" {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req notification.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body",
			slog.String("error", err.Error()),
		)
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := h.validator.Validate(&req); err != nil {
		h.logger.Warn("validation failed",
			slog.String("error", err.Error()),
		)
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check rate limits
	allowed, blockedChannel := CheckRateLimit(h.limiter, apiKey, req.Channels)
	if !allowed {
		h.logger.Warn("rate limit exceeded",
			slog.String("api_key", maskAPIKey(apiKey)),
			slog.String("blocked_channel", blockedChannel),
		)
		WriteError(w, http.StatusTooManyRequests, fmt.Sprintf("rate limit exceeded for channel: %s", blockedChannel))
		return
	}

	// Enqueue notifications
	taskIDs, err := h.client.Enqueue(&req, apiKey)
	if err != nil {
		h.logger.Error("failed to enqueue notification",
			slog.String("error", err.Error()),
		)
		WriteError(w, http.StatusInternalServerError, "failed to queue notification")
		return
	}

	// Return first task ID (or comma-separated if multiple)
	responseID := strings.Join(taskIDs, ",")

	WriteJSON(w, http.StatusAccepted, notification.Response{
		Status: "queued",
		ID:     responseID,
	})
}

// HandleHealth handles GET /health requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Helper functions

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, notification.ErrorResponse{Error: message})
}

// maskAPIKey masks an API key for logging (shows first 4 and last 4 chars)
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
