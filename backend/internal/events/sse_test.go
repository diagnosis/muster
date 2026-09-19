package events

import (
	"net/http/httptest"
	"testing"
)

func Test_SSE_Send(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w)
	err := sse.Send("message.created", "X")
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if !w.Flushed {
		t.Fatalf("expected flushed got false")
	}
	expected := "event: message.created\ndata: X\n\n"
	got := w.Body.String()
	if expected != got {
		t.Errorf("expected %q got %q", expected, got)
	}
}

func Test_SSE_Ping(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w)
	err := sse.Ping()
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	got := w.Body.String()
	expected := ": ping\n\n"
	if got != expected {
		t.Errorf("expected %q got %q", expected, got)
	}
}

func Test_SSE_Headers(t *testing.T) {
	w := httptest.NewRecorder()
	_ = NewSSEWriter(w)
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Error("expected text/event-stream content type")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Error("expected no-cache Cache-Control")
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Error("expected X-Accel-Buffering no")
	}
}
