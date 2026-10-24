// backend/internal/outing/host_actions_test.go
// Tests for what the host does to requests and the outing: accept, decline, remove, restore, cancel, and the request lists.
package outing

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

func Test_Accept_Capacity(t *testing.T) {
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
			svc, f, _, _ := newTestService(t)
			hostID := uuid.New()

			o := seedOuting(tc.maxSize, tc.hostSeats, StatusOpen, hostID, f)

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

func Test_Accept_NonHostForbidden(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	random := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 2, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	err := svc.Accept(context.Background(), random, r.ID)
	wantStatus(t, err, apperr.CodeForbidden)

}

func Test_Decline_PendingWorks(t *testing.T) {
	svc, f, _, _ := newTestService(t)

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
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 2)

	err := svc.Decline(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Accept_AlreadyAcceptedConflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	err := svc.Accept(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_RemoveMember_AcceptedWorks(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)

	err := svc.RemoveMember(context.Background(), hostID, r.ID)
	if err != nil {
		t.Errorf("expected to remove but got error: %v", err)
	}
	if r.Status != RequestStatusRemoved {
		t.Errorf("expected %v got %v", RequestStatusRemoved, r.Status)
	}

}

func Test_RemoveMember_PendingConflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)
	r := seedJoinRequest(o.ID, hikerID, RequestStatusRequested, RoleRider, f, 1)

	err := svc.RemoveMember(context.Background(), hostID, r.ID)
	wantStatus(t, err, apperr.CodeConflict)

}

func Test_RemoveMember_CannotRejoin(t *testing.T) {
	svc, f, _, _ := newTestService(t)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, hikerID, RequestStatusRemoved, RoleRider, f, 1)

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{
		Role:         "rider",
		SeatsOffered: 0,
		Guests:       0,
		Note:         nil,
	})
	wantStatus(t, err, apperr.CodeConflict)
	r, err := f.GetJoinRequestByHiker(context.Background(), o.ID, hikerID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if r.Status != RequestStatusRemoved {
		t.Fatalf("failed re-request must not mutate the row: expected %v got %v", RequestStatusRemoved, r.Status)
	}

}

func Test_Cancel_HostCancelsOpen(t *testing.T) {
	svc, f, _, _ := newTestService(t)
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
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(6, 2, StatusCancelled, hostID, f)

	err := svc.Cancel(context.Background(), hostID, o.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Cancel_HostCancelsPastEvent(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(6, 2, StatusOpen, hostID, f, withStart(time.Now().Add(-2*time.Hour)))

	err := svc.Cancel(context.Background(), hostID, o.ID)
	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_Cancel_RandomCancelsEvent(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	randomID := uuid.New()
	o := seedOuting(6, 2, StatusOpen, hostID, f)

	err := svc.Cancel(context.Background(), randomID, o.ID)
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_JoinRequests_QueueOrder(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	r1 := seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, withRequestedAt(time.Now()))
	r2 := seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, withRequestedAt(time.Now().Add(-1*time.Hour)))
	r3 := seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, withRequestedAt(time.Now().Add(1*time.Hour)))

	pendingRequests, err := svc.JoinRequests(context.Background(), hostID, o.ID, RequestStatusRequested)
	if err != nil {
		t.Fatalf("PendingRequests: %v", err)
	}
	if len(pendingRequests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(pendingRequests))
	}
	wantOrder := []uuid.UUID{r2.ID, r1.ID, r3.ID}
	for i, r := range wantOrder {
		if pendingRequests[i].ID != r {
			t.Errorf("position %d: got %s, want %s", i, pendingRequests[i].ID, r)
		}
	}

}

func Test_JoinRequests_OnlyTheAskedStatus(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleDriver, f, 1, withRequestedAt(time.Now()))
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusDeclined, RoleDriver, f, 1, withRequestedAt(time.Now().Add(-1*time.Hour)))
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, withRequestedAt(time.Now().Add(1*time.Hour)))

	pendingRequests, err := svc.JoinRequests(context.Background(), hostID, o.ID, RequestStatusRequested)
	if err != nil {
		t.Fatalf("ListUpcomingRequest: %v", err)
	}
	if len(pendingRequests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(pendingRequests))
	}
	if pendingRequests[0].Status != RequestStatusRequested {
		t.Errorf("expected %v got %v", RequestStatusRequested, pendingRequests[0].Status)
	}

}

func Test_JoinRequests_Forbidden(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()

	o := seedOuting(7, 2, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, withRequestedAt(time.Now()))
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleDriver, f, 1, withRequestedAt(time.Now().Add(-1*time.Hour)))
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusRequested, RoleRider, f, 2, withRequestedAt(time.Now().Add(1*time.Hour)))
	unknownHost := uuid.New()
	_, err := svc.JoinRequests(context.Background(), unknownHost, o.ID, RequestStatusRequested)
	wantStatus(t, err, apperr.CodeForbidden)

}

func Test_JoinRequests_EmptyNotNil(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(7, 2, StatusOpen, hostID, f)

	got, err := svc.JoinRequests(context.Background(), hostID, o.ID, RequestStatusRequested)
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

func Test_HostActions_Rejections_Conflict(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	member := uuid.New()
	phases := phaseCases(time.Now())
	actions := []struct {
		name   string
		status RequestStatus
		do     func(ctx context.Context, host, req uuid.UUID) error
	}{
		{name: "accept", status: RequestStatusRequested, do: svc.Accept},
		{name: "decline", status: RequestStatusRequested, do: svc.Decline},
		{name: "remove", status: RequestStatusAccepted, do: svc.RemoveMember},
	}

	for _, pp := range phases {
		for _, aa := range actions {
			t.Run(pp.name+" - "+aa.name, func(t *testing.T) {
				outing := seedOuting(6, 4, StatusOpen, host, f, withStart(pp.startAt), withEnd(pp.endsAt))
				jr := seedJoinRequest(outing.ID, member, aa.status, "rider", f, 0)
				err := aa.do(context.Background(), host, jr.ID)
				if pp.wantErr {
					wantStatus(t, err, apperr.CodeConflict)
				} else if err != nil {
					t.Errorf("expected no error got %v", err)
				}

			})
		}
	}

}

func Test_RestoreRequest(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1 := uuid.New()

	cases := []struct {
		name              string
		joinRequestStatus RequestStatus
		wantConflict      bool
	}{
		{"restored from removed", RequestStatusRemoved, false},
		{"restored from declined", RequestStatusDeclined, false},
		{"attempt to restore from accepted", RequestStatusAccepted, true},
		{"attempt to restore from requested", RequestStatusRequested, true},
		{"attempt to restore from withdrawn", RequestStatusWithdrawn, true},
	}

	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			o := seedOuting(6, 4, StatusOpen, host, f)
			j := seedJoinRequest(o.ID, m1, cc.joinRequestStatus, RoleRider, f, 0)
			j.RequestedAt = time.Now().Add(-5 * time.Hour)
			preRequestedAt := j.RequestedAt
			err := svc.RestoreRequest(context.Background(), host, j.ID)
			switch {
			case cc.wantConflict:
				wantStatus(t, err, apperr.CodeConflict)
			case err != nil:
				t.Fatalf("expected no error got %v", err)
			case f.requests[j.ID].Status != RequestStatusRequested:
				t.Errorf("expected status %s got %s", RequestStatusRequested, f.requests[j.ID].Status)
			default:
				if !f.requests[j.ID].RequestedAt.Equal(preRequestedAt) {
					t.Errorf("expected requested at: %v got %v ", preRequestedAt, f.requests[j.ID].RequestedAt)
				}
			}
		})
	}

}

func Test_Restore_Auth(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	fakeHost := uuid.New()
	m1 := uuid.New()

	o := seedOuting(6, 4, StatusOpen, host, f)
	j := seedJoinRequest(o.ID, m1, RequestStatusDeclined, RoleRider, f, 0)
	err := svc.RestoreRequest(context.Background(), fakeHost, j.ID)
	wantStatus(t, err, apperr.CodeForbidden)
	if f.requests[j.ID].Status != RequestStatusDeclined {
		t.Errorf("expected to be declined got %v", f.requests[j.ID].Status)
	}
	err = svc.RestoreRequest(context.Background(), host, j.ID)
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if f.requests[j.ID].Status != RequestStatusRequested {
		t.Errorf("expected to be Requested got %v", f.requests[j.ID].Status)
	}
}

func Test_Restore_OutingAlreadyStarted(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1 := uuid.New()

	o := seedOuting(6, 4, StatusOpen, host, f, withStart(time.Now().Add(-time.Hour)))
	j := seedJoinRequest(o.ID, m1, RequestStatusDeclined, RoleRider, f, 0)
	err := svc.RestoreRequest(context.Background(), host, j.ID)
	wantStatus(t, err, apperr.CodeConflict)
	if f.requests[j.ID].Status != RequestStatusDeclined {
		t.Errorf("expected to be declined got %v", f.requests[j.ID].Status)
	}
}

func Test_Restore_OutingCancelled(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1 := uuid.New()
	o := seedOuting(6, 4, StatusCancelled, host, f, withStart(time.Now().Add(10*time.Hour)))
	j := seedJoinRequest(o.ID, m1, RequestStatusDeclined, RoleRider, f, 0)
	err := svc.RestoreRequest(context.Background(), host, j.ID)
	wantStatus(t, err, apperr.CodeConflict)
	if f.requests[j.ID].Status != RequestStatusDeclined {
		t.Errorf("expected to be declined got %v", f.requests[j.ID].Status)
	}
}

func Test_JoinRequests_DeclinedAndRemoved(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1, m2, m3, m4 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	o := seedOuting(10, 4, StatusOpen, host, f)
	j1 := seedJoinRequest(o.ID, m1, RequestStatusDeclined, RoleDriver, f, 0, withRequestedAt(time.Now().Add(-10*time.Hour)))
	j2 := seedJoinRequest(o.ID, m2, RequestStatusDeclined, RoleDriver, f, 0, withRequestedAt(time.Now().Add(-9*time.Hour)))
	j3 := seedJoinRequest(o.ID, m3, RequestStatusRemoved, RoleDriver, f, 0, withRequestedAt(time.Now().Add(-7*time.Hour)))
	j4 := seedJoinRequest(o.ID, m4, RequestStatusRemoved, RoleDriver, f, 0, withRequestedAt(time.Now().Add(-6*time.Hour)))

	jrs, err := svc.JoinRequests(context.Background(), host, o.ID, RequestStatusDeclined)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(jrs) != 2 {
		t.Fatalf("expected len 2 got %v", len(jrs))
	}
	if jrs[0].ID != j1.ID || jrs[1].ID != j2.ID {
		t.Errorf("declined order: want [%v %v] got [%v %v]", j1.ID, j2.ID, jrs[0].ID, jrs[1].ID)
	}

	jrs, err = svc.JoinRequests(context.Background(), host, o.ID, RequestStatusRemoved)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(jrs) != 2 {
		t.Fatalf("expected len 2 got %v", len(jrs))
	}
	if jrs[0].ID != j3.ID || jrs[1].ID != j4.ID {
		t.Errorf("removed order: want [%v %v] got [%v %v]", j3.ID, j4.ID, jrs[0].ID, jrs[1].ID)
	}

	jrs, err = svc.JoinRequests(context.Background(), host, o.ID, "banana")

	wantStatus(t, err, apperr.CodeBadRequest)

	if len(jrs) != 0 {
		t.Fatalf("expected len 0 got %v", len(jrs))
	}

}

func Test_JoinRequests_DriversFirst(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	h1, d1, h2, d2 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	o := seedOuting(6, 2, StatusOpen, host, f)
	now := time.Now()
	j1 := seedJoinRequest(o.ID, h1, RequestStatusRequested, RoleRider, f, 1, withRequestedAt(now.Add(-10*time.Hour)))
	j2 := seedJoinRequest(o.ID, d1, RequestStatusRequested, RoleDriver, f, 0, withRequestedAt(now.Add(-5*time.Hour)), withSeats(3))
	j3 := seedJoinRequest(o.ID, h2, RequestStatusRequested, RoleRider, f, 1, withRequestedAt(now.Add(-2*time.Hour)))
	j4 := seedJoinRequest(o.ID, d2, RequestStatusRequested, RoleDriver, f, 0, withRequestedAt(now.Add(-1*time.Hour)), withSeats(2))
	reqs, err := svc.JoinRequests(context.Background(), host, o.ID, RequestStatusRequested)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(reqs) != 4 {
		t.Fatalf("expected requested size:4 got %d", len(reqs))
	}
	expectedOrder := []uuid.UUID{j2.ID, j4.ID, j1.ID, j3.ID}
	for i, id := range expectedOrder {
		if reqs[i].ID != id {
			t.Errorf("expected %v got %v", id, reqs[i].ID)
		}
	}

}
