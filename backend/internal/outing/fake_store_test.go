package outing

import (
	"context"
	"sort"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	outings  map[uuid.UUID]*Outing
	requests map[uuid.UUID]*JoinRequest
	hikers   map[uuid.UUID]*Member
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		outings:  map[uuid.UUID]*Outing{},
		requests: map[uuid.UUID]*JoinRequest{},
		hikers:   map[uuid.UUID]*Member{},
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

func (f *fakeStore) SetOutingStatus(ctx context.Context, outingID uuid.UUID, status Status) error {
	o, ok := f.outings[outingID]
	if !ok {
		return apperr.NotFound("outing not found", "outing not found")
	}
	o.Status = status
	return nil
}

func (f *fakeStore) ListUpcoming(ctx context.Context, now time.Time) ([]Outing, error) {
	upcoming := []Outing{}
	for _, o := range f.outings {
		if o.Status == StatusOpen && o.StartsAt.After(now) {
			upcoming = append(upcoming, *o)
		}
	}
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].StartsAt.Before(upcoming[j].StartsAt)
	})
	return upcoming, nil
}

func (f *fakeStore) ListJoinRequests(ctx context.Context, outingID uuid.UUID, status RequestStatus) ([]JoinRequest, error) {
	jrs := []JoinRequest{}

	for _, r := range f.requests {
		if r.OutingID == outingID && r.Status == status {
			jrs = append(jrs, *r)
		}
	}
	sort.Slice(jrs, func(i, j int) bool {
		return jrs[i].CreatedAt.Before(jrs[j].CreatedAt)
	})
	return jrs, nil
}

func (f *fakeStore) ListForHiker(ctx context.Context, hikerID uuid.UUID) (*MyOutings, error) {
	myOutings := &MyOutings{
		Hosting: []Outing{},
		Joined:  []Outing{},
	}
	for _, o := range f.outings {
		if o.HostID == hikerID {
			myOutings.Hosting = append(myOutings.Hosting, *o)
		}
	}
	for _, r := range f.requests {
		if r.HikerID == hikerID && r.Status == RequestStatusAccepted {
			o, err := f.GetOuting(ctx, r.OutingID)
			if err != nil {
				return nil, err
			}
			myOutings.Joined = append(myOutings.Joined, *o)
		}
	}
	sort.Slice(myOutings.Hosting, func(i, j int) bool {
		return myOutings.Hosting[i].StartsAt.Before(myOutings.Hosting[j].StartsAt)
	})
	sort.Slice(myOutings.Joined, func(i, j int) bool {
		return myOutings.Joined[i].StartsAt.Before(myOutings.Joined[j].StartsAt)
	})

	return myOutings, nil
}

func (f *fakeStore) Roster(ctx context.Context, outingID uuid.UUID) ([]Member, error) {
	members := []Member{}
	for _, r := range f.requests {
		if r.OutingID == outingID && r.Status == RequestStatusAccepted {
			if m, ok := f.hikers[r.HikerID]; ok {
				members = append(members, *m)
			}
		}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})
	return members, nil
}

func (f *fakeStore) HostMember(ctx context.Context, hikerID uuid.UUID) (*Member, error) {
	m, ok := f.hikers[hikerID]
	if !ok {
		return nil, apperr.NotFound("hiker not found", "fake: no member")
	}
	return m, nil
}

func (f *fakeStore) UpdateOuting(ctx context.Context, o *Outing) error {
	if _, ok := f.outings[o.ID]; !ok {
		return apperr.NotFound("outing not found", "fake: no outing to update")
	}
	f.outings[o.ID] = o
	return nil
}
