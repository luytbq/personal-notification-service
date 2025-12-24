package notification

import "time"

// Level represents the severity level of a notification
type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)

// ValidLevels contains all valid notification levels
var ValidLevels = map[Level]bool{
	LevelInfo:     true,
	LevelWarning:  true,
	LevelError:    true,
	LevelCritical: true,
}

// LevelPrefix returns the prefix string for a given level
func (l Level) Prefix() string {
	switch l {
	case LevelInfo:
		return "[INFO]"
	case LevelWarning:
		return "[WARNING]"
	case LevelError:
		return "[ERROR]"
	case LevelCritical:
		return "[CRITICAL]"
	default:
		return "[UNKNOWN]"
	}
}

// Channel represents a notification channel
type Channel string

const (
	ChannelTelegram Channel = "telegram"
	ChannelEmail    Channel = "email"
)

// ValidChannels contains all valid notification channels
var ValidChannels = map[Channel]bool{
	ChannelTelegram: true,
	ChannelEmail:    true,
}

// Request represents an incoming notification request
type Request struct {
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Level    Level     `json:"level"`
	Channels []Channel `json:"channel"`
	Source   string    `json:"source,omitempty"`
}

// Notification represents a notification to be sent
type Notification struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Level     Level     `json:"level"`
	Channel   Channel   `json:"channel"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source,omitempty"`
}

// Response represents the API response for a notification request
type Response struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
