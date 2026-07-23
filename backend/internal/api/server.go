// Package api provides Muster's HTTP layer.
package api

import (
	"net/http"
	"time"

	"github.com/diagnosis/go-toolkit/v2/middleware"
	"github.com/diagnosis/go-toolkit/v2/secure"
	"github.com/diagnosis/muster/internal/config"
	"github.com/diagnosis/muster/internal/hiker"
	"github.com/diagnosis/muster/internal/outing"
	"golang.org/x/time/rate"
)

// Server wires HTTP routes to the domain services.
type Server struct {
	hikers  *hiker.Service
	jwt     *secure.JWTSigner
	cfg     *config.Config
	outings *outing.Service
}

// NewServer returns a Server serving the given services.
func NewServer(cfg *config.Config, hikers *hiker.Service, jwt *secure.JWTSigner, outings *outing.Service) *Server {
	return &Server{
		hikers:  hikers,
		jwt:     jwt,
		cfg:     cfg,
		outings: outings,
	}
}

// Routes returns the fully wired HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	// public auth routes
	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	// optionalAuth
	optionalAuth := middleware.OptionalAuth(s.authFromCookie)

	// protected auth routes
	requireAuth := middleware.RequireAuth(s.authFromCookie)
	mux.Handle("POST /api/auth/logout", requireAuth(http.HandlerFunc(s.handleLogout)))
	mux.Handle("GET /api/auth/me", requireAuth(http.HandlerFunc(s.handleMe)))

	// public outing routes
	mux.HandleFunc("GET /api/outings", s.handleListUpcoming)
	mux.Handle("GET /api/outings/{id}", optionalAuth(http.HandlerFunc(s.handleDetail)))
	// protected outing routes
	mux.Handle("POST /api/outings", requireAuth(http.HandlerFunc(s.handleCreateOuting)))
	mux.Handle("POST /api/outings/{id}/requests", requireAuth(http.HandlerFunc(s.handleRequestJoin)))
	mux.Handle("DELETE /api/outings/{id}/requests/me", requireAuth(http.HandlerFunc(s.handleWithdraw)))
	mux.Handle("POST /api/requests/{id}/accept", requireAuth(http.HandlerFunc(s.handleAccept)))
	mux.Handle("POST /api/requests/{id}/decline", requireAuth(http.HandlerFunc(s.handleDecline)))
	mux.Handle("DELETE /api/requests/{id}/member", requireAuth(http.HandlerFunc(s.handleRemoveMember)))
	mux.Handle("POST /api/outings/{id}/cancel", requireAuth(http.HandlerFunc(s.handleCancelOuting)))
	mux.Handle("GET /api/outings/{id}/requests", requireAuth(http.HandlerFunc(s.handlePendingRequests)))
	mux.Handle("GET /api/me/outings", requireAuth(http.HandlerFunc(s.handleMyOutings)))

	var h http.Handler = mux
	h = middleware.RateLimit(rate.Limit(10), 20, 5*time.Minute)(h)
	h = middleware.CorrelationID()(h)
	return h
}

// authFromCookie authenticates a request from the access_token cookie,
// returning the verified user id (JWT sub claim). v0 is cookie-only;
// mobile clients will need an Authorization-header path alongside this.
func (s *Server) authFromCookie(r *http.Request) (string, error) {
	c, err := r.Cookie("access_token")
	if err != nil {
		return "", err
	}
	claims, err := s.jwt.VerifyAccess(c.Value)
	if err != nil {
		return "", err
	}
	return claims.Sub, nil

}
