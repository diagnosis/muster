package hiker

import (
	"context"
	"strings"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	hikers       map[uuid.UUID]*Hiker
	rt           map[string]*RefreshToken
	outingCounts map[uuid.UUID]OutingCounts
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		hikers:       map[uuid.UUID]*Hiker{},
		rt:           map[string]*RefreshToken{},
		outingCounts: make(map[uuid.UUID]OutingCounts),
	}
}

func (f *fakeStore) RotateRefreshToken(ctx context.Context, oldHash, newHash string, platform Platform, expiresAt time.Time) (*RefreshToken, error) {
	now := time.Now()
	token, ok := f.rt[oldHash]
	if !ok {
		return nil, apperr.NotFound("not found", "not found")
	}
	if token.Platform != platform || token.ExpiresAt.Before(now) {
		return nil, apperr.NotFound("not found", "not found")
	}

	token.TokenHash = newHash
	token.PreviousHash = &oldHash
	token.RotatedAt = &now
	token.ExpiresAt = expiresAt
	f.rt[newHash] = token
	delete(f.rt, oldHash)
	return token, nil
}

func (f *fakeStore) RotateFromPrevious(ctx context.Context, previousHash, newHash string, platform Platform, expiresAt time.Time, since time.Time) (*RefreshToken, error) {

	now := time.Now()
	var tok *RefreshToken
	for _, token := range f.rt {
		if token.PreviousHash != nil && *token.PreviousHash == previousHash {
			tok = token
			break
		}

	}
	if tok == nil {
		return nil, apperr.NotFound("not found", "not found")
	}
	if tok.Platform != platform || tok.ExpiresAt.Before(now) {
		return nil, apperr.NotFound("not found", "not found")
	}
	if tok.RotatedAt == nil || !tok.RotatedAt.After(since) {
		return nil, apperr.NotFound("not found", "not found")
	}
	tokHash := tok.TokenHash
	tok.PreviousHash = &tokHash
	tok.TokenHash = newHash
	tok.RotatedAt = &now
	tok.ExpiresAt = expiresAt
	f.rt[newHash] = tok
	delete(f.rt, tokHash)

	return tok, nil
}

func (f *fakeStore) DeleteRefreshToken(ctx context.Context, hash string) error {
	delete(f.rt, hash)
	return nil
}

func (f *fakeStore) DeleteAllRefreshTokens(ctx context.Context, hikerID uuid.UUID) error {
	for _, token := range f.rt {
		if token.HikerID == hikerID {
			delete(f.rt, token.TokenHash)
		}
	}
	return nil
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
func (f *fakeStore) SetVerified(ctx context.Context, hikerID uuid.UUID) error {
	if _, ok := f.hikers[hikerID]; ok {
		now := time.Now()
		f.hikers[hikerID].VerifiedAt = &now
	} else {
		return apperr.NotFound("not found", "not founf")
	}
	return nil
}
func (f *fakeStore) SetPassword(ctx context.Context, hikerID uuid.UUID, hash string) error {
	h, ok := f.hikers[hikerID]
	if !ok {
		return apperr.NotFound("row not found", "row not found")
	}
	h.PasswordHash = hash
	return nil
}

func (f *fakeStore) OutingCounts(ctx context.Context, hikerID uuid.UUID) (OutingCounts, error) {
	v, ok := f.outingCounts[hikerID]
	if !ok {
		return OutingCounts{}, nil
	}
	oc := OutingCounts{
		HostedCount: v.HostedCount,
		WentCount:   v.WentCount,
	}

	return oc, nil
}

var _ Storage = (*fakeStore)(nil)
