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
	if fn.events[0].HikerID != hostID {
		t.Errorf("expected hikerID: %v got %v", hostID, fn.events[0].HikerID)
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

func Test_CancelOuting_NotifiesEachMember(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hiker1ID := uuid.New()
	hiker2ID := uuid.New()
	hiker3ID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hiker1ID, RequestStatusAccepted, RoleRider, f, 1)
	_ = seedJoinRequestWithSeatsOffered(o.ID, hiker2ID, RequestStatusAccepted, RoleDriver, f, 0, 3)
	_ = seedJoinRequest(o.ID, hiker3ID, RequestStatusRequested, RoleRider, f, 1)
	seedMember(hiker1ID, "mahmut", "experienced", f)
	seedMember(hiker2ID, "celal", "beginner", f)
	seedMember(hiker3ID, "bulbul", "beginner", f)
	if err := svc.Cancel(context.Background(), hostID, o.ID); err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if len(fn.events) != 3 {
		t.Fatalf("expected 3 notification, got %d", len(fn.events))
	}
	got := make(map[uuid.UUID]bool)
	for _, e := range fn.events {
		if e.Kind != notification.KindOutingCancelled {
			t.Errorf("expected kind %s got %s", notification.KindOutingCancelled, e.Kind)
		}
		got[e.HikerID] = true
	}
	for _, id := range []uuid.UUID{hiker1ID, hiker2ID, hiker3ID} {
		if !got[id] {
			t.Errorf("hiker %v not notified", id)
		}
	}
	if got[hostID] {
		t.Error("host notified of own cancel")
	}

}

func Test_UpdateOuting_NotifiesEachMember(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hiker1ID := uuid.New()
	hiker2ID := uuid.New()
	hiker3ID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hiker1ID, RequestStatusAccepted, RoleRider, f, 1)
	_ = seedJoinRequestWithSeatsOffered(o.ID, hiker2ID, RequestStatusAccepted, RoleDriver, f, 0, 3)
	_ = seedJoinRequest(o.ID, hiker3ID, RequestStatusRequested, RoleRider, f, 1)
	seedMember(hiker1ID, "mahmut", "experienced", f)
	seedMember(hiker2ID, "celal", "beginner", f)
	seedMember(hiker3ID, "bulbul", "beginner", f)
	title, destination, meetLabel, maxsize, cost, difficulty, pace, note := "Agri Dagi", "Agri", "Agri Dagi etegi", 12, 250, DifficultyHard, PaceRelaxed, "slowly but surely"
	if _, err := svc.Update(context.Background(), hostID, o.ID, UpdateInput{
		Title:            &title,
		Destination:      &destination,
		MeetLabel:        &meetLabel,
		MaxSize:          &maxsize,
		CostPerSeatCents: &cost,
		Difficulty:       &difficulty,
		Pace:             &pace,
		Notes:            &note,
	}); err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fn.events) != 3 {
		t.Fatalf("expected 3 notification, got %d", len(fn.events))
	}
	got := make(map[uuid.UUID]bool)
	for _, e := range fn.events {
		if e.Kind != notification.KindOutingUpdated {
			t.Errorf("expected kind %s got %s", notification.KindOutingUpdated, e.Kind)
		}
		got[e.HikerID] = true
	}
	for _, id := range []uuid.UUID{hiker1ID, hiker2ID, hiker3ID} {
		if !got[id] {
			t.Errorf("hiker %v not notified", id)
		}
	}
	if got[hostID] {
		t.Error("host notified of own cancel")
	}

}

func Test_Rerequest_NotifiesHost(t *testing.T) {
	svc, f, fn := newTestService(t)

	hostID := uuid.New()
	hiker1ID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hiker1ID, RequestStatusAccepted, RoleRider, f, 1)
	seedMember(hiker1ID, "mahmut", "experienced", f)

	if err := svc.Withdraw(context.Background(), hiker1ID, o.ID); err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if _, err := svc.RequestJoin(context.Background(), hiker1ID, o.ID, JoinInput{
		Role:         RoleRider,
		SeatsOffered: 0,
		Guests:       1,
		Note:         nil,
	}); err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if len(fn.events) != 2 {
		t.Errorf("expected 2 got %d", len(fn.events))
	}
	if fn.events[0].Kind != "join_request_withdrawn" {
		t.Errorf("expected join_request_withdrawn got %s", fn.events[0].Kind)
	}
	if fn.events[1].Kind != "join_request_created" {
		t.Errorf("expected join_request_created got %s", fn.events[1].Kind)
	}

}
