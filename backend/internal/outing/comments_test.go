package outing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

func Test_AddComment_MemberSucceeds(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, memberID := uuid.New(), uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, memberID, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(memberID, "mahmut", "experienced", f)

	c, err := svc.AddComment(context.Background(), memberID, o.ID, "Halo!", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if c.Body != "Halo!" || c.HikerID != memberID {
		t.Error("comment not stored as given")
	}
	if len(f.comments) != 1 {
		t.Errorf("expected 1 comment got %d", len(f.comments))
	}
}

func Test_AddComment_NoMemberForbidden(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, strangerID := uuid.New(), uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_, err := svc.AddComment(context.Background(), strangerID, o.ID, "Halo!", nil)
	wantStatus(t, err, apperr.CodeForbidden)
}

// riderID is a pending requester — this also proves the pending-audience door
func Test_AddComment_ReplyToTopLevel(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, driverID, riderID := uuid.New(), uuid.New(), uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_ = seedJoinRequest(o.ID, driverID, RequestStatusAccepted, RoleDriver, f, 0)
	seedMember(driverID, "Black Joe", "experienced", f)
	_ = seedJoinRequest(o.ID, riderID, RequestStatusRequested, RoleRider, f, 0)
	seedMember(riderID, "White Sam", "beginner", f)

	c, err := svc.AddComment(context.Background(), riderID, o.ID, "can i get a ride from 4564 mokako pl, Tacoma", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	reply, err := svc.AddComment(context.Background(), driverID, o.ID, "i can get you white sam. no worries", &c.ID)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if reply.ParentID == nil || *reply.ParentID != c.ID {
		t.Errorf("expected parent %s got %v", c.ID, reply.ParentID)
	}

}

func Test_AddComment_ReplyOnReplyReturnsConflict(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, salihAbi, recepI := uuid.New(), uuid.New(), uuid.New()

	o := seedOuting(6, 4, StatusOpen, hostID, f)

	_ = seedJoinRequest(o.ID, salihAbi, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(salihAbi, "Salih Abi", "experienced", f)
	_ = seedJoinRequest(o.ID, recepI, RequestStatusRequested, RoleRider, f, 0)
	seedMember(recepI, "Recep Ivedik", "beginner", f)

	c, err := svc.AddComment(context.Background(), recepI, o.ID, "Salih abi yolluk yaptin mi?", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	rep, err := svc.AddComment(context.Background(), salihAbi, o.ID, "Yaptim, Recep.", &c.ID)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.AddComment(context.Background(), recepI, o.ID, "Bira aldin mi?", &rep.ID)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_AddComment_ParentFromAnotherOutingReturnsBadRequest(t *testing.T) {
	svc, f, _ := newTestService(t)
	host1ID, host2ID, hikerID := uuid.New(), uuid.New(), uuid.New()

	o1 := seedOuting(6, 4, StatusOpen, host1ID, f)
	o2 := seedOuting(8, 4, StatusOpen, host2ID, f)

_:
	seedJoinRequest(o1.ID, hikerID, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(hikerID, "halit", "beginner", f)

	c1, err := svc.AddComment(context.Background(), host2ID, o2.ID, "rain forecast bring rainproof coat", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.AddComment(context.Background(), hikerID, o1.ID, "i will bring umbrella", &c1.ID)
	wantStatus(t, err, apperr.CodeBadRequest)

}

func Test_AddComment_CancelledAndPassedOutingReturnConflict(t *testing.T) {
	svc, f, _ := newTestService(t)
	host1ID, host2ID, hikerID := uuid.New(), uuid.New(), uuid.New()
	o1 := seedOutingWithStartTime(5, 3, StatusOpen, host1ID, f, time.Now().Add(-6*time.Hour))
	o2 := seedOuting(7, 3, StatusCancelled, host2ID, f)

	_ = seedJoinRequest(o1.ID, hikerID, RequestStatusAccepted, "rider", f, 0)
	_ = seedJoinRequest(o2.ID, hikerID, RequestStatusAccepted, "rider", f, 0)
	seedMember(hikerID, "deli yusuf", "experienced", f)

	_, err := svc.AddComment(context.Background(), hikerID, o1.ID, "olta getireyim mi?", nil)
	wantStatus(t, err, apperr.CodeConflict)
	_, err = svc.AddComment(context.Background(), hikerID, o2.ID, "sapan getireyim mi?", nil)
	wantStatus(t, err, apperr.CodeConflict)
}

func Test_AddComment_BodyOver2000Chs(t *testing.T) {
	body := strings.Repeat("abcdefgjkl", 201)
	svc, f, _ := newTestService(t)
	hostID, memberID := uuid.New(), uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, memberID, RequestStatusAccepted, RoleRider, f, 0)
	seedMember(memberID, "zafer", "experienced", f)

	_, err := svc.AddComment(context.Background(), memberID, o.ID, body, nil)
	wantStatus(t, err, apperr.CodeValidationError)
}

func Test_AddComment_PendingCanComment(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, memberID := uuid.New(), uuid.New()
	o := seedOuting(6, 4, StatusOpen, hostID, f)
	_ = seedJoinRequest(o.ID, memberID, RequestStatusRequested, RoleRider, f, 0)
	seedMember(memberID, "sener", "intermediate", f)
	c, err := svc.AddComment(context.Background(), memberID, o.ID, "lets eat something after hike", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if c.Body != "lets eat something after hike" {
		t.Errorf("expected %s got %s", "lets eat something after hike", c.Body)
	}
	if len(f.comments) != 1 {
		t.Errorf("expected comments len %d got %d", 1, len(f.comments))
	}
}

func Test_AddComment_ReplyOnDeletedParentReturnsConflict(t *testing.T) {
	svc, f, _ := newTestService(t)
	hostID, memberID := uuid.New(), uuid.New()
	o := seedOuting(8, 4, StatusOpen, hostID, f)

	_ = seedJoinRequest(o.ID, memberID, RequestStatusRequested, RoleRider, f, 0)
	seedMember(memberID, "kamil", "intermediate", f)

	c, err := svc.AddComment(context.Background(), hostID, o.ID, "kamil abi mangali getir", nil)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	deletedAt := time.Now().Add(-12 * time.Second)
	c.DeletedAt = &deletedAt
	_, err = svc.AddComment(context.Background(), memberID, o.ID, "sisme yatakta getirem mi?", &c.ID)
	wantStatus(t, err, apperr.CodeConflict)
}
