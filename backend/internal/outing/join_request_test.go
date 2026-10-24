// backend/internal/outing/join_request_test.go
// Tests for a hiker asking to join, withdrawing, and editing a pending request.
package outing

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/notification"
	"github.com/google/uuid"
)

func Test_RequestJoin_FirstRequestCreated(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	hiker := uuid.New()
	r, err := svc.RequestJoin(context.Background(), hiker, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("RequestJoin: unexpected error: %v", err)
	}
	if r.Status != RequestStatusRequested {
		t.Errorf("status = %q, want %q", r.Status, RequestStatusRequested)
	}
	if _, ok := f.requests[r.ID]; !ok {
		t.Error("request was not saved to the store")
	}
}

func Test_RequestJoin_HostCannotSelfJoin(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hostID, o.ID, JoinInput{Role: RoleRider})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func Test_RequestJoin_RiderWithSeats(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, SeatsOffered: 2})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func Test_RequestJoin_DriverZeroSeats(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleDriver, SeatsOffered: 0})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_RequestJoin_TooManyGuests(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 4})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_RequestJoin_CancelledOuting(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusCancelled, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 2})

	wantStatus(t, err, apperr.CodeConflict)
}

func Test_RequestJoin_DeclinedIsTerminal(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	r, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	f.requests[r.ID].Status = RequestStatusDeclined

	_, err = svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_RequestJoin_DuplicateActive(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err = svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_RequestJoin_AfterWithdrawTakesNewDetails(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	r, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	f.requests[r.ID].Status = RequestStatusWithdrawn

	newNote := "new note"

	r2, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{
		Role:         RoleDriver,
		SeatsOffered: 2,
		Guests:       1,
		Note:         &newNote,
	})
	if err != nil {
		t.Fatalf("re-request after withdraw: %v", err)
	}
	if r2.Status != RequestStatusRequested {
		t.Errorf("returned status = %q, want %q", r2.Status, RequestStatusRequested)
	}
	if f.requests[r.ID].Status != RequestStatusRequested {
		t.Errorf("stored status = %q, want %q", f.requests[r.ID].Status, RequestStatusRequested)
	}
	if r2.Role != RoleDriver {
		t.Errorf("role = %q, want driver", r2.Role)
	}
	if r2.SeatsOffered != 2 {
		t.Errorf("seats = %d, want 2", r2.SeatsOffered)
	}
	if r2.Guests != 1 {
		t.Errorf("guests = %d, want 1", r2.Guests)
	}
}

func Test_Withdraw_AcceptedMayWithdraw(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 2, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 0)

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	if err != nil {
		t.Fatalf("Withdraw: unexpected error: %v", err)
	}
	if r.Status != RequestStatusWithdrawn {
		t.Errorf("stored status = %q, want %q", r.Status, RequestStatusWithdrawn)
	}
}

func Test_Withdraw_RequestedOK(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 2, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 0)

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	if err != nil {
		t.Fatalf("Withdraw: unexpected error: %v", err)
	}
	if r.Status != RequestStatusWithdrawn {
		t.Errorf("stored status = %q, want %q", r.Status, RequestStatusWithdrawn)
	}
}

func Test_Withdraw_DeclinedConflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 2, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusDeclined, RoleRider, f, 0)

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	wantStatus(t, err, apperr.CodeConflict)
	if r.Status != RequestStatusDeclined {
		t.Errorf("a refused withdraw must not change the row: got %q, want %q", r.Status, RequestStatusDeclined)
	}
}

func Test_Withdraw_Phase(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	member := uuid.New()
	phases := phaseCases(time.Now())
	for _, status := range []RequestStatus{RequestStatusAccepted, RequestStatusRequested} {
		for _, pp := range phases {
			t.Run(string(status)+" - "+pp.name, func(t *testing.T) {
				outing := seedOuting(6, 4, StatusOpen, host, f, withStart(pp.startAt), withEnd(pp.endsAt))
				_ = seedJoinRequest(outing.ID, member, status, "rider", f, 0)
				err := svc.Withdraw(context.Background(), member, outing.ID)
				if pp.wantErr {
					wantStatus(t, err, apperr.CodeConflict)
				} else if err != nil {
					t.Errorf("expected no error got %v", err)
				}
			})
		}
	}
}

func Test_RequestJoin_StampsRequestedAt(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1, m2 := uuid.New(), uuid.New()

	o := seedOuting(6, 4, StatusOpen, host, f)
	ji := JoinInput{
		Role:         RoleDriver,
		SeatsOffered: 3,
		Guests:       1,
		Note:         nil,
	}
	r, err := svc.RequestJoin(context.Background(), m1, o.ID, ji)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if r.RequestedAt.IsZero() {
		t.Errorf("first request: RequestedAt is zero")
	}
	// case 2
	old := time.Now().Add(-48 * time.Hour)
	r = seedJoinRequest(o.ID, m2, RequestStatusWithdrawn, RoleDriver, f, 0)
	r.RequestedAt = old

	r, err = svc.RequestJoin(context.Background(), m2, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("re-request: expected no error got %v", err)
	}
	if !r.RequestedAt.After(old) {
		t.Errorf("re-request: RequestedAt should move forward, old %v got %v", old, r.RequestedAt)
	}
}

func Test_EditRequest_Table(t *testing.T) {

	svc, f, fn, _ := newTestService(t)
	host := uuid.New()

	type in struct {
		role  Role
		guest int
		seats int
		note  string
	}
	badRequest := apperr.CodeBadRequest

	now := time.Now()
	cases := []struct {
		name    string
		seeded  in
		edit    in
		want    *apperr.Status
		restamp bool
		notify  bool
	}{
		{name: "rider adds guest", seeded: in{role: RoleRider, guest: 0, note: "", seats: 0},
			edit: in{role: RoleRider, guest: 1, seats: 0, note: ""}, want: nil, restamp: true, notify: true},
		{name: "rider drops guest", seeded: in{role: RoleRider, guest: 2, seats: 0, note: ""}, edit: in{
			role:  RoleRider,
			guest: 1,
			seats: 0,
			note:  "",
		}, restamp: false, notify: true},
		{name: "driver offers fewer seats", seeded: in{role: RoleDriver, seats: 3, note: ""}, edit: in{
			role:  RoleDriver,
			guest: 0,
			seats: 2,
			note:  "",
		}, want: nil, restamp: true, notify: true},
		{name: "driver offers more seats", seeded: in{role: RoleDriver, seats: 2, guest: 0, note: ""}, edit: in{
			role:  RoleDriver,
			guest: 0,
			seats: 3,
			note:  "",
		}, want: nil, restamp: false, notify: true},
		{name: "driver becomes rider", seeded: in{
			role:  RoleDriver,
			guest: 0,
			seats: 2,
			note:  "",
		}, edit: in{
			role:  RoleRider,
			guest: 1,
			seats: 0,
			note:  "",
		}, want: nil, restamp: true, notify: true},
		{name: "rider becomes driver", seeded: in{
			role:  RoleRider,
			guest: 2,
			seats: 0,
			note:  "",
		}, edit: in{
			role:  RoleDriver,
			guest: 0,
			seats: 2,
			note:  "",
		}, want: nil, restamp: false, notify: true},
		{name: "no-op", seeded: in{role: RoleRider, guest: 1}, edit: in{role: RoleRider, guest: 1}, want: nil, restamp: false, notify: false},
		{name: "note only", seeded: in{role: RoleRider, guest: 1}, edit: in{
			role:  RoleRider,
			guest: 1,
			seats: 0,
			note:  "Life is cool!",
		}, want: nil, restamp: false, notify: false},
		{name: "negative guests", seeded: in{
			role:  RoleRider,
			guest: 0,
			seats: 0,
			note:  "",
		}, edit: in{
			role:  RoleRider,
			guest: -1,
			seats: 0,
			note:  "",
		}, want: &badRequest},
		{name: "driver with 0 seats", seeded: in{role: RoleRider}, edit: in{
			role:  RoleDriver,
			guest: 0,
			seats: 0,
			note:  "",
		}, want: &badRequest},
		{name: "rider with seats", seeded: in{role: RoleRider}, edit: in{
			role:  RoleRider,
			guest: 0,
			seats: 2,
			note:  "",
		}, want: &badRequest},
	}

	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			h1 := uuid.New()
			o := seedOuting(10, 2, StatusOpen, host, f)
			jr := seedJoinRequest(o.ID, h1, RequestStatusRequested, cc.seeded.role, f, cc.seeded.guest, withRequestedAt(now.Add(-5*time.Hour)), withSeats(cc.seeded.seats))
			before := jr.RequestedAt
			fn.events = nil
			_, err := svc.EditJoinRequest(context.Background(), h1, jr.ID, EditJoinRequestInput{
				Role:         ptr(cc.edit.role),
				Guests:       ptr(cc.edit.guest),
				SeatsOffered: ptr(cc.edit.seats),
				Note:         ptr(cc.edit.note),
			})
			if cc.want != nil {
				wantStatus(t, err, *cc.want)
			} else if err != nil {
				t.Fatalf("expected no error got %v", err)
			}
			if cc.notify {
				if len(fn.events) != 1 {
					t.Fatalf("expected len 1 got %v", len(fn.events))
				}
				if fn.events[0].Kind != notification.KindJoinRequestUpdated {
					t.Errorf("expected %v got %v", notification.KindJoinRequestUpdated, fn.events[0].Kind)
				}
			} else if len(fn.events) != 0 {
				t.Fatalf("expected len 0 got %v", len(fn.events))

			}
			if cc.restamp {
				if before.Equal(f.requests[jr.ID].RequestedAt) {
					t.Error("expected requested at restamped but got same")
				}
			} else {
				if !before.Equal(f.requests[jr.ID].RequestedAt) {
					t.Errorf("expected requested_at untouched (was %v) but got %v", before, f.requests[jr.ID].RequestedAt)
				}
			}
			got := ""
			if n := f.requests[jr.ID].Note; n != nil {
				got = *n
			}
			if got != cc.edit.note {
				t.Errorf("note: want %q got %q", cc.edit.note, got)
			}

		})
	}

}

func Test_EditRequest_Forbidden(t *testing.T) {
	svc, f, fn, _ := newTestService(t)
	host, hiker := uuid.New(), uuid.New()
	o := seedOuting(6, 2, StatusOpen, host, f)
	jr := seedJoinRequest(o.ID, hiker, RequestStatusRequested, RoleRider, f, 0)
	_, err := svc.EditJoinRequest(context.Background(), uuid.New(), jr.ID, EditJoinRequestInput{
		Role:         nil,
		Guests:       ptr(2),
		SeatsOffered: nil,
		Note:         nil,
	})
	wantStatus(t, err, apperr.CodeForbidden)
	if len(fn.events) != 0 {
		t.Fatalf("expected 0 got %d", len(fn.events))
	}
}

func Test_EditRequest_Conflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host, hiker := uuid.New(), uuid.New()
	now := time.Now()
	cases := []struct {
		name         string
		outingStarts time.Time
		reqStatus    RequestStatus
		outingStatus Status
		want         apperr.Status
	}{
		{name: "request status accepted", outingStarts: now.Add(24 * time.Hour), reqStatus: RequestStatusAccepted, outingStatus: StatusOpen, want: apperr.CodeConflict},
		{name: "outing started", outingStarts: now.Add(-1 * time.Hour), reqStatus: RequestStatusRequested, outingStatus: StatusOpen, want: apperr.CodeConflict},
		{name: "outing cancelled", outingStarts: now.Add(36 * time.Hour), reqStatus: RequestStatusRequested, outingStatus: StatusCancelled, want: apperr.CodeConflict},
	}
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			o := seedOuting(6, 4, cc.outingStatus, host, f, withStart(cc.outingStarts))
			jr := seedJoinRequest(o.ID, hiker, cc.reqStatus, RoleRider, f, 0)
			_, err := svc.EditJoinRequest(context.Background(), hiker, jr.ID, EditJoinRequestInput{
				Role:         nil,
				Guests:       ptr(2),
				SeatsOffered: nil,
				Note:         nil,
			})
			wantStatus(t, err, cc.want)
		})
	}

}
