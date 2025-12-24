package channels

import (
	"context"
	"errors"

	"github.com/luytbq/personal-notification-service/internal/notification"
)

// ErrEmailNotImplemented is returned when email sending is attempted
var ErrEmailNotImplemented = errors.New("email channel not implemented")

// EmailChannel sends notifications via email
// TODO: Implement email sending using SMTP or an email service provider
type EmailChannel struct {
	// smtpHost     string
	// smtpPort     int
	// smtpUser     string
	// smtpPassword string
	// fromAddress  string
	// toAddress    string
}

// NewEmailChannel creates a new Email channel
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{}
}

// Name returns the channel name
func (e *EmailChannel) Name() notification.Channel {
	return notification.ChannelEmail
}

// Send sends a notification via email
func (e *EmailChannel) Send(ctx context.Context, n *notification.Notification) error {
	// TODO: Implement email sending
	// Example implementation:
	// 1. Format email with subject: "[LEVEL] Title" and body: Message
	// 2. Append source if provided: "Source: <source>"
	// 3. Connect to SMTP server
	// 4. Send email
	return ErrEmailNotImplemented
}
