// backend/internal/outing/fake_notification_store_test.go

package outing

import (
	"context"
	"time"

	"github.com/diagnosis/muster/internal/notification"
	"github.com/google/uuid"
)

type fakeNotificationStore struct {
	events []*notification.Event
	err    error
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

var _ notification.Storage = (*fakeNotificationStore)(nil)
