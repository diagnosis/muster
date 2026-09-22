package message

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
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
	outingID := uuid.New()
	host := uuid.New()
	members := []uuid.UUID{uuid.New(), uuid.New()}
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, members...)
	hikerID := uuid.New()
	svc := NewService(f, fb)
	err := svc.PostMessage(context.Background(), conv.ID, hikerID, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_GetConv_UnknownConversation(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	svc := NewService(f, fb)
	err := svc.PostMessage(context.Background(), uuid.New(), uuid.New(), "hello")
	wantStatus(t, err, apperr.CodeNotFound)
}

func Test_PostMessage_Happy(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	svc := NewService(f, fb)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
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
		if p.ConversationID != conv.ID || p.Kind != ConversationKindOuting {
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
	svc := NewService(f, fb)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusCancelled, m1, m2)
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_PostMessage_BadBody(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	svc := NewService(f, fb)
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
			err := svc.PostMessage(context.Background(), conv.ID, m1, tt.body)
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
	svc := NewService(f, fb)
	svc.now = func() time.Time {
		return clock
	}
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	for i := 0; i < 10; i++ {
		err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
		if err != nil {
			t.Fatalf("expected no error got %v", err)
		}
	}
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	wantStatus(t, err, apperr.CodeTooManyRequests)
	clock = clock.Add(61 * time.Second)
	err = svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
}

func Test_ListMessage_Happy(t *testing.T) {
	f := newFakeStore()
	fb := newFakeBroadcaster()
	svc := NewService(f, fb)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1, m2)
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.PostMessage(context.Background(), conv.ID, m2, "hi, m1")
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
	svc := NewService(f, fb)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	stranger := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.PostMessage(context.Background(), conv.ID, host, "hi, m1")
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
	svc := NewService(f, fb)
	outingID := uuid.New()
	host := uuid.New()
	m1 := uuid.New()
	conv := f.addOutingConversation(outingID, host, outing.StatusOpen, m1)
	randomConv := uuid.New()
	err := svc.PostMessage(context.Background(), conv.ID, m1, "hello")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.PostMessage(context.Background(), conv.ID, host, "hi, m1")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.messages) != 2 {
		t.Fatalf("expected 2 messages got %d", len(f.messages))
	}
	_, err = svc.ListMessages(context.Background(), randomConv, m1)
	wantStatus(t, err, apperr.CodeNotFound)
}
