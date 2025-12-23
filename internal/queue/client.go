package queue

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/luytbq/personal-notification-service/internal/notification"
)

// Client wraps the asynq client for enqueueing notifications
type Client struct {
	client     *asynq.Client
	maxRetries int
	logger     *slog.Logger
	queueNames *QueueNames
}

// NewClient creates a new queue client
func NewClient(redisAddr, redisPassword string, redisDB, maxRetries int, logger *slog.Logger, queueNames *QueueNames) *Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	return &Client{
		client:     client,
		maxRetries: maxRetries,
		logger:     logger,
		queueNames: queueNames,
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue adds a notification to the queue
// Returns the task ID and any error
func (c *Client) Enqueue(req *notification.Request, apiKey string) ([]string, error) {
	var taskIDs []string
	now := time.Now()

	for _, channel := range req.Channels {
		n := &notification.Notification{
			ID:        uuid.New().String(),
			Title:     req.Title,
			Message:   req.Message,
			Level:     req.Level,
			Channel:   channel,
			APIKey:    apiKey,
			CreatedAt: now,
		}

		task, err := NewNotificationTask(n, c.queueNames.TaskType)
		if err != nil {
			c.logger.Error("failed to create task",
				slog.String("notification_id", n.ID),
				slog.String("channel", string(channel)),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		info, err := c.client.Enqueue(task,
			asynq.MaxRetry(c.maxRetries),
			asynq.Queue(c.queueNames.Notifications),
			asynq.TaskID(n.ID),
		)
		if err != nil {
			c.logger.Error("failed to enqueue task",
				slog.String("notification_id", n.ID),
				slog.String("channel", string(channel)),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to enqueue task: %w", err)
		}

		c.logger.Info("notification queued",
			slog.String("notification_id", n.ID),
			slog.String("channel", string(channel)),
			slog.String("queue", info.Queue),
		)

		taskIDs = append(taskIDs, n.ID)
	}

	return taskIDs, nil
}
