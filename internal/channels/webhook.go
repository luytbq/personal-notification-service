package channels

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/luytbq/personal-notification-service/internal/notification"
)

// WebhookChannel sends notifications via HTTP POST with HMAC-SHA256 signature
type WebhookChannel struct {
	name   notification.Channel
	url    string
	secret string
	client *http.Client
}

// NewWebhookChannel creates a new WebhookChannel
func NewWebhookChannel(name, url, secret string) *WebhookChannel {
	return &WebhookChannel{
		name:   notification.Channel(notification.ChannelWebhookPrefix + name),
		url:    url,
		secret: secret,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the channel name (e.g. "webhook:notifeed")
func (w *WebhookChannel) Name() notification.Channel {
	return w.name
}

// Send POSTs the notification as JSON with an HMAC-SHA256 signature header
func (w *WebhookChannel) Send(ctx context.Context, n *notification.Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PNS-Signature", "sha256="+computeHMAC(body, w.secret))

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
