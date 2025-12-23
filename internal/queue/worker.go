package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/luytbq/personal-notification-service/internal/channels"
	"github.com/luytbq/personal-notification-service/internal/notification"
)

// Worker handles processing of notification tasks
type Worker struct {
	server     *asynq.Server
	mux        *asynq.ServeMux
	registry   *channels.Registry
	logger     *slog.Logger
	queueNames *QueueNames
}

// NewWorker creates a new worker
func NewWorker(redisAddr, redisPassword string, redisDB, concurrency int, registry *channels.Registry, logger *slog.Logger, queueNames *QueueNames) *Worker {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				queueNames.Notifications: 10, // Priority weight
			},
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				// Exponential backoff: 10s, 20s, 40s, 80s, 160s
				return time.Duration(10<<uint(n-1)) * time.Second
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				retried, _ := asynq.GetRetryCount(ctx)
				maxRetry, _ := asynq.GetMaxRetry(ctx)

				var payload NotificationPayload
				if jsonErr := json.Unmarshal(task.Payload(), &payload); jsonErr == nil {
					if retried >= maxRetry {
						// This is the final failure - log for dead letter tracking
						logger.Error("notification moved to dead letter queue",
							slog.String("notification_id", payload.ID),
							slog.String("channel", string(payload.Channel)),
							slog.String("error", err.Error()),
							slog.Int("attempts", retried+1),
							slog.Any("payload", payload),
						)
					} else {
						logger.Warn("notification task failed, will retry",
							slog.String("notification_id", payload.ID),
							slog.String("channel", string(payload.Channel)),
							slog.String("error", err.Error()),
							slog.Int("attempt", retried+1),
							slog.Int("max_retries", maxRetry),
						)
					}
				}
			}),
		},
	)

	mux := asynq.NewServeMux()

	w := &Worker{
		server:     server,
		mux:        mux,
		registry:   registry,
		logger:     logger,
		queueNames: queueNames,
	}

	// Register handlers
	mux.HandleFunc(queueNames.TaskType, w.handleNotification)

	return w
}

// Start starts the worker
func (w *Worker) Start() error {
	return w.server.Start(w.mux)
}

// Shutdown gracefully shuts down the worker
func (w *Worker) Shutdown() {
	w.server.Shutdown()
}

// handleNotification processes a notification task
func (w *Worker) handleNotification(ctx context.Context, task *asynq.Task) error {
	start := time.Now()

	payload, err := ParseNotificationPayload(task)
	if err != nil {
		w.logger.Error("failed to parse notification payload",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	// Get the channel
	ch, ok := w.registry.Get(payload.Channel)
	if !ok {
		w.logger.Error("unknown channel",
			slog.String("notification_id", payload.ID),
			slog.String("channel", string(payload.Channel)),
		)
		return fmt.Errorf("unknown channel: %s", payload.Channel)
	}

	// Convert payload to notification
	n := &notification.Notification{
		ID:        payload.ID,
		Title:     payload.Title,
		Message:   payload.Message,
		Level:     payload.Level,
		Channel:   payload.Channel,
		APIKey:    payload.APIKey,
		CreatedAt: payload.CreatedAt,
	}

	// Send the notification
	if err := ch.Send(ctx, n); err != nil {
		w.logger.Error("notification failed",
			slog.String("notification_id", n.ID),
			slog.String("channel", string(n.Channel)),
			slog.String("status", "failed"),
			slog.String("error", err.Error()),
			slog.Duration("latency", time.Since(start)),
		)
		return err
	}

	w.logger.Info("notification sent",
		slog.String("notification_id", n.ID),
		slog.String("channel", string(n.Channel)),
		slog.String("status", "sent"),
		slog.Duration("latency", time.Since(start)),
	)

	return nil
}
