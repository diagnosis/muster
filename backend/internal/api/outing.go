package api

import (
	"net/http"

	"github.com/diagnosis/go-toolkit/v2/logger"
	"github.com/diagnosis/go-toolkit/v2/responder"
	"github.com/diagnosis/muster/internal/outing"
)

// handleCreateOuting creates a new outing hosted by the authenticated
// hiker. Validation (24h lead time, sizes, enums) lives in the service.
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

// handleRequestJoin creates or revives the caller's join request on the
// outing ({id} = outing id). A withdrawn request flips back to requested,
// keeping its original role/seats/guests.
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

// handleWithdraw pulls the caller's own request from the outing
// ({id} = outing id), whether pending or already accepted.
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

// handleAccept approves a pending join request ({id} = request id).
// Host-only; capacity is checked atomically in the store.
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

// handleDecline rejects a pending join request ({id} = request id).
// Host-only; declined is terminal for this outing.
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

// handleRemoveMember removes an accepted member from the roster
// ({id} = request id). Host-only.
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

// handleListUpcoming returns upcoming open outings, soonest first. Public — the teaser for hikers without accounts.
func (s *Server) handleListUpcoming(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	outings, err := s.outings.ListUpcoming(r.Context())
	if err != nil {
		logger.Warn(r.Context(), "outing: list upcoming failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, outings, correlationID)
}

// handlePendingRequests returns the outing's pending join requests ({id} = outing id), oldest first. Host-only, enforced by the service.
func (s *Server) handlePendingRequests(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hostID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "outing: auth failed on pending requests", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	outingID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "outing: invalid outing id on pending requests", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	jrs, err := s.outings.PendingRequests(r.Context(), hostID, outingID)
	if err != nil {
		logger.Warn(r.Context(), "outing: pending requests failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, jrs, correlationID)
}

// handleMyOutings returns the caller's hosted and joined outings, each soonest first
func (s *Server) handleMyOutings(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "outings: auth failed on my outings", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	myOutings, err := s.outings.MyOutings(r.Context(), hikerID)
	if err != nil {
		logger.Warn(r.Context(), "outings: my outings failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, myOutings, correlationID)
}
