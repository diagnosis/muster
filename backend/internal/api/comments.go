package api

import (
	"encoding/json"
	"net/http"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/responder"
	"github.com/google/uuid"
)

func (s *Server) handleListCommentViews(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "comment: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture outing id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	cvs, err := s.outings.ListComments(r.Context(), outingID, hikerID)
	if err != nil {
		logger.Warn(r.Context(), "failed to load comment views", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, map[string]any{
		"comments": cvs,
	}, correlationID)
}

// CommentInput receives right shape of input for comment
type CommentInput struct {
	Body     string     `json:"body"`
	ParentID *uuid.UUID `json:"parent_id"`
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "comment: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture outing id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	var in CommentInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err = dec.Decode(&in)
	if err != nil {
		logger.Warn(r.Context(), "failed to get comment body", "err", err)
		responder.Error(w, apperr.BadRequest("bad request: improper comment", "failed to decode comment", err), correlationID)
		return
	}

	c, err := s.outings.AddComment(r.Context(), hikerID, outingID, in.Body, in.ParentID)
	if err != nil {
		logger.Warn(r.Context(), "failed to add comment", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusCreated, c, correlationID)

}

func (s *Server) handleSoftDeleteComment(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "comment: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture outing id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	commentID, err := pathUUID(r, "cid")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture comment id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	err = s.outings.DeleteComment(r.Context(), outingID, commentID, hikerID)
	if err != nil {
		logger.Warn(r.Context(), "failed to delete comment", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, map[string]any{
		"message": "comment deleted",
	}, correlationID)

}

func (s *Server) handleLikeComment(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "comment: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	commentID, err := pathUUID(r, "cid")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture comment id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	err = s.outings.LikeComment(r.Context(), commentID, hikerID)
	if err != nil {
		logger.Warn(r.Context(), "failed to like comment", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, map[string]any{
		"message": "comment liked",
	}, correlationID)

}

func (s *Server) handleUnlikeComment(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "comment: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	commentID, err := pathUUID(r, "cid")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture comment id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if err = s.outings.UnlikeComment(r.Context(), commentID, hikerID); err != nil {
		logger.Warn(r.Context(), "failed to unlike comment", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, map[string]any{
		"message": "comment unliked",
	}, correlationID)
}
