package hiker

import (
	"context"
	"strings"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/go-toolkit/v2/secure"
	"github.com/diagnosis/go-toolkit/v2/validator"
	"github.com/google/uuid"
)

// Storage is what the hiker service needs from persistence.
// Implemented by postgres.go.HikerStore.
type Storage interface {
	CreateHiker(ctx context.Context, h *Hiker) error
	GetHikerByEmail(ctx context.Context, email string) (*Hiker, error)
	GetHikerByID(ctx context.Context, id uuid.UUID) (*Hiker, error)
	UpdateHiker(ctx context.Context, h *Hiker) error

	SaveRefreshToken(ctx context.Context, t *RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	DeleteRefreshTokens(ctx context.Context, hikerID uuid.UUID, platform Platform) error
}

// Service implements hiker business rules over a Storage.
type Service struct {
	store Storage
	jwt   *secure.JWTSigner
}

// NewService returns a Service backed by store, minting tokens with jwt.
func NewService(store Storage, jwt *secure.JWTSigner) *Service {
	return &Service{
		store: store,
		jwt:   jwt,
	}
}

// RegisterInput carries the fields required to create a new hiker.
type RegisterInput struct {
	Email      string
	Password   string
	Name       string
	Experience Experience
	HomeArea   *string
	Bio        *string
	Gender     *string
}

// Register creates a new hiker with a hashed password. It returns
// apperr.Validation for bad input; duplicate emails surface from
// storage as apperr.EmailExists.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*Hiker, error) {

	email := strings.ToLower(strings.TrimSpace(in.Email))
	name := strings.TrimSpace(in.Name)
	v := validator.New()
	v.Required("email", email)
	v.Email("email", email)
	v.Required("password", in.Password)
	v.Password("password", in.Password)
	v.Required("name", name)
	verr := v.Errors()
	if verr != nil {
		return nil, verr
	}
	if !in.Experience.Valid() {
		return nil, apperr.BadRequest("experience is invalid", "invalid experience")
	}

	hash, err := secure.HashPassword(in.Password)
	if err != nil {
		return nil, apperr.Internal("could not process registration", "argon2 hash failed", err)
	}
	h := &Hiker{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Experience:   in.Experience,
		HomeArea:     in.HomeArea,
		Bio:          in.Bio,
		Gender:       in.Gender,
	}
	return h, s.store.CreateHiker(ctx, h)
}

// Platform identifies the client type a session belongs to.
type Platform string

// Known platforms, mirroring the user_refresh_tokens.platform values.
const (
	PlatformWeb     Platform = "web"
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// Valid reports whether p is a known platform.
func (p Platform) Valid() bool {
	switch p {
	case PlatformWeb, PlatformIOS, PlatformAndroid:
		return true
	}
	return false
}

// Login authenticates a hiker by email and password. It returns a
// Session on success and apperr.InvalidCredentials for any failure —
// deliberately not revealing whether the email or password was wrong.
func (s *Service) Login(ctx context.Context, email, password string, platform Platform) (*Session, error) {

	if !platform.Valid() {
		return nil, apperr.BadRequest("invalid platform", "invalid platform")
	}
	emailN := strings.ToLower(strings.TrimSpace(email))

	hiker, err := s.store.GetHikerByEmail(ctx, emailN)
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); ok && se.Status == apperr.CodeNotFound {
			return nil, apperr.InvalidCredentials("invalid login credentials", "no hiker for email")
		}
		return nil, err
	}

	ok, err := secure.VerifyPassword(password, hiker.PasswordHash)
	if err != nil {
		return nil, apperr.InvalidCredentials("invalid login credentials", "failed to verify password", err)
	}
	if !ok {
		return nil, apperr.InvalidCredentials("invalid login credentials", "password not match")
	}

	return s.mintSession(ctx, hiker, platform)
}

// Logout revokes the hiker's stored refresh token for the given
// platform. The access token remains valid until it expires — that's
// the price of stateless JWTs, and why they're short-lived.
func (s *Service) Logout(ctx context.Context, id uuid.UUID, platform Platform) error {
	if !platform.Valid() {
		return apperr.BadRequest("invalid platform", "invalid platform")
	}
	return s.store.DeleteRefreshTokens(ctx, id, platform)
}

// Refresh exchanges a valid raw refresh token for a new Session,
// rotating the stored token: the presented token is revoked and a
// replacement issued. Any credential-shaped failure returns
// apperr.InvalidCredentials.
func (s *Service) Refresh(ctx context.Context, rawToken string, platform Platform) (*Session, error) {
	if !platform.Valid() {
		return nil, apperr.BadRequest("invalid platform", "invalid platform")
	}

	rt, err := s.store.GetRefreshTokenByHash(ctx, secure.HashRefreshToken(rawToken))
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); ok && se.Status == apperr.CodeNotFound {
			return nil, apperr.InvalidCredentials("invalid credentials", "unknown refresh token")
		}
		return nil, err
	}
	if rt.ExpiresAt.Before(time.Now()) {
		return nil, apperr.InvalidCredentials("invalid credentials", "token expired")
	}
	if rt.Platform != platform {
		return nil, apperr.InvalidCredentials("invalid credentials", "platform mismatch")
	}

	hiker, err := s.store.GetHikerByID(ctx, rt.HikerID)
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); ok && se.Status == apperr.CodeNotFound {
			return nil, apperr.InvalidCredentials("invalid credentials", "unknown refresh token")
		}
		return nil, err
	}
	return s.mintSession(ctx, hiker, platform)

}
func (s *Service) mintSession(ctx context.Context, hiker *Hiker, platform Platform) (*Session, error) {
	accessToken, err := s.jwt.SignAccess(hiker.ID.String())
	if err != nil {
		return nil, apperr.Internal("could not create session", "access token signing failed", err)
	}
	newRawRefresh, err := secure.GenerateRefreshToken()
	if err != nil {
		return nil, apperr.Internal("could not create session", "refresh token generation failed", err)
	}

	newRt := &RefreshToken{
		ID:        uuid.New(),
		HikerID:   hiker.ID,
		TokenHash: secure.HashRefreshToken(newRawRefresh),
		Platform:  platform,
		ExpiresAt: time.Now().Add(s.jwt.RefreshExpiry()),
	}
	// v0 policy: one session per platform...
	if err := s.store.DeleteRefreshTokens(ctx, hiker.ID, platform); err != nil {
		return nil, err
	}
	if err := s.store.SaveRefreshToken(ctx, newRt); err != nil {
		return nil, err
	}
	return &Session{
		Hiker:        hiker,
		AccessToken:  accessToken,
		RefreshToken: newRawRefresh,
	}, nil
}
