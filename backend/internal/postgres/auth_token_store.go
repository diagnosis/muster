package postgres

import (
	"context"
	"errors"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/authtoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthTokenStore implements authtoken.Storage backed by postgres
type AuthTokenStore struct {
	pool *pgxpool.Pool
}

// NewAuthTokenStore returns AuthTokenStore
func NewAuthTokenStore(pool *pgxpool.Pool) *AuthTokenStore {
	return &AuthTokenStore{pool: pool}
}

var _ authtoken.Storage = (*AuthTokenStore)(nil)

// Save saves token to database
func (s *AuthTokenStore) Save(ctx context.Context, token *authtoken.Token) error {
	q := `
	INSERT INTO auth_tokens (id, hiker_id, token_hash, purpose, created_at, expires_at)
	VALUES ($1, $2, $3, $4, $5, $6)
`
	if _, err := s.pool.Exec(ctx, q, token.ID, token.HikerID, token.Hash, token.Purpose, token.CreatedAt, token.ExpiresAt); err != nil {
		return apperr.Database("failed to save token", "insert token failed", err)
	}
	return nil
}

// GetByHash returns auth token from database if it exits
func (s *AuthTokenStore) GetByHash(ctx context.Context, hash string) (*authtoken.Token, error) {
	q := `
	SELECT id, hiker_id, token_hash, purpose, created_at, expires_at, used_at
	FROM auth_tokens
	WHERE token_hash=$1	
`
	t, err := scanToken(s.pool.QueryRow(ctx, q, hash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("token not found", "token not found")
		}
		return nil, apperr.Database("failed to get token by hash", "token scan row error", err)
	}
	return t, nil
}

// MarkUsed marks auth token as used when consumed.
func (s *AuthTokenStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	q := `
		UPDATE auth_tokens 
		SET used_at = now()
		    WHERE id = $1
`
	ct, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return apperr.Database("failed to mark as used", "update token as used failed", err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("token not found", "token not found")
	}
	return nil
}

func scanToken(row pgx.Row) (*authtoken.Token, error) {
	t := &authtoken.Token{}
	if err := row.Scan(&t.ID, &t.HikerID, &t.Hash, &t.Purpose, &t.CreatedAt, &t.ExpiresAt, &t.UsedAt); err != nil {
		return nil, err
	}
	return t, nil

}
