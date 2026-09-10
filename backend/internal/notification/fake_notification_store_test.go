// backend/internal/notification/fake_notification_store_test.go

package notification

import (
	"context"
	"time"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/google/uuid"
)

type fakeNotificationStore struct {
	events []*Event
	err    error
}

func (f *fakeNotificationStore) ListForHiker(ctx context.Context, hikerID uuid.UUID, limit, offset int) ([]*Event, error) {
	return nil, nil // unused by dispatcher tests
}
func (f *fakeNotificationStore) MarkRead(ctx context.Context, hikerID, id uuid.UUID) error {
	return nil
}
func (f *fakeNotificationStore) MarkAllRead(ctx context.Context, hikerID uuid.UUID) error { return nil }
func (f *fakeNotificationStore) UnreadCount(ctx context.Context, hikerID uuid.UUID) (int, error) {
	return 0, nil
}

func (f *fakeNotificationStore) ListUnsent(ctx context.Context, limit int) ([]*Unsent, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*Unsent
	for _, e := range f.events {
		if e.EmailedAt != nil {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, &Unsent{Event: *e, Email: "test@dev"})
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
	} else {
		logger.Warn(context.Background(), "failed to mark")
	}
	return nil
}

func (f *fakeNotificationStore) Insert(ctx context.Context, e *Event) error {
	if f.err != nil {
		return f.err
	}
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	f.events = append(f.events, e)
	return nil
}

func (f *fakeNotificationStore) getEventByID(id uuid.UUID) (*Event, bool) {
	for _, event := range f.events {
		if event.ID == id {
			return event, true
		}
	}
	return nil, false
}

var _ Storage = (*fakeNotificationStore)(nil)
