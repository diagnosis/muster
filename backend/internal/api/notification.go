package api

import (
	"net/http"
	"strconv"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/responder"
)

func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())

	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "notification: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	limit := getLimit(r.URL.Query().Get("limit"), 20, 50)
	offset := getOffset(r.URL.Query().Get("offset"))

	events, err := s.notifications.ListForHiker(r.Context(), hikerID, limit, offset)
	if err != nil {
		responder.Error(w, err, correlationID)
		return
	}
	unreadCount, err := s.notifications.UnreadCount(r.Context(), hikerID)
	if err != nil {
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, map[string]any{
		"notifications": events,
		"unread_count":  unreadCount,
	}, correlationID)
}

func (s *Server) handleReadNotification(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "notification: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	notificationID, err := pathUUID(r, "id")
	if err != nil {
		logger.Warn(r.Context(), "failed to capture notification id", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if err = s.notifications.MarkRead(r.Context(), hikerID, notificationID); err != nil {
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, map[string]any{"message": "notification marked read"}, correlationID)
}

func (s *Server) handleReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	hikerID, err := getAuthenticatedUserID(r)
	if err != nil {
		logger.Warn(r.Context(), "notification: auth failed", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	if err = s.notifications.MarkAllRead(r.Context(), hikerID); err != nil {
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusOK, map[string]string{"message": "all notifications marked read"}, correlationID)

}

func getLimit(ls string, def, upper int) int {
	limit := def
	l, err := strconv.Atoi(ls)
	if err == nil {
		limit = l
	}
	if limit < 1 {
		limit = 1
	}
	if limit > upper {
		limit = upper
	}
	return limit
}

func getOffset(os string) int {
	offset := 0
	o, err := strconv.Atoi(os)
	if err == nil {
		offset = o
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
