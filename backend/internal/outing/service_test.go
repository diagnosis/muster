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
func seedOutingWithStartTime(maxSize, hostSeats int, status Status, hostID uuid.UUID, f *fakeStore, startTime time.Time) *Outing {
	o := &Outing{ID: uuid.New(), HostID: hostID, StartsAt: startTime, MaxSize: maxSize, HostSeats: hostSeats, Status: status}
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
func seedJoinRequestWithCreatedAt(outingID, hikerID uuid.UUID, status RequestStatus, role Role, f *fakeStore, guest int, createdAt time.Time) *JoinRequest {
	r := &JoinRequest{ID: uuid.New(), OutingID: outingID, HikerID: hikerID, Status: status, Guests: guest, Role: role, CreatedAt: createdAt}
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

func Test_ListUpcoming_SortedOpenFuture(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	host := uuid.New()

	late := seedOuting(6, 2, StatusOpen, host, f)

	far := &Outing{ID: uuid.New(), HostID: host, StartsAt: time.Now().Add(64 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	soon := &Outing{ID: uuid.New(), HostID: host, StartsAt: time.Now().Add(12 * time.Hour), MaxSize: 6, HostSeats: 2, Status: StatusOpen}
	f.outings[far.ID], f.outings[soon.ID] = far, soon

	got, err := svc.ListUpcoming(context.Background())
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 outings, got %d", len(got))
	}

	wantOrder := []uuid.UUID{soon.ID, late.ID, far.ID}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, got[i].ID, id)
		}
	}

}

func Test_PendingRequests_QueueOrder(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	r1 := seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, time.Now())
	r2 := seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, time.Now().Add(-1*time.Hour))
	r3 := seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, time.Now().Add(1*time.Hour))

	pendingRequests, err := svc.PendingRequests(context.Background(), hostID, o.ID)
	if err != nil {
		t.Fatalf("PendingRequests: %v", err)
	}
	if len(pendingRequests) != 3 {
		t.Errorf("expected 3 outings, got %d", len(pendingRequests))
	}
	wantOrder := []uuid.UUID{r2.ID, r1.ID, r3.ID}
	for i, r := range wantOrder {
		if pendingRequests[i].ID != r {
			t.Errorf("position %d: got %s, want %s", i, pendingRequests[i].ID, r)
		}
	}

}

func Test_PendingRequests_Leaked(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusAccepted, RoleDriver, f, 1, time.Now())
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusDeclined, RoleDriver, f, 1, time.Now().Add(-1*time.Hour))
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, time.Now().Add(1*time.Hour))

	pendingRequests, err := svc.PendingRequests(context.Background(), hostID, o.ID)
	if err != nil {
		t.Fatalf("ListUpcomingRequest: %v", err)
	}
	if len(pendingRequests) != 1 {
		t.Errorf("expected 1 outings, got %d", len(pendingRequests))
	}
	if pendingRequests[0].Status != RequestStatusRequested {
		t.Errorf("expected StatusRequested got%v", pendingRequests[0].Status)
	}

}

func Test_PendingRequests_Forbidden(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, time.Now())
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, time.Now().Add(-1*time.Hour))
	_ = seedJoinRequestWithCreatedAt(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, time.Now().Add(1*time.Hour))
	unknownHost := uuid.New()
	_, err := svc.PendingRequests(context.Background(), unknownHost, o.ID)
	wantStatus(t, err, apperr.CodeForbidden)

}
func Test_PendingRequests_EmptyNotNil(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hostID := uuid.New()
	o := seedOuting(7, 2, StatusOpen, hostID, f)

	got, err := svc.PendingRequests(context.Background(), hostID, o.ID)
	if err != nil {
		t.Fatalf("PendingRequests: %v", err)
	}
	if got == nil {
		t.Error("expected empty slice, got nil — list contract is [] never null")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 requests, got %d", len(got))
	}
}

func Test_MyOutings_Buckets(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hikerID := uuid.New()
	_ = seedOuting(6, 2, StatusOpen, hikerID, f)
	_ = seedOuting(4, 2, StatusCancelled, hikerID, f)
	approvedJoin := seedOuting(6, 3, StatusOpen, uuid.New(), f)
	_ = seedJoinRequest(approvedJoin.ID, hikerID, RequestStatusAccepted, RoleRider, f, 0)
	pendingJoin := seedOuting(5, 3, StatusOpen, uuid.New(), f)
	_ = seedJoinRequest(pendingJoin.ID, hikerID, RequestStatusRequested, RoleDriver, f, 0)
	declinedJoin := seedOuting(5, 3, StatusOpen, uuid.New(), f)
	_ = seedJoinRequest(declinedJoin.ID, hikerID, RequestStatusDeclined, RoleRider, f, 1)
	withDrawnJoin := seedOuting(7, 4, StatusOpen, uuid.New(), f)
	_ = seedJoinRequest(withDrawnJoin.ID, hikerID, RequestStatusWithdrawn, RoleRider, f, 1)

	myOutings, err := svc.MyOutings(context.Background(), hikerID)
	if err != nil {
		t.Fatalf("MyOutings: %v", err)
	}
	if len(myOutings.Hosting) != 2 {
		t.Errorf("hosting: len = %d, want 2", len(myOutings.Hosting))
	}
	if len(myOutings.Joined) != 1 {
		t.Errorf("joined: len = %d, want 1", len(myOutings.Joined))
	}
	if len(myOutings.Joined) == 1 && myOutings.Joined[0].ID != approvedJoin.ID {
		t.Errorf("joined[0] = %s, want %s", myOutings.Joined[0].ID, approvedJoin.ID)
	}
}
func Test_MyOutings_Sorted(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hikerID := uuid.New()
	earlySelf := seedOutingWithStartTime(6, 4, StatusOpen, hikerID, f, time.Now().Add(24*time.Hour))
	lateSelf := seedOutingWithStartTime(6, 4, StatusOpen, hikerID, f, time.Now().Add(7*24*time.Hour))

	earlyOther := seedOutingWithStartTime(6, 4, StatusOpen, uuid.New(), f, time.Now().Add(2*24*time.Hour))
	_ = seedJoinRequest(earlyOther.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)
	lateOther := seedOutingWithStartTime(6, 4, StatusOpen, uuid.New(), f, time.Now().Add(10*24*time.Hour))
	_ = seedJoinRequest(lateOther.ID, hikerID, RequestStatusAccepted, RoleDriver, f, 1)

	myOutings, err := svc.MyOutings(context.Background(), hikerID)
	if err != nil {
		t.Fatalf("MyOutings: %v", err)
	}
	wantedOrderSelf := []uuid.UUID{earlySelf.ID, lateSelf.ID}
	for i, id := range wantedOrderSelf {
		if myOutings.Hosting[i].ID != id {
			t.Errorf("expected %s got %s", id, myOutings.Hosting[i].ID)
		}
	}
	wantedOrderOther := []uuid.UUID{earlyOther.ID, lateOther.ID}
	for i, id := range wantedOrderOther {
		if myOutings.Joined[i].ID != id {
			t.Errorf("expected %s got %s", id, myOutings.Joined[i].ID)
		}
	}

}

func Test_MyOutings_EmptyNotNil(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hikerID := uuid.New()

	myOutings, err := svc.MyOutings(context.Background(), hikerID)
	if err != nil {
		t.Fatalf("MyOutings: %v", err)
	}

	gotHosting := myOutings.Hosting
	gotJoined := myOutings.Joined
	if gotHosting == nil {
		t.Error("expected empty slice, got nil — list contract is [] never null")
	}
	if len(gotHosting) != 0 {
		t.Errorf("expected 0 requests, got %d", len(gotHosting))
	}
	if gotJoined == nil {
		t.Error("expected empty slice, got nil — list contract is [] never null")
	}
	if len(gotJoined) != 0 {
		t.Errorf("expected 0 requests, got %d", len(gotJoined))
	}

}
