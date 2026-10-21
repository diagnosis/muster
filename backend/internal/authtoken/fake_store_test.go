package authtoken

import (
	"context"
	"sort"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	tokens map[uuid.UUID]*Token
}

func (f *fakeStore) LiveByHiker(ctx context.Context, hikerID uuid.UUID, purpose Purpose) (*Token, error) {
	liveToks := []*Token{}
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

func (f *fakeStore) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	_, ok := f.tokens[id]
	if !ok {
		return apperr.NotFound("token not found", "token not found")
	}
	f.tokens[id].Attempts++
	return nil
}

func newFakeStore() *fakeStore {
	return &fakeStore{tokens: make(map[uuid.UUID]*Token)}
}

func (f *fakeStore) Save(ctx context.Context, token *Token) error {
	for _, tok := range f.tokens {
		if tok.Hash == token.Hash {
			return apperr.Conflict("token hash already stored", "token hash exists")
		}
	}
	f.tokens[token.ID] = token
	return nil
}

func (f *fakeStore) GetByHash(ctx context.Context, hash string) (*Token, error) {
	for _, t := range f.tokens {
		if t.Hash == hash {
			return t, nil
		}
	}
	return nil, apperr.NotFound("token not found", "token not found")
}

func (f *fakeStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	if t, ok := f.tokens[id]; ok {
		ti := time.Now()
		t.UsedAt = &ti
	} else {
		return apperr.NotFound("token not found", "token not found")
	}
	return nil
}

var _ Storage = (*fakeStore)(nil)
