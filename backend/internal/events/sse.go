package events

import (
	"fmt"
	"net/http"
)

// SSEWriter writes server-sent events to an http.ResponseWriter, flushing
// after every write so each event reaches the client immediately.
type SSEWriter struct {
	w http.ResponseWriter
}

// NewSSEWriter sets the stream response headers (text/event-stream, no-cache,
// X-Accel-Buffering: no) on w and returns a writer for it. Headers must be set
// before the caller writes the status.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	return &SSEWriter{w: w}
}

// Send writes one event with the given event name and data line, then flushes.
func (s *SSEWriter) Send(event, data string) error {
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	if err := http.NewResponseController(s.w).Flush(); err != nil {
		return err
	}
	return nil
}

// Ping writes an SSE comment line (": ping") and flushes. Clients ignore it; it
// keeps proxies and NATs from closing an idle stream (D11).
func (s *SSEWriter) Ping() error {
	if _, err := fmt.Fprint(s.w, ": ping\n\n"); err != nil {
		return err
	}
	if err := http.NewResponseController(s.w).Flush(); err != nil {
		return err
	}
	return nil
}
