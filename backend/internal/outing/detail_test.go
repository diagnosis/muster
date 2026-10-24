// backend/internal/outing/detail_test.go
// Tests for the outing page payload: who sees what, capacity, and the waiting and withdrawn lists.
package outing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test_Detail_FullAssembly(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	driverHiker := uuid.New()
	riderHiker := uuid.New()

	o := seedOuting(8, 2, StatusOpen, hostID, f)
	_ = seedMember(hostID, "Safa", "experienced", f)

	_ = seedJoinRequest(o.ID, driverHiker, RequestStatusAccepted, RoleDriver, f, 1, withSeats(4))
	m1 := seedMember(driverHiker, "mahmut", "beginner", f)

	_ = seedJoinRequest(o.ID, riderHiker, RequestStatusAccepted, RoleRider, f, 0)
	m2 := seedMember(riderHiker, "celil", "intermediate", f)

	detail, err := svc.Detail(context.Background(), o.ID, &hostID)
	if err != nil {
		t.Fatalf("Detail Full: %v", err)
	}
	if len(detail.Roster) != 2 {
		t.Fatalf("expected 2 got %d", len(detail.Roster))
	}
	// host + driver with 1 guest + rider = 4; seats 2 host + 4 driver = 6; spots min(6, 8) - 4 = 2
	want := Capacity{PeopleCount: 4, SeatCapacity: 6, SeatsShort: 0, SpotsLeft: 2, WaitingCount: 0}
	if detail.Capacity != want {
		t.Errorf("capacity: got %+v, want %+v", detail.Capacity, want)
	}

	if detail.Host.Name != "Safa" {
		t.Errorf("expected Safa got %s", detail.Host.Name)
	}

	wantedRosterOrder := []uuid.UUID{*m2.HikerID, *m1.HikerID}
	for i, idx := range wantedRosterOrder {
		if *detail.Roster[i].HikerID != idx {
			t.Errorf("expected memberid: %s and got: %s", idx, detail.Roster[i].HikerID)
		}
	}

}

func Test_Detail_SeatsShortage(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(10, 2, StatusOpen, hostID, f)
	_ = seedMember(hostID, "hostman", "experienced", f)
	r1 := seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleRider, f, 0)
	_ = seedMember(r1.HikerID, "Yamal", "beginner", f)
	r2 := seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleRider, f, 0)
	_ = seedMember(r2.HikerID, "suluman", "beginner", f)

	detail, err := svc.Detail(context.Background(), o.ID, nil)
	if err != nil {
		t.Fatalf("Detail Shortage Test: %v", err)
	}
	// host + 2 riders = 3 people in 2 host seats: one short, no spots
	want := Capacity{PeopleCount: 3, SeatCapacity: 2, SeatsShort: 1, SpotsLeft: 0, WaitingCount: 0}
	if detail.Capacity != want {
		t.Errorf("capacity: got %+v, want %+v", detail.Capacity, want)
	}
}

func Test_Detail_MyRequest(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(10, 2, StatusOpen, hostID, f)
	_ = seedMember(hostID, "hostest", "experienced", f)
	r1 := seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleRider, f, 0)
	_ = seedMember(r1.HikerID, "FreeMan", "beginner", f)

	// viewer WITH a request → attached
	detailOfRequester, err := svc.Detail(context.Background(), o.ID, &r1.HikerID)
	if err != nil {
		t.Fatalf("requester view: %v", err)
	}
	if detailOfRequester.MyRequest == nil {
		t.Fatal("expected MyRequest populated for viewer with a request, got nil")
	}
	if detailOfRequester.MyRequest.ID != r1.ID {
		t.Errorf("expected request %s got %s", r1.ID, detailOfRequester.MyRequest.ID)
	}

	// viewer WITHOUT a request → nil, and crucially no error (the swallow)
	stranger := uuid.New()
	detailOfStranger, err := svc.Detail(context.Background(), o.ID, &stranger)
	if err != nil {
		t.Fatalf("stranger view must not error: %v", err)
	}
	if detailOfStranger.MyRequest != nil {
		t.Error("expected nil MyRequest for viewer without a request")
	}
}

func Test_Detail_RosterExcludesNonAccepted(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	o := seedOuting(10, 2, StatusOpen, hostID, f)
	seedMember(hostID, "combu", "experienced", f)
	accepted := uuid.New()
	requested := uuid.New()
	declined := uuid.New()
	_ = seedJoinRequest(o.ID, accepted, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(accepted, "javier", "experienced", f)
	_ = seedJoinRequest(o.ID, declined, RequestStatusDeclined, RoleRider, f, 0)
	seedMember(declined, "mazlum", "beginner", f)
	_ = seedJoinRequest(o.ID, requested, RequestStatusRequested, RoleRider, f, 0)
	seedMember(requested, "kalu", "intermediate", f)

	detail, err := svc.Detail(context.Background(), o.ID, &hostID)
	if err != nil {
		t.Fatalf("error on detail:%v", err)
	}
	// declined counts nowhere; requested counts as waiting, not as going
	want := Capacity{PeopleCount: 2, SeatCapacity: 2, SeatsShort: 0, SpotsLeft: 0, WaitingCount: 1}
	if detail.Capacity != want {
		t.Errorf("capacity: got %+v, want %+v", detail.Capacity, want)
	}
	if len(detail.Roster) != 1 {
		t.Fatalf("expected roster size was 1 but got %d", len(detail.Roster))
	}
	if *detail.Roster[0].HikerID != accepted {
		t.Errorf("roster[0] = %s, want the accepted hiker %s", detail.Roster[0].HikerID, accepted)
	}

}

func Test_Detail_Phase(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name       string
		start, end time.Time
		want       Phase
	}{
		{name: "upcoming", start: now.Add(30 * time.Hour), end: now.Add(50 * time.Hour), want: PhaseUpcoming},
		{name: "in progress", start: now.Add(-10 * time.Hour), end: now.Add(10 * time.Hour), want: PhaseInProgress},
		{name: "past", start: now.Add(-30 * time.Hour), end: now.Add(-20 * time.Hour), want: PhasePast},
	}
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			svc, f, _, _ := newTestService(t)
			host := uuid.New()
			end := cc.end
			o := seedOuting(6, 4, StatusOpen, host, f, withStart(cc.start), withEnd(&end))
			seedMember(host, "cafer", "beginner", f)
			detail, err := svc.Detail(context.Background(), o.ID, &host)
			if err != nil {
				t.Fatalf("expected no error got %v", err)
			}
			if detail.Outing.Phase != cc.want {
				t.Errorf("phase: got %q, want %q", detail.Outing.Phase, cc.want)
			}
		})
	}
}

func Test_Detail_VisibilityByViewer(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1, m2, m3 := uuid.New(), uuid.New(), uuid.New()
	stranger := uuid.New()

	o := seedOuting(6, 4, StatusOpen, host, f)
	r := seedJoinRequest(o.ID, m1, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(host, "salih", "beginner", f)
	seedMember(m1, "cafer", "beginner", f)

	seedJoinRequest(o.ID, m2, RequestStatusRequested, RoleDriver, f, 0, withSeats(2))
	seedJoinRequest(o.ID, m3, RequestStatusRequested, RoleRider, f, 0)
	seedMember(m2, "jale", "beginner", f) // the fake's waiting list skips hikers it can't name
	seedMember(m3, "hale", "beginner", f)

	// logged out: counts, no names
	d, err := svc.Detail(context.Background(), o.ID, nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(d.Roster) != 0 {
		t.Errorf("expected 0 got %d", len(d.Roster))
	}
	want := Capacity{PeopleCount: 2, SeatCapacity: 4, SeatsShort: 0, SpotsLeft: 2, WaitingCount: 2}
	if d.Capacity != want {
		t.Errorf("capacity: got %+v, want %+v", d.Capacity, want)
	}
	// stranger
	d, err = svc.Detail(context.Background(), o.ID, &stranger)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(d.Roster) != 1 {
		t.Fatalf("expected 1 got %d", len(d.Roster))
	}
	if d.Roster[0].RequestID != nil {
		t.Errorf("stranger: RequestID should be hidden, got %v", d.Roster[0].RequestID)
	}
	if d.Roster[0].HikerID != nil {
		t.Errorf("stranger: HikerID should be hidden, got %v", d.Roster[0].HikerID)
	}
	// member
	d, err = svc.Detail(context.Background(), o.ID, &m1)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(d.Roster) != 1 {
		t.Fatalf("expected 1 got %d", len(d.Roster))
	}
	if d.Roster[0].RequestID != nil {
		t.Errorf("member: RequestID should be hidden, got %v", d.Roster[0].RequestID)
	}
	if d.Roster[0].HikerID == nil || *d.Roster[0].HikerID != m1 {
		t.Errorf("member: expected HikerID %v got %v", m1, d.Roster[0].HikerID)
	}
	// host
	d, err = svc.Detail(context.Background(), o.ID, &host)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(d.Roster) != 1 {
		t.Fatalf("expected 1 got %d", len(d.Roster))
	}
	if d.Roster[0].RequestID == nil || *d.Roster[0].RequestID != r.ID {
		t.Errorf("host: expected RequestID %v got %v", r.ID, d.Roster[0].RequestID)
	}
	if d.Roster[0].HikerID == nil || *d.Roster[0].HikerID != m1 {
		t.Errorf("host: expected HikerID %v got %v", m1, d.Roster[0].HikerID)
	}

}

func Test_Detail_WaitingAndWithdrawnLists(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	m1, m2, m3, m4 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	regMem := uuid.New()
	stranger := uuid.New()

	o := seedOuting(10, 4, StatusOpen, host, f)
	seedMember(host, "mahir", "beginner", f)
	_ = seedJoinRequest(o.ID, regMem, RequestStatusAccepted, RoleDriver, f, 0)
	seedMember(regMem, "tahir", "beginner", f)

	now := time.Now()
	w1 := seedJoinRequest(o.ID, m1, RequestStatusRequested, RoleDriver, f, 1, withSeats(3))
	w1.RequestedAt = now.Add(-1 * time.Hour) // jale asked later
	seedMember(m1, "jale", "beginner", f)
	w2 := seedJoinRequest(o.ID, m2, RequestStatusRequested, RoleRider, f, 0)
	w2.RequestedAt = now.Add(-2 * time.Hour) // hale asked first
	seedMember(m2, "hale", "beginner", f)

	x1 := seedJoinRequest(o.ID, m3, RequestStatusWithdrawn, RoleRider, f, 0)
	x1.UpdatedAt = now.Add(-2 * time.Hour) // haluk withdrew first
	seedMember(m3, "haluk", "intermediate", f)
	x2 := seedJoinRequest(o.ID, m4, RequestStatusWithdrawn, RoleRider, f, 0)
	x2.UpdatedAt = now.Add(-1 * time.Hour) // kalu withdrew most recently
	seedMember(m4, "kalu", "beginner", f)

	d, err := svc.Detail(context.Background(), o.ID, nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(d.Waiting) != 0 {
		t.Fatalf("logged out: expected no waiting names, got %d", len(d.Waiting))
	}
	if len(d.Withdrawn) != 0 {
		t.Fatalf("logged out: expected no withdrawn names, got %d", len(d.Withdrawn))
	}

	// every logged-in viewer sees the same two lists in the same order
	viewers := []struct {
		name string
		id   uuid.UUID
	}{
		{"stranger", stranger},
		{"host", host},
		{"member", regMem},
	}
	for _, v := range viewers {
		t.Run(v.name, func(t *testing.T) {
			d, err := svc.Detail(context.Background(), o.ID, &v.id)
			if err != nil {
				t.Fatalf("expected no error got %v", err)
			}
			if len(d.Waiting) != 2 {
				t.Fatalf("expected 2 waiting, got %d", len(d.Waiting))
			}
			if len(d.Withdrawn) != 2 {
				t.Fatalf("expected 2 withdrawn, got %d", len(d.Withdrawn))
			}
			// jale drives, so she leads despite asking later
			if d.Waiting[0].Name != "jale" || d.Waiting[1].Name != "hale" {
				t.Errorf("waiting order: want [jale hale] got [%s %s]", d.Waiting[0].Name, d.Waiting[1].Name)
			}
			if d.Withdrawn[0].Name != "kalu" || d.Withdrawn[1].Name != "haluk" {
				t.Errorf("withdrawn order: want [kalu haluk] got [%s %s]", d.Withdrawn[0].Name, d.Withdrawn[1].Name)
			}
		})
	}
}

func Test_Detail_IsMe(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	h1, h2 := uuid.New(), uuid.New()
	o := seedOuting(6, 3, StatusOpen, hostID, f)
	seedMember(hostID, "sevket", "beginner", f)
	seedMember(h1, "salih", "beginner", f)
	seedMember(h2, "hale", "experienced", f)
	_ = seedJoinRequest(o.ID, h1, RequestStatusRequested, RoleRider, f, 0)
	_ = seedJoinRequest(o.ID, h2, RequestStatusRequested, RoleRider, f, 1)

	// as a waiter: exactly one row is me, and it is mine
	detail, err := svc.Detail(context.Background(), o.ID, &h1)
	if err != nil {
		t.Fatalf("detail as h1: %v", err)
	}
	if len(detail.Waiting) != 2 {
		t.Fatalf("expected 2 waiting got %d", len(detail.Waiting))
	}
	for _, w := range detail.Waiting {
		want := w.HikerID == h1
		if w.IsMe != want {
			t.Errorf("%s: IsMe = %v, want %v", w.Name, w.IsMe, want)
		}
	}

	// as the host: nobody is me
	detail, err = svc.Detail(context.Background(), o.ID, &hostID)
	if err != nil {
		t.Fatalf("detail as host: %v", err)
	}
	for _, w := range detail.Waiting {
		if w.IsMe {
			t.Errorf("%s: IsMe should be false for the host", w.Name)
		}
	}
}

func Test_Detail_IsMe_Withdrawn(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	h1, h2 := uuid.New(), uuid.New()
	o := seedOuting(6, 3, StatusOpen, hostID, f)
	seedMember(hostID, "sevket", "beginner", f)
	seedMember(h1, "salih", "beginner", f)
	seedMember(h2, "hale", "experienced", f)
	_ = seedJoinRequest(o.ID, h1, RequestStatusWithdrawn, RoleRider, f, 0)
	_ = seedJoinRequest(o.ID, h2, RequestStatusWithdrawn, RoleRider, f, 1)

	// as a waiter: exactly one row is me, and it is mine
	detail, err := svc.Detail(context.Background(), o.ID, &h1)
	if err != nil {
		t.Fatalf("detail as h1: %v", err)
	}
	if len(detail.Withdrawn) != 2 {
		t.Fatalf("expected 2 withdrawn got %d", len(detail.Withdrawn))
	}
	for _, w := range detail.Withdrawn {
		want := w.HikerID == h1
		if w.IsMe != want {
			t.Errorf("%s: IsMe = %v, want %v", w.Name, w.IsMe, want)
		}
	}

	// as the host: nobody is me
	detail, err = svc.Detail(context.Background(), o.ID, &hostID)
	if err != nil {
		t.Fatalf("detail as host: %v", err)
	}
	for _, w := range detail.Withdrawn {
		if w.IsMe {
			t.Errorf("%s: IsMe should be false for the host", w.Name)
		}
	}
}

func Test_Detail_Capacity(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hostID := uuid.New()
	h1, h2, d1 := uuid.New(), uuid.New(), uuid.New()

	o := seedOuting(6, 2, StatusOpen, hostID, f) // max 6, host seats 2
	seedMember(hostID, "sevket", "beginner", f)
	seedMember(h1, "salih", "beginner", f)
	seedMember(h2, "hale", "experienced", f)
	seedMember(d1, "jale", "intermediate", f)
	seedJoinRequest(o.ID, h1, RequestStatusAccepted, RoleRider, f, 2)                // 3 people
	seedJoinRequest(o.ID, d1, RequestStatusAccepted, RoleDriver, f, 1, withSeats(3)) // 2 people, 3 seats
	seedJoinRequest(o.ID, h2, RequestStatusRequested, RoleRider, f, 0)               // waiting

	want := Capacity{PeopleCount: 6, SeatCapacity: 5, SeatsShort: 1, SpotsLeft: 0, WaitingCount: 1}

	viewers := []struct {
		name   string
		viewer *uuid.UUID
	}{
		{"host", &hostID},
		{"logged out", nil},
	}
	for _, v := range viewers {
		t.Run(v.name, func(t *testing.T) {
			detail, err := svc.Detail(context.Background(), o.ID, v.viewer)
			if err != nil {
				t.Fatalf("detail: %v", err)
			}
			if detail.Capacity != want {
				t.Errorf("capacity: got %+v, want %+v", detail.Capacity, want)
			}
		})
	}

	// the count is public, the names are not
	detail, err := svc.Detail(context.Background(), o.ID, nil)
	if err != nil {
		t.Fatalf("detail as guest: %v", err)
	}
	if len(detail.Waiting) != 0 {
		t.Errorf("guest saw %d waiting names, want 0", len(detail.Waiting))
	}
}
