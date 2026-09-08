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

func Test_Accept_NotifiesRequester(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	err := svc.Accept(context.Background(), hostID, r.ID)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hikerID {
		t.Errorf("expected hikerID: %v got %v", hikerID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindJoinRequestApproved {
		t.Errorf("expected event_kind: %s got %s", notification.KindJoinRequestApproved, fn.events[0].Kind)
	}
}
func Test_Request_NotifiesHost(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	if _, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{
		Role:         RoleDriver,
		SeatsOffered: 4,
		Guests:       1,
		Note:         nil,
	}); err != nil {
		t.Fatalf("expected not error but got %v", err)
	}

	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hikerID {
		t.Errorf("expected hikerID: %v got %v", hikerID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindJoinRequestCreated {
		t.Errorf("expected event_kind: %s got %s", notification.KindJoinRequestCreated, fn.events[0].Kind)
	}

}

func Test_Withdrawn_BeforeAccepted_NotifiesHost(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	if err := svc.Withdraw(context.Background(), hikerID, o.ID); err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hostID {
		t.Errorf("expected hikerID: %v got %v", hostID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindJoinRequestWithdrawn {
		t.Errorf("expected event_kind: %s got %s", notification.KindJoinRequestWithdrawn, fn.events[0].Kind)
	}

}

func Test_Withdrawn_AfterAccepted_NotifiesHost(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	if err := svc.Withdraw(context.Background(), hikerID, o.ID); err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hostID {
		t.Errorf("expected hikerID: %v got %v", hostID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindJoinRequestWithdrawn {
		t.Errorf("expected event_kind: %s got %s", notification.KindJoinRequestWithdrawn, fn.events[0].Kind)
	}

}

func Test_RemoveMember_NotifiesRemovedHiker(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	if err := svc.RemoveMember(context.Background(), hostID, r.ID); err != nil {
		t.Fatalf("expected not error got %v", err)
	}
	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != hikerID {
		t.Errorf("expected hikerID: %v got %v", hikerID, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindMemberRemoved {
		t.Errorf("expected event_kind: %s got %s", notification.KindMemberRemoved, fn.events[0].Kind)
	}

}

func Test_CancelOuting_NotifiesEachMember(t *testing.T){

}
