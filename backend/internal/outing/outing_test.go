// backend/internal/outing/outing_test.go
// Tests for creating, updating and listing outings, and their phase.
package outing

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

func Test_ListUpcoming_SortedOpenFuture(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()

	late := seedOuting(6, 2, StatusOpen, host, f)

	far := seedOuting(6, 2, StatusOpen, host, f, withStart(time.Now().Add(64*time.Hour)))
	soon := seedOuting(6, 2, StatusOpen, host, f, withStart(time.Now().Add(12*time.Hour)))

	got, err := svc.ListUpcoming(context.Background())
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 outings, got %d", len(got))
	}

	wantOrder := []uuid.UUID{soon.ID, late.ID, far.ID}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, got[i].ID, id)
		}
	}

}

func Test_MyOutings_Buckets(t *testing.T) {
	svc, f, _, _ := newTestService(t)
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
		t.Fatalf("joined: len = %d, want 1", len(myOutings.Joined))
	}
	if myOutings.Joined[0].ID != approvedJoin.ID {
		t.Errorf("joined[0] = %s, want %s", myOutings.Joined[0].ID, approvedJoin.ID)
	}
}

func Test_MyOutings_Sorted(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	hikerID := uuid.New()
	earlySelf := seedOuting(6, 4, StatusOpen, hikerID, f, withStart(time.Now().Add(24*time.Hour)))
	lateSelf := seedOuting(6, 4, StatusOpen, hikerID, f, withStart(time.Now().Add(7*24*time.Hour)))

	earlyOther := seedOuting(6, 4, StatusOpen, uuid.New(), f, withStart(time.Now().Add(2*24*time.Hour)))
	_ = seedJoinRequest(earlyOther.ID, hikerID, RequestStatusAccepted, RoleRider, f, 1)
	lateOther := seedOuting(6, 4, StatusOpen, uuid.New(), f, withStart(time.Now().Add(10*24*time.Hour)))
	_ = seedJoinRequest(lateOther.ID, hikerID, RequestStatusAccepted, RoleDriver, f, 1)

	myOutings, err := svc.MyOutings(context.Background(), hikerID)
	if err != nil {
		t.Fatalf("MyOutings: %v", err)
	}
	if len(myOutings.Hosting) != 2 || len(myOutings.Joined) != 2 {
		t.Fatalf("expected 2 hosting and 2 joined, got %d and %d", len(myOutings.Hosting), len(myOutings.Joined))
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
	svc, _, _, _ := newTestService(t)
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

func Test_Update_HappyPatch(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusOpen, host, f)
	updatedTitle := "updated title"
	updatedSize := 5
	titleAndSizeUpdated, err := svc.Update(context.Background(), host, o.ID, UpdateInput{Title: &updatedTitle, MaxSize: &updatedSize})
	if err != nil {
		t.Fatalf("failed to update:%v", err)
	}
	if titleAndSizeUpdated.MaxSize != 5 {
		t.Errorf("expected maxsize: %d, got %d", 5, titleAndSizeUpdated.MaxSize)
	}
	if titleAndSizeUpdated.Title != "updated title" {
		t.Errorf("expected title: %s, got %s", "updated title", titleAndSizeUpdated.Title)
	}

}

func Test_Update_NonHostForbidden(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusOpen, host, f)
	updatedTitle := "updated title"
	updatedSize := 5
	_, err := svc.Update(context.Background(), uuid.New(), o.ID, UpdateInput{Title: &updatedTitle, MaxSize: &updatedSize})
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_Update_CancelledConflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusCancelled, host, f)
	updatedTitle := "updated title"
	updatedSize := 5
	_, err := svc.Update(context.Background(), host, o.ID, UpdateInput{Title: &updatedTitle, MaxSize: &updatedSize})
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_Update_PastBadRequest(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusOpen, host, f, withStart(time.Now().Add(-2*time.Hour)))
	updatedTitle := "won't matter"
	_, err := svc.Update(context.Background(), host, o.ID, UpdateInput{Title: &updatedTitle})
	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_Update_SeatShrinkAllowed(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusOpen, host, f)
	updatedHostSeats := 0
	updated, err := svc.Update(context.Background(), host, o.ID, UpdateInput{HostSeats: &updatedHostSeats})
	if err != nil {
		t.Fatalf("failed to shrink seats: %v", err)
	}
	if updated.HostSeats != 0 {
		t.Errorf("returned host seats = %d, want 0", updated.HostSeats)
	}
	if f.outings[o.ID].HostSeats != 0 {
		t.Errorf("stored host seats = %d, want 0", f.outings[o.ID].HostSeats)
	}
}

func Test_Update_InvalidPatchRejected(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(6, 3, StatusOpen, host, f)
	updateStart := time.Now().Add(1 * time.Hour)
	_, err := svc.Update(context.Background(), host, o.ID, UpdateInput{StartsAt: &updateStart})
	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_Update_MaxSizeBelowPeopleConflicts(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	o := seedOuting(8, 4, StatusOpen, host, f)
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleRider, f, 1)
	_ = seedJoinRequest(o.ID, uuid.New(), RequestStatusAccepted, RoleRider, f, 0)

	maxSize := 3
	_, err := svc.Update(context.Background(), host, o.ID, UpdateInput{MaxSize: &maxSize})
	wantStatus(t, err, apperr.CodeConflict)

	okSize := 4
	_, err = svc.Update(context.Background(), host, o.ID, UpdateInput{MaxSize: &okSize})
	if err != nil {
		t.Errorf("exact-fit capacity should be allowed, got %v", err)
	}
}

func Test_Outing_PhaseAt(t *testing.T) {
	start := time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Hour)

	cases := []struct {
		name   string
		endsAt *time.Time
		now    time.Time
		want   Phase
	}{
		{name: "before the start", now: start.Add(-time.Hour), want: PhaseUpcoming},
		{name: "exactly at the start", now: start, want: PhaseInProgress},
		{name: "no end, 6 hours in", now: start.Add(6 * time.Hour), want: PhaseInProgress},
		{name: "no end, exactly 12 hours in", now: start.Add(12 * time.Hour), want: PhasePast},
		{name: "no end, 13 hours in", now: start.Add(13 * time.Hour), want: PhasePast},
		{name: "has end, before it", endsAt: &end, now: start.Add(20 * time.Hour), want: PhaseInProgress},
		{name: "has end, exactly at it", endsAt: &end, now: end, want: PhasePast},
		{name: "has end, after it", endsAt: &end, now: end.Add(time.Hour), want: PhasePast},
		{name: "has end, before the start", endsAt: &end, now: start.Add(-time.Hour), want: PhaseUpcoming},
	}

	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			o := &Outing{StartsAt: start, EndsAt: cc.endsAt}
			if got := o.PhaseAt(cc.now); got != cc.want {
				t.Errorf("got %q, want %q", got, cc.want)
			}
		})
	}
}

func Test_ListUpcoming_Phase(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	inProgressEnd := time.Now().Add(10 * time.Hour)
	upComingEnd := time.Now().Add(50 * time.Hour)
	pastEnd := time.Now().Add(-10 * time.Hour)
	seedOuting(6, 4, StatusOpen, host, f, withStart(time.Now().Add(30*time.Hour)), withEnd(&upComingEnd))
	seedOuting(8, 4, StatusOpen, host, f, withStart(time.Now().Add(10*time.Hour)), withEnd(&upComingEnd))
	seedOuting(6, 4, StatusOpen, host, f, withStart(time.Now().Add(-30*time.Hour)), withEnd(&pastEnd))
	seedOuting(6, 4, StatusOpen, host, f, withStart(time.Now().Add(-10*time.Hour)), withEnd(&inProgressEnd))
	outings, err := svc.ListUpcoming(context.Background())
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(outings) != 2 {
		t.Fatalf("expected 2 got %d", len(outings))
	}
	for _, o := range outings {
		if o.Phase != PhaseUpcoming {
			t.Errorf("expected phase: %v got %v", PhaseUpcoming, o.Phase)
		}
	}
}

func Test_Update_DeadlineRejections(t *testing.T) {
	host := uuid.New()
	svc, f, _, _ := newTestService(t)

	title := "renamed"
	in3Days := time.Now().Add(72 * time.Hour)
	in1Hour := time.Now().Add(1 * time.Hour)

	cases := []struct {
		name     string
		startsAt time.Time
		in       UpdateInput
	}{
		{name: "outing in 6 hours, start moved 3 days out", startsAt: time.Now().Add(6 * time.Hour), in: UpdateInput{StartsAt: &in3Days}},
		{name: "outing in 6 hours, edit title", startsAt: time.Now().Add(6 * time.Hour), in: UpdateInput{Title: &title}},
		{name: "outing in 3 days, start moved to 1 hour from now", startsAt: time.Now().Add(72 * time.Hour), in: UpdateInput{StartsAt: &in1Hour}},
	}
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			o := seedOuting(6, 4, StatusOpen, host, f, withStart(cc.startsAt))
			_, err := svc.Update(context.Background(), host, o.ID, cc.in)
			wantStatus(t, err, apperr.CodeBadRequest)
		})
	}
}

func Test_Update_DeadlinePass(t *testing.T) {
	host := uuid.New()
	svc, f, _, _ := newTestService(t)
	o := seedOuting(6, 4, StatusOpen, host, f, withStart(time.Now().Add(18*time.Hour)))
	title := "Poo Poo Point"
	u, err := svc.Update(context.Background(), host, o.ID, UpdateInput{Title: &title})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if u.Title != title {
		t.Fatalf("expected %s got %s", title, u.Title)
	}
}

func Test_Create_StartsWithin24HoursRejected(t *testing.T) {
	host := uuid.New()
	svc, _, _, _ := newTestService(t)

	in := CreateInput{
		Title:       "Rattlesnake Ledge",
		Destination: "North Bend",
		MeetLabel:   "Trailhead lot",
		StartsAt:    time.Now().Add(18 * time.Hour),
		MaxSize:     6,
		HostSeats:   4,
		Difficulty:  DifficultyEasy,
		Pace:        PaceRelaxed,
	}
	_, err := svc.Create(context.Background(), host, in)
	wantStatus(t, err, apperr.CodeBadRequest)
}

func Test_MyOutings_Phase(t *testing.T) {
	svc, f, _, _ := newTestService(t)
	host := uuid.New()
	member := uuid.New()
	seedMember(member, "jale", "beginner", f)

	now := time.Now()
	endIn10h := now.Add(10 * time.Hour)
	ended200hAgo := now.Add(-200 * time.Hour)

	cases := []struct {
		name             string
		start            time.Time
		endsAt           *time.Time
		want             Phase
		isDiscussionOpen bool
	}{
		{name: "starts in 30h", start: now.Add(30 * time.Hour), want: PhaseUpcoming, isDiscussionOpen: true},
		{name: "started 2h ago, no end", start: now.Add(-2 * time.Hour), want: PhaseInProgress, isDiscussionOpen: true},
		{name: "started 20h ago, ends in 10h", start: now.Add(-20 * time.Hour), endsAt: &endIn10h, want: PhaseInProgress, isDiscussionOpen: true},
		{name: "started 13h ago, no end", start: now.Add(-13 * time.Hour), want: PhasePast, isDiscussionOpen: true},
		{name: "started 210h ago, ended 200h ago", start: now.Add(-210 * time.Hour), endsAt: &ended200hAgo, want: PhasePast, isDiscussionOpen: false},
	}

	want := map[uuid.UUID]Phase{}
	names := map[uuid.UUID]string{}
	discussionOpen := map[uuid.UUID]bool{}
	for _, cc := range cases {
		o := seedOuting(6, 4, StatusOpen, host, f, withStart(cc.start), withEnd(cc.endsAt))
		seedJoinRequest(o.ID, member, RequestStatusAccepted, "rider", f, 0)
		want[o.ID] = cc.want
		names[o.ID] = cc.name
		discussionOpen[o.ID] = cc.isDiscussionOpen
	}

	check := func(t *testing.T, list []Outing) {
		t.Helper()
		if len(list) != len(cases) {
			t.Fatalf("got %d outings, want %d", len(list), len(cases))
		}
		for _, o := range list {
			if o.Phase != want[o.ID] {
				t.Errorf("%s: got %q, want %q", names[o.ID], o.Phase, want[o.ID])
			}
			if o.IsDiscussionOpen != discussionOpen[o.ID] {
				t.Errorf("%s: got %t, want %t", names[o.ID], o.IsDiscussionOpen, discussionOpen[o.ID])
			}
		}
	}

	t.Run("hosting", func(t *testing.T) {
		mine, err := svc.MyOutings(context.Background(), host)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		check(t, mine.Hosting)
	})
	t.Run("joined", func(t *testing.T) {
		mine, err := svc.MyOutings(context.Background(), member)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		check(t, mine.Joined)
	})
}

func Test_CreateAndUpdate_Phase(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	host := uuid.New()
	starts := time.Now().Add(25 * time.Hour)
	ends := starts.Add(10 * time.Hour)
	in := CreateInput{
		Title:            "test",
		Destination:      "test",
		MeetLabel:        "test",
		HostSeats:        2,
		CostPerSeatCents: 20,
		MeetLat:          nil,
		MeetLng:          nil,
		StartsAt:         starts,
		MaxSize:          10,
		Difficulty:       DifficultyHard,
		Pace:             PaceRelaxed,
		Notes:            nil,
		EndsAt:           &ends,
	}
	o, err := svc.Create(context.Background(), host, in)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if o.Phase != PhaseUpcoming {
		t.Fatalf("expected %s got %s", PhaseUpcoming, o.Phase)
	}

	title := "updated"
	startsAt := time.Now().Add(48 * time.Hour)
	endsAt := time.Now().Add(60 * time.Hour)
	maxCount := 16
	u, err := svc.Update(context.Background(), host, o.ID, UpdateInput{StartsAt: &startsAt, EndsAt: &endsAt, MaxSize: &maxCount, Title: &title})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if u.Phase != PhaseUpcoming {
		t.Fatalf("expected %s got %s", PhaseUpcoming, u.Phase)
	}
}
