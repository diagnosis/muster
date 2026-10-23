// backend/internal/outing/helpers_test.go
package outing

import (
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

// newTestService wires a Service to in-memory fakes and returns them so a
// test can seed state and inspect what the service wrote or sent.
func newTestService(t *testing.T) (*Service, *fakeStore, *fakeNotificationStore, *fakeBroadcaster) {
	t.Helper()
	f := newFakeStore()
	fn := &fakeNotificationStore{}
	fb := newFakeBroadcaster()
	return NewService(f, fn, fb), f, fn, fb
}

// --- seeding ---
//
// Two seed helpers cover every fixture. The positional arguments are the
// fields almost every test sets; anything rarer is an option, so a call site
// names exactly what makes its fixture special:
//
//	o := seedOuting(6, 4, StatusOpen, host, f, withStart(t0), withEnd(&t1))
//	jr := seedJoinRequest(o.ID, h, RequestStatusRequested, RoleDriver, f, 0, withSeats(3))
//
// Both return the stored pointer, so a test may also change a field after
// seeding (jr.UpdatedAt = ...) and the fake sees it.

type outingOption func(*Outing)

func withStart(t time.Time) outingOption { return func(o *Outing) { o.StartsAt = t } }
func withEnd(t *time.Time) outingOption  { return func(o *Outing) { o.EndsAt = t } }

// seedOuting stores an outing starting in 48 hours, then applies opts.
func seedOuting(maxSize, hostSeats int, status Status, hostID uuid.UUID, f *fakeStore, opts ...outingOption) *Outing {
	o := &Outing{
		ID: uuid.New(), HostID: hostID,
		Title: "seeded outing", Destination: "seeded dest", MeetLabel: "seeded meet",
		StartsAt: time.Now().Add(48 * time.Hour),
		MaxSize:  maxSize, HostSeats: hostSeats,
		Difficulty: DifficultyModerate, Pace: PaceRelaxed,
		Status: status,
	}
	for _, opt := range opts {
		opt(o)
	}
	f.outings[o.ID] = o
	return o
}

type requestOption func(*JoinRequest)

func withSeats(n int) requestOption             { return func(r *JoinRequest) { r.SeatsOffered = n } }
func withRequestedAt(t time.Time) requestOption { return func(r *JoinRequest) { r.RequestedAt = t } }

// seedJoinRequest stores a request with the given guests, then applies opts.
func seedJoinRequest(outingID, hikerID uuid.UUID, status RequestStatus, role Role, f *fakeStore, guests int, opts ...requestOption) *JoinRequest {
	r := &JoinRequest{ID: uuid.New(), OutingID: outingID, HikerID: hikerID, Status: status, Role: role, Guests: guests}
	for _, opt := range opts {
		opt(r)
	}
	f.requests[r.ID] = r
	return r
}

// seedMember stores a hiker so name lookups (host card, roster, lists) find them.
func seedMember(hikerID uuid.UUID, name, experience string, f *fakeStore) *Member {
	m := &Member{HikerID: &hikerID, Name: name, Experience: experience}
	f.hikers[hikerID] = m
	return m
}

// --- assertions ---

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}

func ptr[T any](v T) *T {
	return &v
}

// --- phase fixtures ---

// phaseCase is an outing timing relative to now and whether host or
// hiker actions on it must be refused (only upcoming outings accept them).
type phaseCase struct {
	name    string
	startAt time.Time
	endsAt  *time.Time
	wantErr bool
}

// phaseCases is the shared timing table for every "refused once the outing
// has started" rule: upcoming passes, in progress and past are refused,
// each with and without an explicit end.
func phaseCases(now time.Time) []phaseCase {
	upEnd := now.Add(56 * time.Hour)
	inEnd := now.Add(4 * time.Hour)
	pastEnd := now.Add(-38 * time.Hour)
	return []phaseCase{
		{name: "upcoming", startAt: now.Add(48 * time.Hour), wantErr: false},
		{name: "upcoming with end", startAt: now.Add(48 * time.Hour), endsAt: &upEnd, wantErr: false},
		{name: "in progress no end", startAt: now.Add(-6 * time.Hour), wantErr: true},
		{name: "in progress with end", startAt: now.Add(-6 * time.Hour), endsAt: &inEnd, wantErr: true},
		{name: "past no end", startAt: now.Add(-48 * time.Hour), wantErr: true},
		{name: "past with end", startAt: now.Add(-48 * time.Hour), endsAt: &pastEnd, wantErr: true},
	}
}
