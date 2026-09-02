package hiker

import (
	"context"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/authtoken"
	"github.com/google/uuid"
)

type tokenFakeStore struct {
	tokens map[string]*authtoken.Token
}

func newTokenFakeStore() *tokenFakeStore {
	return &tokenFakeStore{
		tokens: make(map[string]*authtoken.Token),
	}
}

func (f *tokenFakeStore) Save(ctx context.Context, t *authtoken.Token) error {
	f.tokens[t.Hash] = t
	return nil
}

func (f *tokenFakeStore) GetByHash(ctx context.Context, hash string) (*authtoken.Token, error) {
	if v, ok := f.tokens[hash]; ok {
		return v, nil
	}
	return nil, apperr.NotFound("token not found", "token not found")
}

func (f *tokenFakeStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	for _, t := range f.tokens {
		if t.ID == id {
			now := time.Now()
			t.UsedAt = &now
			return nil
		}
	}
	return apperr.NotFound("token not found", "token not found")
}

var _ authtoken.Storage = (*tokenFakeStore)(nil)
