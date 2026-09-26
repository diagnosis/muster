package api

import (
	"net/http"
	"time"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/responder"
	"github.com/diagnosis/muster/internal/events"
)

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to get authenticated user", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	client := &events.Client{
		HikerID: hikerID,
		Send:    make(chan events.Event, 16),
	}
	s.hub.Register(client)
	defer s.hub.Unregister(client)
	sse := events.NewSSEWriter(w)
	w.WriteHeader(http.StatusOK)
	err = sse.Ping()
	if err != nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-client.Send:
			if !ok {
				return
			}
			if err = sse.Send(ev.Type, ev.Data); err != nil {
				return
			}
		case <-ticker.C:
			if err = sse.Ping(); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}

}
