package message

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/events"
	"github.com/diagnosis/muster/internal/notification"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
)

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}

func Test_PostMessage_NonMember(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	outingID := uuid.New()
	host := uuid.New()
	members := []uuid.UUID{uuid.New(), uuid.New()}
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, members...)
	hikerID := uuid.New()
	svc := NewService(f, fb, fn)
	_, err := svc.PostMessage(context.Background(), conv.ID, hikerID, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_PostMessage_UnknownConversation(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	_, err := svc.PostMessage(context.Background(), uuid.New(), uuid.New(), "hello")
	wantStatus(t, err, apperr.CodeNotFound)
}

func Test_PostMessage_Happy(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if len(f.messages) != 1 {
		t.Fatalf("expected 1 message got %d", len(f.messages))
	}
	var got Message

	for _, m := range f.messages {
		got = m
	}

	if got.Body != "hello" {
		t.Errorf("expected body hello got %s", got.Body)
	}
	if got.ConversationID != conv.ID {
		t.Errorf("expected %v got %v", got.ConversationID, conv.ID)
	}
	if got.Seq != 1 {
		t.Errorf("expected 1 seq got %d", got.Seq)
	}
	if got.HikerID != m1 {
		t.Errorf("expected %v got %v", m1, got.HikerID)
	}
	for _, usr := range []uuid.UUID{host, m1, m2} {
		got := fb.sentTo(usr)
		if len(got) != 1 {
			t.Fatalf("hiker %v: want 1 event, got %d", usr, len(got))
		}
		if got[0].Type != "message.created" {
			t.Errorf("hiker %v: type = %q", usr, got[0].Type)
		}
		var p poke
		if err := json.Unmarshal([]byte(got[0].Data), &p); err != nil {
			t.Fatalf("hiker %v: bad poke json: %v", usr, err)
		}
		if p.ConversationID != conv.ID || p.ConversationKind != ConversationKindOuting {
			t.Errorf("hiker %v: poke = %+v", usr, p)
		}
		if strings.Contains(got[0].Data, "hello") {
			t.Errorf("poke carries the body: %s", got[0].Data)
		}

	}
	if got := fb.sentTo(uuid.New()); len(got) != 0 {
		t.Errorf("stranger got %v", got)
	}

}

func Test_PostMessage_OutingCancelled(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusCancelled, m1, m2)
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_PostMessage_BadBody(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)

	tests := []struct {
		name     string
		body     string
		wantErr  bool
		expected apperr.Status
	}{
		{name: "empty", body: "", wantErr: true, expected: apperr.CodeBadRequest},
		{name: "empty with space", body: "   ", wantErr: true, expected: apperr.CodeBadRequest},
		{name: "501 chars", body: strings.Repeat("a", 501), wantErr: true, expected: apperr.CodeBadRequest},
		{name: "500 chars", body: strings.Repeat("a", 500), wantErr: false},
		{name: "500 chars non-english", body: strings.Repeat("ş", 500), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.PostMessage(context.Background(), conv.ID, m1, tt.body)
			if tt.wantErr {
				wantStatus(t, err, tt.expected)
			} else if err != nil {
				t.Fatalf("expected no err got %v", err)
			}

		})
	}

}

func Test_PostMessage_Limit(t *testing.T) {
	clock := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	svc.now = func() time.Time {
		return clock
	}
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	for i := 0; i < 10; i++ {
		_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
		if err != nil {
			t.Fatalf("expected no error got %v", err)
		}
	}
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	wantStatus(t, err, apperr.CodeTooManyRequests)
	clock = clock.Add(61 * time.Second)
	_, err = svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
}

func Test_ListMessage_Happy(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	messages, err := svc.ListMessages(context.Background(), conv.ID, m1)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	if messages[0].Body != "hello" {
		t.Errorf("expected body hello got %s", messages[0].Body)
	}
	if messages[1].Body != "hi, m1" {
		t.Errorf("expected body hi, m1 got %s", messages[1].Body)
	}
	if messages[0].Seq != 1 {
		t.Errorf("expected seq 1 got %d", messages[0].Seq)
	}
	if messages[1].Seq != 2 {
		t.Errorf("expected seq 2 got %d", messages[1].Seq)
	}
}

func Test_ListMessage_Stranger(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	stranger := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, host, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	_, err = svc.ListMessages(context.Background(), conv.ID, stranger)
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_ListMessage_UnknownConversation(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	randomConv := uuid.New()
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, host, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	_, err = svc.ListMessages(context.Background(), randomConv, m1)
	wantStatus(t, err, apperr.CodeNotFound)
}

// delete message tests
func newOutingConv(t *testing.T) (*fakeStore, *fakeBroadcaster, *fakeNotificationStore, *Service, *Conversation, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	f := newFakeStore()
	fb := newFakeBroadcaster()
	fn := &fakeNotificationStore{}
	svc := NewService(f, fb, fn)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)
	return f, fb, fn, svc, conv, host, m1, m2
}

func Test_DeleteMessage_AuthorDeletes(t *testing.T) {
	f, fb, _, svc, conv, host, m1, m2 := newOutingConv(t)
	message1, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	fb.sent = nil
	err = svc.DeleteMessage(context.Background(), message1.ID, m1)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 1 {
		t.Fatalf("expected 1 messages got %d", len(f.messages))
	}
	if _, ok := f.messages[message1.ID]; ok {
		t.Fatal("expected message1 to be deleted")
	}

	for _, usr := range []uuid.UUID{host, m1, m2} {
		got := fb.sentTo(usr)
		if len(got) != 1 {
			t.Fatalf("hiker %v: want 1 event, got %d", usr, len(got))
		}
		if got[0].Type != "message.deleted" {
			t.Errorf("hiker %v: type = %q", usr, got[0].Type)
		}
		var p poke
		if err := json.Unmarshal([]byte(got[0].Data), &p); err != nil {
			t.Fatalf("hiker %v: bad poke json: %v", usr, err)
		}
		if p.ConversationID != conv.ID || p.ConversationKind != ConversationKindOuting {
			t.Errorf("hiker %v: poke = %+v", usr, p)
		}
	}

}

func Test_DeleteMessage_HostDeletes(t *testing.T) {
	f, fb, _, svc, conv, host, m1, m2 := newOutingConv(t)
	message1, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	fb.sent = nil
	err = svc.DeleteMessage(context.Background(), message1.ID, host)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 1 {
		t.Fatalf("expected 1 messages got %d", len(f.messages))
	}
	if _, ok := f.messages[message1.ID]; ok {
		t.Fatal("expected message1 to be deleted")
	}

	for _, usr := range []uuid.UUID{host, m1, m2} {
		got := fb.sentTo(usr)
		if len(got) != 1 {
			t.Fatalf("hiker %v: want 1 event, got %d", usr, len(got))
		}
		if got[0].Type != "message.deleted" {
			t.Errorf("hiker %v: type = %q", usr, got[0].Type)
		}
		var p poke
		if err := json.Unmarshal([]byte(got[0].Data), &p); err != nil {
			t.Fatalf("hiker %v: bad poke json: %v", usr, err)
		}
		if p.ConversationID != conv.ID || p.ConversationKind != ConversationKindOuting {
			t.Errorf("hiker %v: poke = %+v", usr, p)
		}
	}

}

func Test_DeleteMessage_OtherMemberDeletes(t *testing.T) {
	f, fb, _, svc, conv, _, m1, m2 := newOutingConv(t)
	message1, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	fb.sent = nil
	err = svc.DeleteMessage(context.Background(), message1.ID, m2)
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 2 {
		t.Errorf("expected 2 but got %d", len(f.messages))
	}
	if len(fb.sent) != 0 {
		t.Errorf("expected 0 but got %d", len(fb.sent))
	}

}

func Test_DeleteMessage_StrangerDeletes(t *testing.T) {
	f, fb, _, svc, conv, _, m1, m2 := newOutingConv(t)
	message1, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	fb.sent = nil
	err = svc.DeleteMessage(context.Background(), message1.ID, uuid.New())
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 2 {
		t.Errorf("expected 2 but got %d", len(f.messages))
	}
	if len(fb.sent) != 0 {
		t.Errorf("expected 0 but got %d", len(fb.sent))
	}

}

func Test_DeleteMessage_UnknownMessage(t *testing.T) {
	f, fb, _, svc, conv, _, m1, m2 := newOutingConv(t)
	_, err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	fb.sent = nil
	err = svc.DeleteMessage(context.Background(), uuid.New(), m1)
	wantStatus(t, err, apperr.CodeNotFound)
	if len(f.messages) != 2 {
		t.Errorf("expected 2 but got %d", len(f.messages))
	}
	if len(fb.sent) != 0 {
		t.Errorf("expected 0 but got %d", len(fb.sent))
	}

}

func Test_StartDM_CreatesPending(t *testing.T) {
	_, _, fn, svc, _, _, m1, m2 := newOutingConv(t)

	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if fn.events[0].HikerID != m2 {
		t.Errorf("expected hikerID: %v got %v", m2, fn.events[0].HikerID)
	}
	if fn.events[0].Kind != notification.KindDMRequested {
		t.Errorf("expected event_kind: %s got %s", notification.KindDMRequested, fn.events[0].Kind)
	}
	if fn.events[0].Payload["conversation_id"] != dm.ID.String() {
		t.Errorf("payload conversation_id = %v", fn.events[0].Payload["conversation_id"])
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	if dm.Kind != ConversationKindDM {
		t.Errorf("expected kind got %s", dm.Kind)
	}
	if dm.DmInitiator == nil || *dm.DmInitiator != m1 {
		t.Errorf("expected dminitiator %v got %v", m1, dm.DmInitiator)
	}
	if dm.DmA == nil || dm.DmB == nil || dm.DmA.String() > dm.DmB.String() {
		t.Errorf("expected dma smaller then dmb got %t", dm.DmA.String() < dm.DmB.String())
	}

}
func Test_StartDM_ReversedPair_SameConversation(t *testing.T) {
	f, fb, _, svc, _, _, m1, m2 := newOutingConv(t)
	dm1, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}
	if dm1.DmStatus == nil || *dm1.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm1.DmStatus)
	}
	dm2, err := svc.StartDM(context.Background(), m2, m1)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}

	if len(f.dms) != 1 {
		t.Fatalf("expected 1 got %d", len(f.dms))
	}
	if dm2.ID != dm1.ID {
		t.Fatalf("expected id unchanged got dm1: %v dm2: %v", dm1.ID, dm2.ID)
	}
	if *dm2.DmInitiator != m1 {
		t.Errorf("expected %v is initiator got %v", m1, *dm2.DmInitiator)
	}

}
func Test_StartDM_Self(t *testing.T) {
	_, _, _, svc, _, _, m1, _ := newOutingConv(t)
	_, err := svc.StartDM(context.Background(), m1, m1)
	wantStatus(t, err, apperr.CodeBadRequest)

}
func Test_StartDM_HostDmsRequester(t *testing.T) {
	f, _, _, svc, conv, host, _, _ := newOutingConv(t)
	hikerSelim := uuid.New()
	f.addPendingRequest(*conv.OutingID, hikerSelim)
	dm, err := svc.StartDM(context.Background(), host, hikerSelim)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmInitiator == nil || *dm.DmInitiator != host {
		t.Errorf("expected host: %v initiated got %v", host, *dm.DmInitiator)
	}
}
func Test_StartDM_MemberDMsPending(t *testing.T) {
	f, _, _, svc, conv, _, m1, _ := newOutingConv(t)
	hikerSelim := uuid.New()
	f.addPendingRequest(*conv.OutingID, hikerSelim)
	_, err := svc.StartDM(context.Background(), m1, hikerSelim)
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_StartDM_Strangers(t *testing.T) {
	f, _, _, svc, _, _, _, _ := newOutingConv(t)
	hikerCafer := uuid.New()
	hikerSero := uuid.New()
	_, err := svc.StartDM(context.Background(), hikerCafer, hikerSero)
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.dms) != 0 {
		t.Errorf("expected 0 dm got %v", len(f.dms))
	}
}

func Test_PostDM_Accepted(t *testing.T) {
	f, _, _, svc, _, _, m1, m2 := newOutingConv(t)
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	f.setDMStatus(dm.ID, DMStatusAccepted, nil)
	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.PostMessage(context.Background(), dm.ID, m2, "hi")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Errorf("expected 2 messages got %d", len(f.messages))
	}

}
func Test_PostDM_Declined(t *testing.T) {
	f, _, _, svc, _, _, m1, m2 := newOutingConv(t)
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	f.setDMStatus(dm.ID, DMStatusDeclined, nil)
	_, err = svc.PostMessage(context.Background(), dm.ID, m2, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 0 {
		t.Errorf("expected 0 messages got %d", len(f.messages))
	}
	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hi")
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 0 {
		t.Errorf("expected 0 messages got %d", len(f.messages))
	}

}
func Test_PostDM_Pending(t *testing.T) {
	clock := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	f, _, _, svc, _, _, m1, m2 := newOutingConv(t)
	svc.now = func() time.Time {
		return clock
	}
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hello, have ice axe?")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 1 {
		t.Errorf("expected 1 messages got %d", len(f.messages))
	}
	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hello, have micro spike?")
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 1 {
		t.Errorf("expected 1 messages got %d", len(f.messages))
	}
	_, err = svc.PostMessage(context.Background(), dm.ID, m2, "hi")
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 1 {
		t.Errorf("expected 1 messages got %d", len(f.messages))
	}
	clock = clock.Add(2 * time.Minute)
	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hello, got it?")
	wantStatus(t, err, apperr.CodeForbidden)
	if len(f.messages) != 1 {
		t.Errorf("expected 1 messages got %d", len(f.messages))
	}

}
func countType(es []events.Event, t string) int {
	count := 0
	for _, e := range es {
		if e.Type == t {
			count++
		}
	}
	return count
}
func Test_AcceptDM(t *testing.T) {
	f, fb, fn, svc, _, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fn.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fn.events))
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %d", countType(got, "dm.requested"))
	}
	fb.sent = nil
	err = svc.AcceptDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if countType(got, "notification.created") != 1 {
		t.Errorf("expected 1 notification poke, got %d", countType(got, "notification.created"))
	}
	got = fb.sentTo(m1)
	if countType(got, "dm.accepted") != 1 {
		t.Fatalf("expected 1 got %d", countType(got, "dm.accepted"))
	}
	if len(fn.events) != 2 {
		t.Fatalf("expected 2 notification, got %d", len(fn.events))
	}
	if fn.events[1].HikerID != m1 {
		t.Errorf("expected hikerID: %v got %v", m1, fn.events[1].HikerID)
	}
	if fn.events[1].Kind != notification.KindDMAccepted {
		t.Errorf("expected event_kind: %s got %s", notification.KindDMAccepted, fn.events[1].Kind)
	}
	if fn.events[1].Payload["conversation_id"] != dm.ID.String() {
		t.Errorf("payload conversation_id = %v", fn.events[1].Payload["conversation_id"])
	}

	if *f.converstations[dm.ID].DmStatus != DMStatusAccepted {
		t.Fatalf("expected %s got %s", DMStatusAccepted, *f.converstations[dm.ID].DmStatus)
	}

	message, err := svc.PostMessage(context.Background(), dm.ID, m2, "hello m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 1 {
		t.Fatalf("expected 1 message got %v", len(f.messages))
	}
	if message.ConversationID != dm.ID {
		t.Errorf("expected conv id: %v got: %v", dm.ID, message.ID)
	}
	if message.HikerID != m2 {
		t.Errorf("expected hiker id: %v got: %v", m2, message.HikerID)
	}
	if message.Body != "hello m1" {
		t.Errorf("expected message: hello m1 got: %s", message.Body)
	}

}

func Test_AcceptDM_Rejections(t *testing.T) {
	f, fb, _, svc, c, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}
	cases := []struct {
		name      string
		convID    uuid.UUID
		input     uuid.UUID
		setStatus bool
		dmStatus  DMStatus
		expected  apperr.Status
	}{
		{name: "initiator accepts", input: m1, expected: apperr.CodeForbidden, convID: dm.ID},
		{name: "stranger accepts", input: uuid.New(), expected: apperr.CodeForbidden, convID: dm.ID},
		{name: "outing conv", convID: c.ID, input: m2, expected: apperr.CodeBadRequest},
		{name: "already accepted", setStatus: true, dmStatus: DMStatusAccepted, input: m2, expected: apperr.CodeConflict, convID: dm.ID},
		{name: "unknow conv", convID: uuid.New(), input: m1, expected: apperr.CodeNotFound},
		{name: "declined", setStatus: true, dmStatus: DMStatusDeclined, input: m2, expected: apperr.CodeConflict, convID: dm.ID},
	}
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			if cc.setStatus {
				f.setDMStatus(cc.convID, cc.dmStatus, nil)
			}
			fb.sent = nil
			err = svc.AcceptDM(context.Background(), cc.convID, cc.input)
			wantStatus(t, err, cc.expected)
			if cc.expected == apperr.CodeForbidden {
				if *f.converstations[dm.ID].DmStatus != DMStatusPending {
					t.Errorf("status changed to %s on a forbidden accept", *f.converstations[dm.ID].DmStatus)
				}
				if len(fb.sent) != 0 {
					t.Errorf("poke sent on a forbidden accept: %v", fb.sent)
				}
			}
		})
	}
}

func Test_Decline_DM(t *testing.T) {
	f, fb, _, svc, _, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}
	fb.sent = nil
	err = svc.DeclineDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m1)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %v", len(got))
	}
	if got[0].Type != "dm.declined" {
		t.Fatalf("expected type: dm.declined got: %v", got[0].Type)
	}

	if *f.converstations[dm.ID].DmStatus != DMStatusDeclined {
		t.Fatalf("expected %s got %s", DMStatusDeclined, *f.converstations[dm.ID].DmStatus)
	}

	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "lolo yapma dayi")
	wantStatus(t, err, apperr.CodeForbidden)

	if *f.converstations[dm.ID].DmDeclinedBy != m2 {
		t.Errorf("expected declined by %v got %v", m2, *f.converstations[dm.ID].DmDeclinedBy)
	}

}

func Test_Decline_DM_Rejections(t *testing.T) {
	f, fb, _, svc, c, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}
	fb.sent = nil
	cases := []struct {
		name          string
		setStatus     bool
		status        DMStatus
		useOutingConv bool
		unknown       bool
		hikerID       uuid.UUID
		expected      apperr.Status
	}{
		{name: "stranger", hikerID: uuid.New(), expected: apperr.CodeForbidden},
		{name: "outing conv id", hikerID: m1, useOutingConv: true, expected: apperr.CodeBadRequest},
		{name: "unknown conv id", hikerID: m1, expected: apperr.CodeNotFound, unknown: true},
		{name: "already declined", setStatus: true, status: DMStatusDeclined, hikerID: m1, expected: apperr.CodeConflict},
	}
	for _, cc := range cases {

		t.Run(cc.name, func(t *testing.T) {
			convID := dm.ID
			if cc.useOutingConv {
				convID = c.ID
			}
			if cc.unknown {
				convID = uuid.New()
			}
			if cc.setStatus {
				f.setDMStatus(dm.ID, DMStatusDeclined, &m2)
			}
			err = svc.DeclineDM(context.Background(), convID, cc.hikerID)
			wantStatus(t, err, cc.expected)

		})
	}

}

func Test_Decline_AcceptedDM(t *testing.T) {
	f, fb, _, svc, _, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}

	fb.sent = nil
	err = svc.AcceptDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m1)
	if countType(got, "dm.accepted") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.accepted"))
	}

	fb.sent = nil
	err = svc.DeclineDM(context.Background(), dm.ID, m1)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m2)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %v", len(got))
	}
	if got[0].Type != "dm.declined" {
		t.Fatalf("expected type: dm.declined got: %v", got[0].Type)
	}

	if *f.converstations[dm.ID].DmStatus != DMStatusDeclined {
		t.Fatalf("expected %s got %s", DMStatusDeclined, *f.converstations[dm.ID].DmStatus)
	}

	_, err = svc.PostMessage(context.Background(), dm.ID, m2, "lolo yapma dayi")
	wantStatus(t, err, apperr.CodeForbidden)

	if *f.converstations[dm.ID].DmDeclinedBy != m1 {
		t.Errorf("expected declined by %v got %v", m1, *f.converstations[dm.ID].DmDeclinedBy)
	}

}

func Test_Reopen_DM(t *testing.T) {
	f, fb, fn, svc, _, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)

	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}

	fb.sent = nil
	err = svc.DeclineDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m1)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %v", len(got))
	}
	if countType(got, "dm.declined") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.declined"))
	}

	fb.sent = nil
	err = svc.ReopenDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m1)
	if countType(got, "dm.reopened") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.reopened"))
	}
	if len(fn.events) != 2 {
		t.Fatalf("expected 2 got %v", len(fn.events))
	}
	if fn.events[1].HikerID != m1 {
		t.Errorf("expected hikerID: %v got %v", m1, fn.events[1].HikerID)
	}
	if fn.events[1].Kind != notification.KindDMReopened {
		t.Errorf("expected event_kind: %s got %s", notification.KindDMReopened, fn.events[1].Kind)
	}
	if fn.events[1].Payload["conversation_id"] != dm.ID.String() {
		t.Errorf("payload conversation_id = %v", fn.events[1].Payload["conversation_id"])
	}

	if *f.converstations[dm.ID].DmStatus != DMStatusAccepted {
		t.Fatalf("expected %s got %s", DMStatusAccepted, *f.converstations[dm.ID].DmStatus)
	}
	if f.converstations[dm.ID].DmDeclinedBy != nil {
		t.Fatalf("expected declined by nil got %v", f.converstations[dm.ID].DmDeclinedBy)
	}

	_, err = svc.PostMessage(context.Background(), dm.ID, m1, "hello, hale. bak beni soktun ne hale")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

}

func Test_Reopen_Rejections(t *testing.T) {
	f, fb, _, svc, c, _, m1, m2 := newOutingConv(t)
	fb.sent = nil
	dm, err := svc.StartDM(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if dm.DmStatus == nil || *dm.DmStatus != DMStatusPending {
		t.Errorf("expected status pending got %v", dm.DmStatus)
	}
	got := fb.sentTo(m2)
	if countType(got, "dm.requested") != 1 {
		t.Fatalf("expected 1 got %v", countType(got, "dm.requested"))
	}

	fb.sent = nil
	err = svc.DeclineDM(context.Background(), dm.ID, m2)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got = fb.sentTo(m1)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %v", len(got))
	}
	if got[0].Type != "dm.declined" {
		t.Fatalf("expected type: dm.declined got: %v", got[0].Type)
	}

	cases := []struct {
		name         string
		dmID         uuid.UUID
		hikerID      uuid.UUID
		setNewStatus bool
		status       DMStatus
		expected     apperr.Status
	}{
		{name: "stranger", dmID: dm.ID, hikerID: uuid.New(), expected: apperr.CodeForbidden},
		{name: "outing conv", dmID: c.ID, hikerID: m2, expected: apperr.CodeBadRequest},
		{name: "unknown", dmID: uuid.New(), hikerID: m2, expected: apperr.CodeNotFound},
		{name: "other user attempt to reopen", dmID: dm.ID, hikerID: m1, expected: apperr.CodeForbidden},
		{name: "accepted cannot be reopened", dmID: dm.ID, hikerID: m2, expected: apperr.CodeConflict, setNewStatus: true, status: DMStatusAccepted},
		{name: "pending cannot be reopened", dmID: dm.ID, hikerID: m2, expected: apperr.CodeConflict, setNewStatus: true, status: DMStatusPending},
	}

	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			if cc.setNewStatus {
				f.setDMStatus(cc.dmID, cc.status, nil)
			}
			err = svc.ReopenDM(context.Background(), cc.dmID, cc.hikerID)
			wantStatus(t, err, cc.expected)
		})
	}

}
