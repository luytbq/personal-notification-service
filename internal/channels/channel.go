package channels

import (
	"context"

	"github.com/luytbq/personal-notification-service/internal/notification"
)

// Channel defines the interface for notification channels
type Channel interface {
	// Name returns the channel name
	Name() notification.Channel

	// Send sends a notification through this channel
	Send(ctx context.Context, n *notification.Notification) error
}

// Registry holds all registered notification channels
type Registry struct {
	channels map[notification.Channel]Channel
}

// NewRegistry creates a new channel registry
func NewRegistry() *Registry {
	return &Registry{
		channels: make(map[notification.Channel]Channel),
	}
}

// Register registers a channel
func (r *Registry) Register(ch Channel) {
	r.channels[ch.Name()] = ch
}

// Get returns a channel by name
func (r *Registry) Get(name notification.Channel) (Channel, bool) {
	ch, ok := r.channels[name]
	return ch, ok
}
