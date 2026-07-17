// Package api provides Muster's HTTP layer.
package api

import (
	"net/http"

	"github.com/diagnosis/go-toolkit/v2/middleware"
	"github.com/diagnosis/go-toolkit/v2/secure"
	"github.com/diagnosis/muster/internal/config"
	"github.com/diagnosis/muster/internal/hiker"
)

// Server wires HTTP routes to the domain services.
type Server struct {
	hikers *hiker.Service
	jwt    *secure.JWTSigner
	cfg    *config.Config
}

// NewServer returns a Server serving the given services.
func NewServer(cfg *config.Config, hikers *hiker.Service, jwt *secure.JWTSigner) *Server {
	return &Server{
		hikers: hikers,
		jwt:    jwt,
		cfg:    cfg,
	}
}

// Routes returns the fully wired HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	// public auth routes
	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	// protected auth routes
	requireAuth := middleware.RequireAuth(s.authFromCookie)
	mux.Handle("POST /api/auth/logout", requireAuth(http.HandlerFunc(s.handleLogout)))
	mux.Handle("GET /api/auth/me", requireAuth(http.HandlerFunc(s.handleMe)))

	var h http.Handler = mux
	h = middleware.CorrelationID()(h)
	return h
}

// v0 only uses cookie. mobile will need header
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
