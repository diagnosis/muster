package outing

import (
	"context"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/google/uuid"
)

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

var _ Storage = (*fakeStore)(nil)

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

// SeatsOffered/HostSeats = total seats incl. the driver; headcount incl. host must fit both max_size and seat capacity (riders only).
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
