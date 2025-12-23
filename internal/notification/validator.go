package notification

import (
	"errors"
	"strings"
)

var (
	ErrEmptyTitle     = errors.New("title is required")
	ErrEmptyMessage   = errors.New("message is required")
	ErrInvalidLevel   = errors.New("invalid level: must be one of info, warning, error, critical")
	ErrEmptyChannels  = errors.New("at least one channel is required")
	ErrInvalidChannel = errors.New("invalid channel: must be one of telegram, email")
)

// Validator validates notification requests
type Validator struct{}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a notification request
func (v *Validator) Validate(req *Request) error {
	// Validate title
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}

	// Validate message
	if strings.TrimSpace(req.Message) == "" {
		return ErrEmptyMessage
	}

	// Validate level
	if !ValidLevels[req.Level] {
		return ErrInvalidLevel
	}

	// Validate channels
	if len(req.Channels) == 0 {
		return ErrEmptyChannels
	}

	for _, ch := range req.Channels {
		if !ValidChannels[ch] {
			return ErrInvalidChannel
		}
	}

	return nil
}
