// Package notify provides the notification dispatcher and backend implementations
// (Slack, GitHub Issues, Jira) for sending check alert notifications.
package notify

import (
	"context"

	"github.com/drop-the-mic/operator/internal/state"
)

// Event contains the information needed to send a notification.
type Event struct {
	PolicyRef   string
	CheckID     string
	Description string
	Severity    string
	Verdict     string
	Reasoning   string
	Transition  state.Transition
}

// Notifier is the interface for notification backends.
type Notifier interface {
	// Send sends a notification for the given event.
	Send(ctx context.Context, event Event) error
	// Name returns the notifier name for logging.
	Name() string
}

// Dispatcher fans out notifications to all configured notifiers.
type Dispatcher struct {
	notifiers []Notifier
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(notifiers ...Notifier) *Dispatcher {
	return &Dispatcher{notifiers: notifiers}
}

// Dispatch sends an event to all notifiers.
func (d *Dispatcher) Dispatch(ctx context.Context, event Event) []error {
	var errs []error
	for _, n := range d.notifiers {
		if err := n.Send(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
