package api

import (
	"context"
	"net/http"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/responder"
	"github.com/diagnosis/muster/internal/message"
	"github.com/google/uuid"
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

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
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
	messages, err := s.messages.ListMessages(r.Context(), convID, hikerID)
	if err != nil {
		logger.Warn(r.Context(), "failed to list messages", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if messages == nil {
		messages = []*message.Message{}
	}
	responder.JSON(w, http.StatusOK, map[string]any{
		"messages": messages,
	}, correlationID)
}

func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "message: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	messageID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture message id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	if err = s.messages.DeleteMessage(r.Context(), messageID, hikerID); err != nil {
		logger.Warn(r.Context(), "failed to delete message", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type dmInput struct {
	HikerID string `json:"hiker_id"`
}

func (s *Server) handleStartDM(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	me, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "dm: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	var in dmInput
	if err = decodeJSON(r, &in); err != nil {
		logger.Warn(r.Context(), "bad request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	other, err := uuid.Parse(in.HikerID)
	if err != nil {
		logger.Warn(r.Context(), "bad uuid", "err", err)
		responder.Error(w, apperr.BadRequest("bad uuid", "bad uuid", err), correlationID)
		return
	}

	c, err := s.messages.StartDM(r.Context(), me, other)
	if err != nil {
		logger.Error(r.Context(), "failed to start dm", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusCreated, c, correlationID)

}

func (s *Server) handleAcceptDM(w http.ResponseWriter, r *http.Request) {
	s.dmTransition(w, r, s.messages.AcceptDM, "accept DM")
}

func (s *Server) handleDeclineDM(w http.ResponseWriter, r *http.Request) {
	s.dmTransition(w, r, s.messages.DeclineDM, "decline DM")
}

func (s *Server) handleReopenDM(w http.ResponseWriter, r *http.Request) {
	s.dmTransition(w, r, s.messages.ReopenDM, "reopen DM")
}

func (s *Server) dmTransition(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, convID, actor uuid.UUID) error, label string) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	me, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "dm: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	convID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to parse uuid for conversation", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	err = fn(r.Context(), convID, me)
	if err != nil {
		logger.Warn(r.Context(), label+":failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	me, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "conv: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	convs, err := s.messages.ListConversations(r.Context(), me)
	if err != nil {
		logger.Warn(r.Context(), "failed to get conversations", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if convs == nil {
		convs = []*message.ConversationSummary{}
	}

	responder.JSON(w, http.StatusOK, map[string]any{
		"conversations": convs,
	}, correlationID)
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	me, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "dm: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	convID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to parse uuid", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	conv, err := s.messages.GetConversation(r.Context(), convID, me)
	if err != nil {
		logger.Warn(r.Context(), "failed to get conversation", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, conv, correlationID)
}
