package api

import (
	"encoding/json"
	"net/http"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/go-toolkit/v2/logger"
	"github.com/diagnosis/go-toolkit/v2/middleware"
	"github.com/diagnosis/go-toolkit/v2/responder"
	"github.com/diagnosis/muster/internal/hiker"
	"github.com/google/uuid"
)

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var in hiker.RegisterInput
	correlationID, _ := logger.GetCorrelationID(r.Context())
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		logger.Error(r.Context(), "invalid body", "err", err)
		responder.Error(w, apperr.BadRequest("invalid request body", "json decode failed", err), correlationID)
		return
	}
	h, err := s.hikers.Register(r.Context(), in)
	if err != nil {
		logger.Error(r.Context(), "failed to create a hiker.", "err", err)
		responder.Error(w, err, correlationID)
		return
	}

	responder.JSON(w, http.StatusCreated, h, correlationID)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	correlationID, _ := logger.GetCorrelationID(r.Context())
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		logger.Error(r.Context(), "invalid body", "err", err)
		responder.Error(w, apperr.BadRequest("invalid request body", "json decode failed", err), correlationID)
		return
	}

	session, err := s.hikers.Login(r.Context(), in.Email, in.Password, hiker.PlatformWeb)
	if err != nil {
		logger.Error(r.Context(), "failed to login", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	s.setSessionCookies(w, session)
	responder.JSON(w, http.StatusOK, session.Hiker, correlationID)

}

func (s *Server) setSessionCookies(w http.ResponseWriter, sess *hiker.Session) {
	secure := s.cfg.App.Env != "dev" // hardcoded true would eat cookies on localhost http
	http.SetCookie(w, &http.Cookie{
		Name: "access_token", Value: sess.AccessToken, Path: "/",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		MaxAge: int(s.cfg.JWT.AccessTokenExpiry.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name: "refresh_token", Value: sess.RefreshToken, Path: "/",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		MaxAge: int(s.cfg.JWT.RefreshTokenExpiry.Seconds()),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	// validate if refresh is fine
	c, err := r.Cookie("refresh_token")
	if err != nil {
		logger.Error(r.Context(), "failed to get refresh token", "err", err)
		responder.Error(w, apperr.TokenError("refresh token invalid or expired", "refresh token issue"), correlationID)
		return
	}
	// v0 is web only
	sess, err := s.hikers.Refresh(r.Context(), c.Value, hiker.PlatformWeb)
	if err != nil {
		logger.Error(r.Context(), "error capture session", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	s.setSessionCookies(w, sess)
	responder.JSON(w, http.StatusOK, sess.Hiker, correlationID)

}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	idStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Error(r.Context(), "user not logged in")
		responder.Error(w, apperr.Unauthorized("already logged out", "already logged out"), correlationID)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Error(r.Context(), "failed to parse id", "err", err)
		responder.Error(w, apperr.Unauthorized("unauthorized", "unauthorized"), correlationID)
		return
	}
	if err := s.hikers.Logout(r.Context(), id, hiker.PlatformWeb); err != nil {
		logger.Error(r.Context(), "failed to fetch logout service", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	s.clearSessionCookies(w)
	responder.JSON(w, http.StatusOK, nil, correlationID)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	correlationID, _ := logger.GetCorrelationID(r.Context())
	idStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Error(r.Context(), "user not logged in")
		responder.Error(w, apperr.Unauthorized("unauthorized", "user not logged in"), correlationID)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Error(r.Context(), "failed to parse id", "err", err)
		responder.Error(w, apperr.Unauthorized("unauthorized", "unauthorized"), correlationID)
		return
	}
	h, err := s.hikers.GetByID(r.Context(), id)
	if err != nil {
		logger.Error(r.Context(), "failed to get hiker", "err", err)
		responder.Error(w, err, correlationID)
		return
	}
	responder.JSON(w, http.StatusOK, h, correlationID)

}

func (s *Server) clearSessionCookies(w http.ResponseWriter) {
	secure := s.cfg.App.Env != "dev" // hardcoded true would eat cookies on localhost http
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   int(-1),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   int(-1),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
