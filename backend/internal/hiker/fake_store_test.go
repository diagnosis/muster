package hiker

import (
	"context"
	"strings"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	hikers map[uuid.UUID]*Hiker
	rt     map[string]*RefreshToken
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		hikers: map[uuid.UUID]*Hiker{},
		rt:     map[string]*RefreshToken{},
	}
}

func (f *fakeStore) CreateHiker(ctx context.Context, h *Hiker) error {
	for _, hiker := range f.hikers {
		if strings.EqualFold(hiker.Email, h.Email) {
			return apperr.EmailExists("email exists", "duplicate email violation")
		}
	}

	f.hikers[h.ID] = h
	return nil
}

func (f *fakeStore) GetHikerByEmail(ctx context.Context, email string) (*Hiker, error) {
	for _, hiker := range f.hikers {
		if strings.EqualFold(hiker.Email, email) {
			return hiker, nil
		}
	}
	return nil, apperr.NotFound("hiker not found", "hiker not found")
}

func (f *fakeStore) GetHikerByID(ctx context.Context, id uuid.UUID) (*Hiker, error) {
	if _, ok := f.hikers[id]; ok {
		return f.hikers[id], nil
	}
	return nil, apperr.NotFound("hiker not found", "hiker not found")
}

func (f *fakeStore) UpdateHiker(ctx context.Context, h *Hiker) error {
	if _, ok := f.hikers[h.ID]; ok {
		f.hikers[h.ID] = h
	} else {
		return apperr.NotFound("hiker not found", "hiker not found")
	}
	return nil
}

func (f *fakeStore) SaveRefreshToken(ctx context.Context, t *RefreshToken) error {

	f.rt[t.TokenHash] = t
	return nil
}

func (f *fakeStore) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	if v, ok := f.rt[hash]; ok {
		return v, nil
	}
	return nil, apperr.NotFound("refresh token not found", "refresh token not found")
}

func (f *fakeStore) DeleteRefreshTokens(ctx context.Context, hikerID uuid.UUID, platform Platform) error {

	for _, token := range f.rt {
		if token.HikerID == hikerID && token.Platform == platform {
			delete(f.rt, token.TokenHash)
		}
	}
	return nil
}

var _ Storage = (*fakeStore)(nil)
