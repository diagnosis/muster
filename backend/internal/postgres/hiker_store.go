package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
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
           created_at, updated_at, deleted_at, verified_at
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
           created_at, updated_at, deleted_at, verified_at
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

// SetVerified sets verified_at to now.
// zero rows: unknown, already-verified, or deleted — all NotFound for v1.
func (s *HikerStore) SetVerified(ctx context.Context, hikerID uuid.UUID) error {
	q := `
		UPDATE hikers
		SET verified_at = now(), updated_at = now()
		WHERE id = $1 AND verified_at IS NULL AND deleted_at IS NULL
`
	ct, err := s.pool.Exec(ctx, q, hikerID)
	if err != nil {
		return apperr.Database("cannot verify email", "failed to verify email", err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("row not found", "row not found")
	}
	return nil
}

// SetPassword sets new password
func (s *HikerStore) SetPassword(ctx context.Context, hikerID uuid.UUID, hash string) error {
	q := `
		UPDATE hikers
		SET password_hash = $2, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
`
	ct, err := s.pool.Exec(ctx, q, hikerID, hash)
	if err != nil {
		return apperr.Database("cannot set password", "failed to set password", err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("row not found", "row not found")
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

// RotateRefreshToken swaps oldHash for newHash on the row that holds it, in one
// statement, recording oldHash as previous_hash and stamping rotated_at. The
// row must match platform and be unexpired; those are WHERE conditions, not
// checks after, so a row is never rotated for a client that will be refused.
// NotFound when no row matches.
func (s *HikerStore) RotateRefreshToken(ctx context.Context, oldHash, newHash string, platform hiker.Platform, expiresAt time.Time) (*hiker.RefreshToken, error) {
	q := `
		UPDATE user_refresh_tokens 
		SET token_hash = $2, previous_hash = token_hash,
		    rotated_at = now(), expires_at = $4
		WHERE token_hash = $1 AND platform = $3 AND expires_at > now()
		RETURNING id, hiker_id, token_hash, platform, expires_at, created_at, previous_hash, rotated_at;
`
	var tok hiker.RefreshToken
	row := s.pool.QueryRow(ctx, q, oldHash, newHash, platform, expiresAt)
	if err := row.Scan(
		&tok.ID,
		&tok.HikerID,
		&tok.TokenHash,
		&tok.Platform,
		&tok.ExpiresAt,
		&tok.CreatedAt,
		&tok.PreviousHash,
		&tok.RotatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("not found", "not found")
		}
		return nil, apperr.Database("failed to get new token", "failed to rotate token", err)
	}
	return &tok, nil
}

// RotateFromPrevious is the grace path (design R3): the row whose previous_hash
// is previousHash and whose rotated_at is within grace is rotated again — for a
// client that never received the last rotation's cookie and retried with the
// token it still held. Same platform and expiry conditions as RotateRefreshToken.
// NotFound when no row matches, including when the window has passed.
func (s *HikerStore) RotateFromPrevious(ctx context.Context, previousHash, newHash string, platform hiker.Platform, expiresAt time.Time, since time.Time) (*hiker.RefreshToken, error) {
	q := `
		UPDATE user_refresh_tokens 
		SET token_hash = $2, previous_hash = token_hash, rotated_at = now(),
		    expires_at = $4
		WHERE previous_hash = $1 AND platform = $3 AND rotated_at > $5 AND expires_at > now()
		RETURNING id, hiker_id, token_hash, platform, expires_at, created_at, previous_hash, rotated_at;
`
	var tok hiker.RefreshToken
	row := s.pool.QueryRow(ctx, q, previousHash, newHash, platform, expiresAt, since)
	if err := row.Scan(
		&tok.ID,
		&tok.HikerID,
		&tok.TokenHash,
		&tok.Platform,
		&tok.ExpiresAt,
		&tok.CreatedAt,
		&tok.PreviousHash,
		&tok.RotatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("not found", "not found")
		}
		return nil, apperr.Database("failed to get new token", "failed to rotate token", err)
	}
	return &tok, nil
}

// DeleteRefreshToken revokes one session: the row holding hash. Deleting a
// token that isn't there is not an error, so a second logout still succeeds.
func (s *HikerStore) DeleteRefreshToken(ctx context.Context, hash string) error {
	q := `DELETE FROM user_refresh_tokens WHERE token_hash = $1`
	if _, err := s.pool.Exec(ctx, q, hash); err != nil {
		return apperr.Database("could not sign out", "delete refresh token failed", err)
	}
	return nil
}

// DeleteAllRefreshTokens revokes every session the hiker holds, on every
// device. Used by password reset: no session that knew the old password survives.
func (s *HikerStore) DeleteAllRefreshTokens(ctx context.Context, hikerID uuid.UUID) error {
	q := `DELETE FROM user_refresh_tokens WHERE hiker_id = $1`
	if _, err := s.pool.Exec(ctx, q, hikerID); err != nil {
		return apperr.Database("could not sign out everywhere", "delete all refresh tokens failed", err)
	}
	return nil
}

// OutingCounts returns how many past outings the hiker hosted (not cancelled)
// and how many past outings they were accepted on. "Past" mirrors
// outing.effectiveEnd: ends_at when set, else starts_at + 12 hours — change both or neither.
func (s *HikerStore) OutingCounts(ctx context.Context, hikerID uuid.UUID) (hiker.OutingCounts, error) {
	q := `
		SELECT
		  (SELECT COUNT(*) FROM outings o
			WHERE o.host_id = $1 AND o.status <> 'cancelled'
			  AND COALESCE(o.ends_at, o.starts_at + interval '12 hours') <= now()) AS hosted,
		  (SELECT COUNT(*) FROM join_requests jr JOIN outings o ON o.id = jr.outing_id
			WHERE jr.hiker_id = $1 AND jr.status = 'accepted' AND o.status <> 'cancelled'
			  AND COALESCE(o.ends_at, o.starts_at + interval '12 hours') <= now()) AS went
`
	oc := hiker.OutingCounts{}
	if err := s.pool.QueryRow(ctx, q, hikerID).Scan(&oc.HostedCount, &oc.WentCount); err != nil {
		return hiker.OutingCounts{}, apperr.Database("failed to get counts", "failed to scan outings", err)
	}
	return oc, nil
}

// adding helper methods
func scanHiker(row pgx.Row) (*hiker.Hiker, error) {
	h := &hiker.Hiker{}
	err := row.Scan(
		&h.ID, &h.Email, &h.PasswordHash, &h.Name, &h.Experience,
		&h.HomeArea, &h.Bio, &h.Gender,
		&h.CreatedAt, &h.UpdatedAt, &h.DeletedAt, &h.VerifiedAt,
	)
	return h, err
}
