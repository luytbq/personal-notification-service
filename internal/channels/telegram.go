package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/luytbq/personal-notification-service/internal/notification"
)

const (
	telegramAPIURL = "https://api.telegram.org/bot%s/sendMessage"
)

// TelegramChannel sends notifications via Telegram Bot API
type TelegramChannel struct {
	botToken string
	chatID   string
	client   *http.Client
}

// NewTelegramChannel creates a new Telegram channel
func NewTelegramChannel(botToken, chatID string) *TelegramChannel {
	return &TelegramChannel{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the channel name
func (t *TelegramChannel) Name() notification.Channel {
	return notification.ChannelTelegram
}

// telegramMessage represents the Telegram sendMessage request
type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// telegramResponse represents the Telegram API response
type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

// Send sends a notification via Telegram
func (t *TelegramChannel) Send(ctx context.Context, n *notification.Notification) error {
	// Format message with level prefix
	// TODO: Try other parse modes (Markdown, HTML) for richer formatting
	text := fmt.Sprintf("%s %s\n\n%s", n.Level.Prefix(), n.Title, n.Message)

	msg := telegramMessage{
		ChatID: t.chatID,
		Text:   text,
		// ParseMode left empty for plain text
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	url := fmt.Sprintf(telegramAPIURL, t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp telegramResponse
	if err := json.Unmarshal(respBody, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse telegram response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s (code: %d)", telegramResp.Description, telegramResp.ErrorCode)
	}

	return nil
}
