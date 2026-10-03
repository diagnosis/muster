// backend/internal/notification/dispacher_test.go
package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test_drain_sendsUnsentAndStamps(t *testing.T) {
	f := &fakeNotificationStore{}
	fm := &fakeMailer{}
	d := NewDispatcher(f, fm, 30*time.Second, "https://test.com")

	hiker := uuid.New()

	for i := 0; i < 3; i++ {
		_ = f.Insert(context.Background(), &Event{HikerID: hiker, Kind: KindJoinRequestDeclined, Payload: map[string]any{"outing_title": "X"}})
	}

	sent := &Event{HikerID: hiker, Kind: KindOutingCancelled, Payload: map[string]any{"outing_title": "Y"}}
	_ = f.Insert(context.Background(), sent)
	now := time.Now()
	sent.EmailedAt = &now

	err := d.drain(context.Background())
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(fm.sent) != 3 {
		t.Fatalf("expected 3 sends, got %d", len(fm.sent))
	}
	for _, e := range f.events {
		if e.Kind == KindJoinRequestDeclined && e.EmailedAt == nil {
			t.Error("sent event was not stamped — will re-send next tick")
		}
	}
}

func Test_drain_DMKinds(t *testing.T) {
	f := &fakeNotificationStore{}
	fm := &fakeMailer{}
	d := NewDispatcher(f, fm, 30*time.Second, "https://test.com")
	convID := uuid.New().String()

	cases := []struct {
		name        string
		kind        Kind
		payload     map[string]any
		wantSubject string
		wantInBody  string
		wantCTAURL  string
	}{
		{
			name:        "dm requested",
			kind:        KindDMRequested,
			payload:     map[string]any{"conversation_id": convID, "from_name": "Ayla"},
			wantSubject: "Muster - new message request",
			wantInBody:  "Ayla",
			wantCTAURL:  "https://test.com/conversations/" + convID,
		},
		{
			name:        "dm accepted",
			kind:        KindDMAccepted,
			payload:     map[string]any{"conversation_id": convID, "from_name": "Zuzu"},
			wantSubject: "Muster - message request accepted",
			wantInBody:  "Zuzu",
			wantCTAURL:  "https://test.com/conversations/" + convID,
		},
		{
			name:        "dm reopened",
			kind:        KindDMReopened,
			payload:     map[string]any{"conversation_id": convID, "from_name": "Muku"},
			wantSubject: "Muster - message conversation reopened",
			wantInBody:  "Muku",
			wantCTAURL:  "https://test.com/conversations/" + convID,
		},
		{
			name:        "dm request with from_name empty string",
			kind:        KindDMRequested,
			payload:     map[string]any{"conversation_id": convID, "from_name": ""},
			wantSubject: "Muster - new message request",
			wantInBody:  "A hiker",
			wantCTAURL:  "https://test.com/conversations/" + convID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &Unsent{Event: Event{Kind: tc.kind, Payload: tc.payload}}
			got := d.contentFor(context.Background(), u)

			if got.Subject != tc.wantSubject {
				t.Errorf("subject: got %q, want %q", got.Subject, tc.wantSubject)
			}
			if !strings.Contains(got.Body, tc.wantInBody) {
				t.Errorf("body %q does not contain %q", got.Body, tc.wantInBody)
			}
			if got.CTAURL != tc.wantCTAURL {
				t.Errorf("cta url: got %q, want %q", got.CTAURL, tc.wantCTAURL)
			}
		})
	}
}
