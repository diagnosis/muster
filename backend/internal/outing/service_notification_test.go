// backend/internal/outing/service_notification_test.go
package outing

import (
	"context"
	"testing"

	"github.com/diagnosis/muster/internal/notification"
	"github.com/google/uuid"
)

func newTestService(t *testing.T) (*Service, *fakeStore, *fakeNotificationStore) {
	t.Helper()
	f := newFakeStore()
	fn := &fakeNotificationStore{}
	return NewService(f, fn), f, fn
}

func Test_Decline_NotifiesRequester(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	err := svc.Decline(context.Background(), hostID, r.ID)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hikerID {
		t.Errorf("expected hikerID: %v got %v", hikerID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindJoinRequestDeclined {
		t.Errorf("expected event_kind: %s got %s", notification.KindJoinRequestDeclined, fn.events[0].Kind)
	}

}
