package outing

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/google/uuid"
)

// --- helper methods --
func seedOuting(maxSize, hostSeats int, status Status, hostID uuid.UUID, f *fakeStore) *Outing {
	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour), MaxSize: maxSize, HostSeats: hostSeats, Status: status}
	f.outings[o.ID] = o
	return o
}

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}

func seedJoinRequest(outingID, hikerID uuid.UUID, status RequestStatus, role Role, f *fakeStore, guest int) *JoinRequest {
	r := &JoinRequest{ID: uuid.New(), OutingID: outingID, HikerID: hikerID, Status: status, Guests: guest, Role: role}
	f.requests[r.ID] = r
	return r
}

// --- RequestJoin ---

func TestRequestJoin_FirstRequestCreated(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

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

func TestRequestJoin_HostCannotSelfJoin(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hostID, o.ID, JoinInput{Role: RoleRider})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func TestRequestJoin_RiderWithSeats(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, SeatsOffered: 2})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func TestRequestJoin_DriverZeroSeats(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleDriver, SeatsOffered: 0})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func TestRequestJoin_TooManyGuests(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 4})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func TestRequestJoin_CancelledOuting(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusCancelled, hostID, f)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 2})

	wantStatus(t, err, apperr.CodeConflict)
}

func TestRequestJoin_DeclinedIsTerminal(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

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

func TestRequestJoin_DuplicateActive(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

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

func TestRequestJoin_WithdrawnMayRequest(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	r, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	f.requests[r.ID] = r
	f.requests[r.ID].Status = RequestStatusWithdrawn

	r2, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider})
	if err != nil {
		t.Fatalf("re-request after withdraw: %v", err)
	}
	if r2.Status != RequestStatusRequested {
		t.Errorf("returned status = %q, want %q", r2.Status, RequestStatusRequested)
	}
	if f.requests[r.ID].Status != RequestStatusRequested {
		t.Errorf("stored status = %q, want %q", f.requests[r.ID].Status, RequestStatusRequested)
	}
}

func TestAccept_Capacity(t *testing.T) {
	cases := []struct {
		name       string
		maxSize    int
		hostSeats  int
		seeded     []JoinRequest
		candidate  JoinRequest
		wantErr    bool
		wantStatus apperr.Status
	}{
		{
			name:      "driver accepted at full seats",
			maxSize:   10,
			hostSeats: 1,
			candidate: JoinRequest{Role: RoleDriver, SeatsOffered: 4, Status: RequestStatusRequested},
			wantErr:   false,
		},
		{
			name:       "rider rejected at full seats",
			maxSize:    10,
			hostSeats:  1,
			wantStatus: apperr.CodeConflict,
			wantErr:    true,
			candidate:  JoinRequest{Role: RoleRider, Status: RequestStatusRequested},
		},
		{
			name:       "rider rejected at full cap despite seats",
			maxSize:    2,
			hostSeats:  5,
			seeded:     []JoinRequest{{HikerID: uuid.New(), Role: RoleRider, Status: RequestStatusAccepted}},
			candidate:  JoinRequest{Role: RoleRider, Status: RequestStatusRequested},
			wantErr:    true,
			wantStatus: apperr.CodeConflict,
		},
		{
			name:       "driver rejected at full cap",
			maxSize:    2,
			hostSeats:  5,
			seeded:     []JoinRequest{{HikerID: uuid.New(), Role: RoleRider, Status: RequestStatusAccepted}},
			candidate:  JoinRequest{Role: RoleDriver, Status: RequestStatusRequested},
			wantErr:    true,
			wantStatus: apperr.CodeConflict,
		},
		{
			name:       "guests consume slots",
			maxSize:    3,
			hostSeats:  5,
			candidate:  JoinRequest{Role: RoleRider, Status: RequestStatusRequested, Guests: 2},
			wantErr:    true,
			wantStatus: apperr.CodeConflict,
		},
		{
			name:      "plain rider accepted",
			maxSize:   8,
			hostSeats: 4,
			candidate: JoinRequest{Role: RoleRider, Status: RequestStatusRequested, Guests: 1},
			wantErr:   false,
		},
		{
			name:      "capacity zero accepts driver",
			maxSize:   6,
			hostSeats: 0,
			candidate: JoinRequest{Role: RoleDriver, SeatsOffered: 4, Status: RequestStatusRequested},
			wantErr:   false,
		},
		{
			name:       "capacity zero rejects rider",
			maxSize:    6,
			hostSeats:  0,
			candidate:  JoinRequest{Role: RoleRider, Status: RequestStatusRequested},
			wantErr:    true,
			wantStatus: apperr.CodeConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			svc := NewService(f)
			hostID := uuid.New()

			o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour),
				MaxSize: tc.maxSize, HostSeats: tc.hostSeats, Status: StatusOpen}
			f.outings[o.ID] = o

			for i := range tc.seeded {
				r := tc.seeded[i]
				r.ID, r.OutingID, r.HikerID = uuid.New(), o.ID, uuid.New()
				f.requests[r.ID] = &r
			}

			cand := tc.candidate
			cand.ID, cand.OutingID, cand.HikerID = uuid.New(), o.ID, uuid.New()
			f.requests[cand.ID] = &cand

			err := svc.Accept(context.Background(), hostID, cand.ID)
			if tc.wantErr {
				wantStatus(t, err, tc.wantStatus)
				return
			}
			if err != nil {
				t.Fatalf("Accept: unexpected error: %v", err)
			}
			if f.requests[cand.ID].Status != RequestStatusAccepted {
				t.Errorf("stored status = %q, want accepted", f.requests[cand.ID].Status)
			}
		})
	}
}

func TestAccept_NonHostForbidden(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	random := uuid.New()
	hikerID := uuid.New()

	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	r := &JoinRequest{ID: uuid.New(), HikerID: hikerID, Guests: 1, Status: RequestStatusRequested, OutingID: o.ID}
	f.outings[o.ID] = o
	f.requests[r.ID] = r

	err := svc.Accept(context.Background(), random, r.ID)
	wantStatus(t, err, apperr.CodeForbidden)

}

func Test_Withdraw_AcceptedMayWithdraw(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	f.outings[o.ID] = o
	r := &JoinRequest{ID: uuid.New(), OutingID: o.ID, HikerID: hikerID, Status: RequestStatusAccepted}
	f.requests[r.ID] = r

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	if err != nil {
		t.Fatalf("Withdraw: unexpected error: %v", err)
	}
	if r.Status != RequestStatusWithdrawn {
		t.Errorf("stored status = %q, want %q", r.Status, RequestStatusWithdrawn)
	}
}

func Test_Withdraw_RequestedOK(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	f.outings[o.ID] = o
	r := &JoinRequest{ID: uuid.New(), OutingID: o.ID, HikerID: hikerID, Status: RequestStatusRequested}
	f.requests[r.ID] = r

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	if err != nil {
		t.Fatalf("Withdraw: unexpected error: %v", err)
	}
	if r.Status != RequestStatusWithdrawn {
		t.Errorf("stored status = %q, want %q", r.Status, RequestStatusWithdrawn)
	}
}
func Test_Withdraw_DeclinedConflicts(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(48 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	f.outings[o.ID] = o
	r := &JoinRequest{ID: uuid.New(), OutingID: o.ID, HikerID: hikerID, Status: RequestStatusDeclined}
	f.requests[r.ID] = r

	err := svc.Withdraw(context.Background(), hikerID, o.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Decline_PendingWorks(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 2)

	err := svc.Decline(context.Background(), hostID, r.ID)
	if err != nil {
		t.Fatalf("found an error %v", err)
	}
	if r.Status != RequestStatusDeclined {
		t.Errorf("expected %v got %v", RequestStatusDeclined, r.Status)
	}
}

func Test_Decline_AcceptedConflicts(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 2)

	err := svc.Decline(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Accept_AlreadyAcceptedConflicts(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	err := svc.Accept(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_RemoveMember_AcceptedWorks(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	err := svc.RemoveMember(context.Background(), hostID, r.ID)
	if err != nil {
		t.Errorf("expected to remove but got error: %v", err)
	}
	if r.Status != RequestStatusWithdrawn {
		t.Errorf("expected %v got %v", RequestStatusWithdrawn, r.Status)
	}

}

func Test_RemoveMember_PendingConflicts(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	err := svc.RemoveMember(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)

}

func Test_Cancel_HostCancelsOpen(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()
	o := seedOuting(6, 2, StatusOpen, hostID, f)

	if err := svc.Cancel(context.Background(), hostID, o.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if f.outings[o.ID].Status != StatusCancelled {
		t.Errorf("stored status = %q, want cancelled", f.outings[o.ID].Status)
	}
}

func Test_Cancel_HostCancelsCancelled(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()
	o := seedOuting(6, 2, StatusCancelled, hostID, f)

	err := svc.Cancel(context.Background(), hostID, o.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Cancel_HostCancelsPastEvent(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()
	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: time.Now().Add(-2 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	f.outings[o.ID] = o

	err := svc.Cancel(context.Background(), hostID, o.ID)
	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_Cancel_RandomCancelsEvent(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()
	randomID := uuid.New()
	o := seedOuting(6, 2, StatusOpen, hostID, f)

	err := svc.Cancel(context.Background(), randomID, o.ID)
	wantStatus(t, err, apperr.CodeForbidden)
}
