package hiker

import (
	"context"
	"sort"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/authtoken"
	"github.com/google/uuid"
)

type tokenFakeStore struct {
	tokens  map[string]*authtoken.Token
	saveErr error
}

func newTokenFakeStore() *tokenFakeStore {
	return &tokenFakeStore{
		tokens: make(map[string]*authtoken.Token),
	}
}

func (f *tokenFakeStore) Save(ctx context.Context, t *authtoken.Token) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if _, ok := f.tokens[t.Hash]; ok {
		return apperr.Conflict("token hash already stored", "token hash exists")
	}
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
func (f *tokenFakeStore) LiveByHiker(ctx context.Context, hikerID uuid.UUID, purpose authtoken.Purpose) (*authtoken.Token, error) {
	liveToks := []*authtoken.Token{}
	for _, t := range f.tokens {
		if t.HikerID == hikerID && t.Purpose == purpose && t.UsedAt == nil {
			liveToks = append(liveToks, t)
		}
	}
	if len(liveToks) == 0 {
		return nil, apperr.NotFound("live token not found", "live token not found")
	}
	sort.Slice(liveToks, func(i, j int) bool {
		return liveToks[i].CreatedAt.After(liveToks[j].CreatedAt)
	})
	cp := *liveToks[0]

	return &cp, nil

}

func (f *tokenFakeStore) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	for _, tok := range f.tokens {
		if tok.ID == id {
			f.tokens[tok.Hash].Attempts++
			return nil
		}
	}

	return apperr.NotFound("token not found", "token not found")

}

var _ authtoken.Storage = (*tokenFakeStore)(nil)
