package message

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/diagnosis/go-toolkit/v3/apperr"
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
	conv := f.addOutingConversation(outingID, host, members...)
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
	conv := f.addOutingConversation(outingID, host, m1, m2)
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
