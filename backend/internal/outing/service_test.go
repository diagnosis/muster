package outing

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/google/uuid"
)

func TestRequestJoin_FirstRequestCreated(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

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
	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

	_, err := svc.RequestJoin(context.Background(), hostID, o.ID, JoinInput{Role: RoleRider})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func TestRequestJoin_RiderWithSeats(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}

	f.outings[o.ID] = o

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, SeatsOffered: 2})

	wantStatus(t, err, apperr.CodeBadRequest)

}

func TestRequestJoin_DriverZeroSeats(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()
	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o
	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleDriver, SeatsOffered: 0})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func TestRequestJoin_TooManyGuests(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 4})

	wantStatus(t, err, apperr.CodeBadRequest)
}

func TestRequestJoin_CancelledOuting(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusCancelled,
	}
	f.outings[o.ID] = o

	_, err := svc.RequestJoin(context.Background(), hikerID, o.ID, JoinInput{Role: RoleRider, Guests: 2})

	wantStatus(t, err, apperr.CodeConflict)
}

func TestRequestJoin_DeclinedIsTerminal(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)

	hostID := uuid.New()
	hikerID := uuid.New()

	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

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

	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

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

	o := &Outing{
		ID:        uuid.New(),
		HostID:    hostID,
		StartsAt:  time.Now().Add(48 * time.Hour),
		MaxSize:   6,
		HostSeats: 4,
		Status:    StatusOpen,
	}
	f.outings[o.ID] = o

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

type fakeStore struct {
	outings  map[uuid.UUID]*Outing
	requests map[uuid.UUID]*JoinRequest
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		outings:  map[uuid.UUID]*Outing{},
		requests: map[uuid.UUID]*JoinRequest{},
	}
}

func (f *fakeStore) GetOuting(ctx context.Context, id uuid.UUID) (*Outing, error) {
	o, ok := f.outings[id]
	if !ok {
		return nil, apperr.NotFound("outing not found", "fake: no outing")
	}
	return o, nil
}
func (f *fakeStore) CreateOuting(ctx context.Context, o *Outing) error {
	f.outings[o.ID] = o
	return nil
}
func (f *fakeStore) CreateJoinRequest(ctx context.Context, r *JoinRequest) error {
	f.requests[r.ID] = r
	return nil
}

func (f *fakeStore) GetJoinRequest(ctx context.Context, id uuid.UUID) (*JoinRequest, error) {
	r, ok := f.requests[id]
	if !ok {
		return nil, apperr.NotFound("request not found", "fake: no request")
	}
	return r, nil
}

func (f *fakeStore) GetJoinRequestByHiker(ctx context.Context, outingID, hikerID uuid.UUID) (*JoinRequest, error) {
	for _, r := range f.requests {
		if r.OutingID == outingID && r.HikerID == hikerID {
			return r, nil
		}
	}
	return nil, apperr.NotFound("request not found", "fake: no request for hiker")

}

func (f *fakeStore) SetJoinRequestStatus(ctx context.Context, id uuid.UUID, s RequestStatus) error {
	r, ok := f.requests[id]
	if !ok {
		return apperr.NotFound("request not found", "fake: no request")
	}
	r.Status = s
	return nil
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

func (f *fakeStore) AcceptIfCapacity(ctx context.Context, requestID uuid.UUID) error {
	r, ok := f.requests[requestID]
	if !ok {
		return apperr.NotFound("request not found", "request not found")
	}
	o, ok := f.outings[r.OutingID]
	if !ok {
		return apperr.NotFound("outing not found", "outing not found")
	}
	people, seatCap := 0, 0
	for _, req := range f.requests {
		if req.OutingID == r.OutingID && req.Status == RequestStatusAccepted {
			people += 1 + req.Guests
			if req.Role == RoleDriver {
				seatCap += req.SeatsOffered
			}
		}
	}
	people += 1            // the host
	seatCap += o.HostSeats // the host's car

	need := people + 1 + r.Guests
	if need > o.MaxSize {
		return apperr.Conflict("outing is full", "cap exceeded")
	}
	if r.Role == RoleRider && need > seatCap {
		return apperr.Conflict("not enough seats", "seat capacity exceeded")
	}
	r.Status = RequestStatusAccepted
	return nil
}

var _ Storage = (*fakeStore)(nil)

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}
