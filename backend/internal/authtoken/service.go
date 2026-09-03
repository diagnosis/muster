package authtoken

import (
	"context"
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
}

// Service mints and consumes single-use auth tokens over a Storage
type Service struct {
	store Storage
}

// NewService returns Service
func NewService(store Storage) *Service {
	return &Service{store: store}
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
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		UsedAt:    nil,
	}
	if err = s.store.Save(ctx, t); err != nil {
		return "", err
	}

	return raw, nil
}

// Consume validates purpose, single-use, and expiry, then burns the token; a token never consumes twice (the contract the four tests pin)
func (s *Service) Consume(ctx context.Context, raw string, purpose Purpose) (uuid.UUID, error) {
	if !purpose.Valid() {
		return uuid.Nil, apperr.BadRequest("invalid purpose", "invalid purpose")
	}

	hash := secure.HashRefreshToken(raw)
	t, err := s.store.GetByHash(ctx, hash)
	if err != nil {
		return uuid.Nil, err
	}
	if t.Purpose != purpose {
		return uuid.Nil, apperr.Conflict("purpose mismatch", "purpose mismatch")
	}

	if t.UsedAt != nil {
		return uuid.Nil, apperr.Conflict("token is used", "token is used")
	}
	if t.ExpiresAt.Before(time.Now()) {
		return uuid.Nil, apperr.Conflict("token is expired", "token is expired")
	}

	if err = s.store.MarkUsed(ctx, t.ID); err != nil {
		return uuid.Nil, err
	}

	return t.HikerID, nil
}
