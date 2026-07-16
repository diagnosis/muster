package postgres

import (
	"context"
	"errors"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/muster/internal/hiker"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HikerStore implements hiker.Storage backed by Postgres.
type HikerStore struct {
	pool *pgxpool.Pool
}

var _ hiker.Storage = (*HikerStore)(nil)

// NewHikerStore returns a HikerStore using the given connection pool.
func NewHikerStore(pool *pgxpool.Pool) *HikerStore {
	return &HikerStore{pool: pool}
}

// CreateHiker inserts a new hiker, scanning the DB-generated timestamps
// back into h. A duplicate email translates to apperr.EmailExists.
func (s *HikerStore) CreateHiker(ctx context.Context, h *hiker.Hiker) error {
	q := `
	INSERT INTO hikers (id,email, password_hash, name, experience, home_area, bio, gender)
	VALUES ($1, $2, $3, $4 ,$5, $6, $7, $8)
	RETURNING created_at, updated_at
`
	err := s.pool.QueryRow(ctx, q,
		h.ID, h.Email, h.PasswordHash, h.Name, h.Experience, h.HomeArea, h.Bio, h.Gender,
	).Scan(&h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return apperr.EmailExists("email already registered", "unique violation on hikers.email", err)
		}
		return apperr.Database("could not create hiker", "insert hikers failed", err)
	}
	return nil
}

// GetHikerByEmail returns the hiker with the given email, excluding
// soft-deleted accounts. A missing row translates to apperr.NotFound.
func (s *HikerStore) GetHikerByEmail(ctx context.Context, email string) (*hiker.Hiker, error) {
	q := `
    SELECT id, email, password_hash, name, experience, home_area, bio, gender,
           created_at, updated_at, deleted_at
    FROM hikers
    WHERE email = $1 AND deleted_at IS NULL`

	h, err := scanHiker(s.pool.QueryRow(ctx, q, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("hiker not found", "no row for email")
		}
		return nil, apperr.Database("could not load hiker", "select hikers by email failed", err)
	}
	return h, nil
}

// GetHikerByID returns the hiker with the given id, excluding
// soft-deleted accounts. A missing row translates to apperr.NotFound.
func (s *HikerStore) GetHikerByID(ctx context.Context, id uuid.UUID) (*hiker.Hiker, error) {
	q := `
    SELECT id, email, password_hash, name, experience, home_area, bio, gender,
           created_at, updated_at, deleted_at
    FROM hikers
    WHERE id = $1 AND deleted_at IS NULL`

	h, err := scanHiker(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("hiker not found", "no row for id")
		}
		return nil, apperr.Database("could not load hiker", "select hikers by id failed", err)
	}
	return h, nil
}

// UpdateHiker updates the mutable profile fields of an existing,
// non-deleted hiker. A missing row translates to apperr.NotFound.
func (s *HikerStore) UpdateHiker(ctx context.Context, h *hiker.Hiker) error {
	q := `
		UPDATE hikers
		SET name = $2, experience = $3, home_area = $4, bio = $5, gender = $6,
    	updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
`
	err := s.pool.QueryRow(ctx, q,
		h.ID, h.Name, h.Experience, h.HomeArea, h.Bio, h.Gender,
	).Scan(&h.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("hiker not found", "no row for id")
		}
		return apperr.Database("could not update hiker", "update hikers failed", err)
	}
	return nil
}

// SaveRefreshToken stores a hashed refresh-token record. A duplicate
// (hiker, platform) pair translates to apperr.Conflict
func (s *HikerStore) SaveRefreshToken(ctx context.Context, t *hiker.RefreshToken) error {
	q := `
		INSERT INTO user_refresh_tokens (id, hiker_id, token_hash, platform, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at	
`
	err := s.pool.QueryRow(ctx, q,
		t.ID, t.HikerID, t.TokenHash, t.Platform, t.ExpiresAt,
	).Scan(&t.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return apperr.Conflict("session already exists", "unique violation on hiker+platform", err)
		}
		return apperr.Database("could not save refresh token", "save refresh token failed", err)
	}
	return nil
}

// GetRefreshTokenByHash returns the refresh-token record matching the
// given hash. A missing row translates to apperr.NotFound.
func (s *HikerStore) GetRefreshTokenByHash(ctx context.Context, hash string) (*hiker.RefreshToken, error) {
	q := `
		SELECT id, hiker_id, token_hash, platform, expires_at, created_at
		FROM user_refresh_tokens
		WHERE token_hash = $1
`
	t := &hiker.RefreshToken{}

	err := s.pool.QueryRow(ctx, q, hash).Scan(&t.ID, &t.HikerID, &t.TokenHash, &t.Platform, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("token not found", "no row for hash")
		}
		return nil, apperr.Database("could not load token", "select token by hash failed", err)
	}
	return t, nil
}

// DeleteRefreshTokens removes all refresh tokens for the hiker on the
// given platform. Deleting zero rows is not an error.
func (s *HikerStore) DeleteRefreshTokens(ctx context.Context, hikerID uuid.UUID, platform hiker.Platform) error {
	q := `DELETE FROM user_refresh_tokens WHERE hiker_id = $1 AND platform = $2`
	_, err := s.pool.Exec(ctx, q, hikerID, platform)
	if err != nil {
		return apperr.Database("could not delete refresh tokens", "delete user_refresh_tokens failed", err)
	}
	return nil
}

// adding helper methods
func scanHiker(row pgx.Row) (*hiker.Hiker, error) {
	h := &hiker.Hiker{}
	err := row.Scan(
		&h.ID, &h.Email, &h.PasswordHash, &h.Name, &h.Experience,
		&h.HomeArea, &h.Bio, &h.Gender,
		&h.CreatedAt, &h.UpdatedAt, &h.DeletedAt,
	)
	return h, err
}
