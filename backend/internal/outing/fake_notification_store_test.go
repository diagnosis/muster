// backend/internal/outing/fake_notification_store_test.go

package outing

import (
	"context"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/notification"
	"github.com/google/uuid"
)

type fakeNotificationStore struct {
	events []*notification.Event
	err    error
}

func (f *fakeNotificationStore) ListUnsent(ctx context.Context, limit int) ([]*notification.Unsent, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*notification.Unsent
	for _, e := range f.events {
		if e.EmailedAt != nil {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, &notification.Unsent{Event: *e, Email: "test@dev"})
	}
	return out, nil
}

func (f *fakeNotificationStore) MarkEmailed(ctx context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if e, ok := f.getEventByID(id); ok {
		now := time.Now()
		e.EmailedAt = &now
		return nil
	} else {
		return apperr.NotFound("no event found", "no event found")
	}

}

func (f *fakeNotificationStore) Insert(ctx context.Context, e *notification.Event) error {
	if f.err != nil {
		return f.err
	}
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	f.events = append(f.events, e)
	return nil
}

func (f *fakeNotificationStore) getEventByID(id uuid.UUID) (*notification.Event, bool) {
	if f.err != nil {
		return nil, false
	}
	for _, event := range f.events {
		if event.ID == id {
			return event, true
		}
	}
	return nil, false
}
func (f *fakeNotificationStore) ListForHiker(ctx context.Context, hikerID uuid.UUID, limit, offset int) ([]*notification.Event, error) {
	return nil, nil // unused by dispatcher tests
}
func (f *fakeNotificationStore) MarkRead(ctx context.Context, hikerID, id uuid.UUID) error {
	return nil
}
func (f *fakeNotificationStore) MarkAllRead(ctx context.Context, hikerID uuid.UUID) error { return nil }
func (f *fakeNotificationStore) UnreadCount(ctx context.Context, hikerID uuid.UUID) (int, error) {
	return 0, nil
}

var _ notification.Storage = (*fakeNotificationStore)(nil)
