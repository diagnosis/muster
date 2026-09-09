package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/mailer"
	"github.com/diagnosis/muster/internal/email"
)

// Dispatcher drains the notification outbox: it periodically finds events
// whose email hasn't been sent and sends them. One per process, started in main.
type Dispatcher struct {
	store    Storage
	mailer   mailer.Mailer
	interval time.Duration
	baseURL  string
}

// NewDispatcher returns a Dispatcher; interval defaults to 30s when zero.
func NewDispatcher(store Storage, mailer mailer.Mailer, interval time.Duration, baseURL string) *Dispatcher {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &Dispatcher{
		store:    store,
		mailer:   mailer,
		interval: interval,
		baseURL:  baseURL,
	}
}

// Run drains on each tick until ctx is cancelled. Transient drain failures
// are logged and retried next tick — only ctx cancellation stops the loop.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.drain(ctx); err != nil {
				logger.Error(ctx, "failed to drain", "err", err)
			}
		}
	}
}

func (d *Dispatcher) drain(ctx context.Context) error {
	unsents, err := d.store.ListUnsent(ctx, 50)
	if err != nil {
		return err
	}
	for _, u := range unsents {
		c := d.contentFor(ctx, u)
		html, err := email.Render(c)
		if err != nil {
			logger.Warn(ctx, "failed to render email html", "err", err)
			continue
		}
		if err = d.mailer.Send(ctx, []string{u.Email}, c.Subject, html); err != nil {
			logger.Warn(ctx, "failed to send email", "err", err)
		} else {
			if err = d.store.MarkEmailed(ctx, u.Event.ID); err != nil {
				logger.Warn(ctx, "failed to mark as emailed", "err", err)
			}
		}
	}
	return nil
}

func (d *Dispatcher) contentFor(ctx context.Context, u *Unsent) email.Content {
	title, _ := u.Event.Payload["outing_title"].(string)
	var id string
	if v, ok := u.Event.Payload["outing_id"].(string); ok {
		id = v
	}
	switch u.Event.Kind {
	case KindJoinRequestApproved:
		return email.Content{
			Subject:  "Muster - join request approved",
			Heading:  "You're in",
			Body:     fmt.Sprintf("Your request to join %s was approved.", title),
			CTALabel: "View outing",
			CTAURL:   d.outingURL(id),
		}
	case KindJoinRequestDeclined:
		return email.Content{
			Subject: "Muster - join request declined",
			Heading: "You're out",
			Body:    fmt.Sprintf("Your request to join %s was declined.", title),
		}
	case KindJoinRequestWithdrawn:
		return email.Content{
			Subject:  "Muster - join request withdrawn",
			Heading:  "Member withdrew",
			Body:     fmt.Sprintf("A member withdrew their request from %s.", title),
			CTALabel: "View outing",
			CTAURL:   d.outingURL(id),
		}
	case KindJoinRequestCreated:
		return email.Content{
			Subject:  "Muster - join request created",
			Heading:  "New Join Request",
			Body:     fmt.Sprintf("A member request to join %s.", title),
			CTALabel: "View outing",
			CTAURL:   d.outingURL(id),
		}
	case KindMemberRemoved:
		return email.Content{
			Subject: "Muster - member removed",
			Heading: "You're removed",
			Body:    fmt.Sprintf("You are removed from incoming outing %s", title),
		}
	case KindOutingCancelled:
		return email.Content{
			Subject: "Muster - outing cancelled",
			Heading: "Outing cancelled",
			Body:    fmt.Sprintf("the outing %s was cancelled", title),
		}
	case KindOutingUpdated:
		return email.Content{
			Subject:  "Muster - outing updated",
			Heading:  "Outing updated",
			Body:     fmt.Sprintf("the outing %s was updated", title),
			CTALabel: "View outing",
			CTAURL:   d.outingURL(id),
		}
	}
	logger.Warn(ctx, "contentFor: unhandled kind", "kind", u.Event.Kind)
	return email.Content{
		Subject: "Muster notification",
		Heading: "Muster",
		Body:    "You have a new notification. Open Muster to see the details.",
	}

}
func (d *Dispatcher) outingURL(id string) string {
	return fmt.Sprintf("%s/outings/%s", d.baseURL, id)
}
