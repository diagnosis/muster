package api

import (
	"net/http"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/responder"
)

type messageInput struct {
	Body string `json:"body"`
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "message: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	convID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture conv id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	var in messageInput
	if err = decodeJSON(r, &in); err != nil {
		logger.Warn(r.Context(), "bad request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	message, err := s.messages.PostMessage(r.Context(), convID, hikerID, in.Body)
	if err != nil {
		logger.Warn(r.Context(), "failed to post message", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusCreated, message, correlationID)
}
