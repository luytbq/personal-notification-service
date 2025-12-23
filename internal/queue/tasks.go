package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/luytbq/personal-notification-service/internal/notification"
)

// Task type suffix (will be prefixed with Redis key prefix)
const (
	TaskTypeSuffix = "notification:send"
)

// QueueNames holds the prefixed queue and task names
type QueueNames struct {
	Notifications string
	TaskType      string
}

// NewQueueNames creates queue names with the given prefix
func NewQueueNames(prefix string) *QueueNames {
	return &QueueNames{
		Notifications: fmt.Sprintf("%s:notifications", prefix),
		TaskType:      fmt.Sprintf("%s:%s", prefix, TaskTypeSuffix),
	}
}

// NotificationPayload represents the payload for a notification task
type NotificationPayload struct {
	ID        string               `json:"id"`
	Title     string               `json:"title"`
	Message   string               `json:"message"`
	Level     notification.Level   `json:"level"`
	Channel   notification.Channel `json:"channel"`
	APIKey    string               `json:"api_key"`
	CreatedAt time.Time            `json:"created_at"`
}

// NewNotificationTask creates a new notification task
func NewNotificationTask(n *notification.Notification, taskType string) (*asynq.Task, error) {
	payload := NotificationPayload{
		ID:        n.ID,
		Title:     n.Title,
		Message:   n.Message,
		Level:     n.Level,
		Channel:   n.Channel,
		APIKey:    n.APIKey,
		CreatedAt: n.CreatedAt,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(taskType, data), nil
}

// ParseNotificationPayload parses a notification task payload
func ParseNotificationPayload(task *asynq.Task) (*NotificationPayload, error) {
	var payload NotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
