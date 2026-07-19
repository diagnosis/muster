package api

import (
	"net/http"

	"github.com/diagnosis/go-toolkit/v2/logger"
	"github.com/diagnosis/go-toolkit/v2/responder"
	"github.com/diagnosis/muster/internal/outing"
)

func (s *Server) handleCreateOuting(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hostID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	var in outing.CreateInput

	if err = decodeJSON(r, &in); err != nil {
		logger.Warn(r.Context(), "bad request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	o, err := s.outings.Create(r.Context(), hostID, in)
	if err != nil {
		logger.Warn(r.Context(), "failed to create an outing", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusCreated, o, correlationID)

}

func (s *Server) handleRequestJoin(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hikerID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture outing id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	var in outing.JoinInput
	if err = decodeJSON(r, &in); err != nil {
		logger.Warn(r.Context(), "bad request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	jr, err := s.outings.RequestJoin(r.Context(), hikerID, outingID, in)
	if err != nil {
		logger.Warn(r.Context(), "failed to create joinRequest", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusCreated, jr, correlationID)
}

func (s *Server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hikerID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture outing id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	if err = s.outings.Withdraw(r.Context(), hikerID, outingID); err != nil {
		logger.Warn(r.Context(), "failed to withdraw from outing", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, map[string]string{
		"message": "withdrawn successfully",
	}, correlationID)
}

func (s *Server) handleAccept(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hostID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	jrID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture joinRequest id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	if err = s.outings.Accept(r.Context(), hostID, jrID); err != nil {
		logger.Warn(r.Context(), "failed to accept join request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "join request accepted successfully",
		},
		correlationID,
	)
}

func (s *Server) handleDecline(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hostID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	jrID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture joinRequest id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	if err = s.outings.Decline(r.Context(), hostID, jrID); err != nil {
		logger.Warn(r.Context(), "failed to decline join request", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "join request declined successfully",
		},
		correlationID,
	)

}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "failed to capture hostID", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	jrID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture joinRequest id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	if err = s.outings.RemoveMember(r.Context(), hostID, jrID); err != nil {
		logger.Warn(r.Context(), "failed to remove member from outing", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "member removed successfully",
		},
		correlationID,
	)

}

// handleCancelOuting cancels the outing ({id} = outing id). Host-only,
// enforced by the service.
func (s *Server) handleCancelOuting(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "outing: auth failed on cancel", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "outing: invalid outing id on cancel", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if err = s.outings.Cancel(r.Context(), hostID, outingID); err != nil {
		logger.Warn(r.Context(), "outing: cancel failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, map[string]string{"message": "outing cancelled"}, correlationID)
}
