package authtoken

import (
	"context"
	"errors"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/secure"
	"github.com/google/uuid"
)

// Storage implemented by postgres and fakes; MarkUsed persists the burn, rules live in the service
type Storage interface {
	Save(ctx context.Context, token *Token) error
	GetByHash(ctx context.Context, hash string) (*Token, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	LiveByHiker(ctx context.Context, hikerID uuid.UUID, purpose Purpose) (*Token, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
}

// Service mints and consumes single-use auth tokens over a Storage
type Service struct {
	store Storage
	otp   func() (string, error)
	now   func() time.Time
}

// NewService returns Service
func NewService(store Storage) *Service {
	return &Service{store: store, otp: secure.GenerateOTP, now: time.Now}
}

// Mint returns the raw token; only its hash is stored (the one fact every caller must know)
func (s *Service) Mint(ctx context.Context, hikerID uuid.UUID, purpose Purpose, ttl time.Duration) (string, error) {
	if !purpose.Valid() {
		return "", apperr.BadRequest("invalid purpose", "invalid purpose")
	}
	raw, err := secure.GenerateRefreshToken()
	if err != nil {
		return "", apperr.Internal("failed to generate token", "token create failed", err)
	}
	t := &Token{
		ID:        uuid.New(),
		HikerID:   hikerID,
		Hash:      secure.HashRefreshToken(raw),
		Purpose:   purpose,
		CreatedAt: s.now(),
		ExpiresAt: s.now().Add(ttl),
		UsedAt:    nil,
	}
	if err = s.store.Save(ctx, t); err != nil {
		return "", err
	}

	return raw, nil
}

// Consume validates purpose, single-use, and expiry, then burns the token; a token never consumes twice (the contract the four tests pin)
func (s *Service) Consume(ctx context.Context, raw string, purpose Purpose) (uuid.UUID, error) {
	t, err := s.lookup(ctx, raw, purpose)
	if err != nil {
		return uuid.Nil, err
	}
	if err = s.store.MarkUsed(ctx, t.ID); err != nil {
		return uuid.Nil, err
	}

	return t.HikerID, nil
}

// Check reports whether raw is a token that Consume would accept for purpose
// right now: issued, for this purpose, unused and unexpired. It does not burn
// the token, so it can be called any number of times. A page uses it to decide
// whether to show a form; the action itself must still call Consume, because a
// token can expire or be used between the check and the submit.
func (s *Service) Check(ctx context.Context, raw string, purpose Purpose) error {
	if _, err := s.lookup(ctx, raw, purpose); err != nil {
		return err
	}

	return nil
}

// lookup is the one home of the validity rule shared by Check and Consume. It
// finds the token by the hash of raw and refuses one that was issued for a
// different purpose, has already been used, or has expired. The returned token
// is valid at the moment of the call; nothing is written.
func (s *Service) lookup(ctx context.Context, raw string, purpose Purpose) (*Token, error) {
	if !purpose.Valid() {
		return nil, apperr.BadRequest("invalid purpose", "invalid purpose")
	}

	hash := secure.HashRefreshToken(raw)
	t, err := s.store.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if t.Purpose != purpose {
		return nil, apperr.Conflict("purpose mismatch", "purpose mismatch")
	}

	if t.UsedAt != nil {
		return nil, apperr.Conflict("token is used", "token is used")
	}
	if t.ExpiresAt.Before(s.now()) {
		return nil, apperr.Conflict("token is expired", "token is expired")
	}
	return t, nil
}

// ErrTooSoon is returned by MintCode when the hiker already holds a login
// code younger than the resend window. Callers treat it as "nothing to do".
var ErrTooSoon = errors.New("login code requested too soon")

// MintCode issues a fresh six-digit login code for hikerID valid for ttl and
// returns it in the clear for mailing. Any live code is burned first; a code
// younger than the resend window returns ErrTooSoon and mints nothing.
func (s *Service) MintCode(ctx context.Context, hikerID uuid.UUID, ttl time.Duration) (string, error) {

	tok, err := s.store.LiveByHiker(ctx, hikerID, PurposeLoginCode)
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); !ok || se.Status != apperr.CodeNotFound {
			return "", err
		}

	} else {
		if tok.CreatedAt.After(s.now().Add(-resendWindow)) {
			return "", ErrTooSoon
		}
		if err = s.store.MarkUsed(ctx, tok.ID); err != nil {
			return "", err
		}
	}

	raw, err := s.otp()
	if err != nil {
		return "", err
	}
	t := &Token{
		ID:        uuid.New(),
		HikerID:   hikerID,
		Hash:      hashCode(hikerID, raw),
		Purpose:   PurposeLoginCode,
		CreatedAt: s.now(),
		ExpiresAt: s.now().Add(ttl),
		UsedAt:    nil,
		Attempts:  0,
	}

	if err = s.store.Save(ctx, t); err != nil {
		return "", err
	}

	return raw, nil
}

const maxCodeAttempts = 5

// resendWindow is how long a hiker must wait after a login code before asking for another.
const resendWindow = 30 * time.Second

// ErrBadCode is the single answer for every rejected login code — missing,
// expired, wrong, or burned — so a caller cannot tell which.
var ErrBadCode = apperr.Unauthorized("code is wrong or expired", "login code rejected")

// ConsumeCode checks code against hikerID's live login code and marks it used
// on success. A wrong code counts an attempt; the fifth burns the code. Every
// failure returns ErrBadCode.
func (s *Service) ConsumeCode(ctx context.Context, hikerID uuid.UUID, code string) error {

	tok, err := s.store.LiveByHiker(ctx, hikerID, PurposeLoginCode)
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); ok && se.Status == apperr.CodeNotFound {
			return ErrBadCode
		}
		return err
	}
	if tok.ExpiresAt.Before(s.now()) {
		return ErrBadCode
	}

	if !verifyCode(hikerID, code, tok.Hash) {
		if err = s.store.IncrementAttempts(ctx, tok.ID); err != nil {
			return err
		}
		if tok.Attempts+1 >= maxCodeAttempts {
			if err = s.store.MarkUsed(ctx, tok.ID); err != nil {
				return err
			}
		}
		return ErrBadCode
	}
	if err = s.store.MarkUsed(ctx, tok.ID); err != nil {
		return err
	}

	return nil
}

func hashCode(hikerID uuid.UUID, raw string) string {
	return secure.HashOTP(hikerID.String() + raw)
}

func verifyCode(hikerID uuid.UUID, code, hash string) bool {
	return secure.VerifyOTP(hikerID.String()+code, hash)
}
