package message

import (
	"context"
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
	outingID := uuid.New()
	host := uuid.New()
	members := []uuid.UUID{uuid.New(), uuid.New()}
	conv := f.addOutingConversation(outingID, host, members...)
	hikerID := uuid.New()
	svc := NewService(f)
	err := svc.PostMessage(context.Background(), conv.ID, hikerID, "hello")
	wantStatus(t, err, apperr.CodeForbidden)
}

func Test_GetConv_UnknownConversation(t *testing.T){
	f := newFakeStore()
	svc := NewService(f)
	err := svc.PostMessage(context.Background(), uuid.New(), uuid.New(), "hello")
	wantStatus(t, err, apperr.CodeNotFound)
}