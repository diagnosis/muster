package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/middleware"
	"github.com/diagnosis/muster/internal/events"
	"github.com/google/uuid"
)

func Test_EventHandler_NoUserID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()
	h := events.NewHub()
	s := &Server{hub: h}
	s.handleEvents(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected %d and got %d", http.StatusUnauthorized, w.Code)
	}
	if w.Header().Get("Content-Type") == "text/event-stream" {
		t.Error("expected to not get text/event-stream as Content-Type")
	}

}

func Test_EventHandler_RequestContextWithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hikerID := uuid.New()
	r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	r = r.WithContext(middleware.SetUserID(ctx, hikerID.String()))
	w := httptest.NewRecorder()
	done := make(chan struct{})
	h := events.NewHub()
	s := &Server{hub: h}
	go func() {
		s.handleEvents(w, r)
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for h.Count(hikerID) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("failed to get count within 1 second")
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.BroadcastToUser(hikerID, events.Event{
		Type: "message.created",
		Data: "X",
	})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not exit on cancel")
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type: text/event-stream got %s", w.Header().Get("Content-Type"))
	}
	expectedBody := "event: message.created\ndata: X\n\n"
	actualBody := w.Body.String()
	if !strings.Contains(actualBody, expectedBody) {
		t.Errorf("expected %q got %q", expectedBody, actualBody)
	}

}

func Test_EventHandler_Cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hikerID := uuid.New()
	r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	r = r.WithContext(middleware.SetUserID(ctx, hikerID.String()))
	w := httptest.NewRecorder()
	done := make(chan struct{})
	h := events.NewHub()
	s := &Server{hub: h}
	go func() {
		s.handleEvents(w, r)
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for h.Count(hikerID) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("failed to get count within 1 second")
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.BroadcastToUser(hikerID, events.Event{
		Type: "message.created",
		Data: "X",
	})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not exit on cancel")
	}
	if h.Count(hikerID) != 0 {
		t.Errorf("expected hiker count 0 got %d", h.Count(hikerID))
	}
	h.BroadcastToUser(hikerID, events.Event{Type: "message.created", Data: "c"})
}
