// backend/internal/notification/dispacher_test.go
package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test_drain_sendsUnsentAndStamps(t *testing.T) {
	f := &fakeNotificationStore{}
	fm := &fakeMailer{}
	d := NewDispatcher(f, fm, 30*time.Second, "https://test.com")

	hiker := uuid.New()

	for i := 0; i < 3; i++ {
		_ = f.Insert(context.Background(), &Event{HikerID: hiker, Kind: KindJoinRequestDeclined, Payload: map[string]any{"outing_title": "X"}})
	}

	sent := &Event{HikerID: hiker, Kind: KindOutingCancelled, Payload: map[string]any{"outing_title": "Y"}}
	_ = f.Insert(context.Background(), sent)
	now := time.Now()
	sent.EmailedAt = &now

	err := d.drain(context.Background())
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(fm.sent) != 3 {
		t.Fatalf("expected 3 sends, got %d", len(fm.sent))
	}
	for _, e := range f.events {
		if e.Kind == KindJoinRequestDeclined && e.EmailedAt == nil {
			t.Error("sent event was not stamped — will re-send next tick")
		}
	}
}
