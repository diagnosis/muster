package authtoken

import (
	"context"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	tokens map[uuid.UUID]*Token
}

func newFakeStore() *fakeStore {
	return &fakeStore{tokens: make(map[uuid.UUID]*Token)}
}

func (f *fakeStore) Save(ctx context.Context, token *Token) error {
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
