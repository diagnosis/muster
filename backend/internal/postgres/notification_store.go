package postgres

import (
	"context"
	"encoding/json"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/muster/internal/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationStore persists notification events in Postgres.
type NotificationStore struct {
	pool *pgxpool.Pool
}

// ListUnsent returns up to limit notification events whose email has not
// been sent (emailed_at IS NULL), oldest first, each joined with its
// recipient's address. The dispatcher drains these.
func (s *NotificationStore) ListUnsent(ctx context.Context, limit int) ([]*notification.Unsent, error) {
	q := `
		SELECT n.id, n.hiker_id, n.kind, n.payload, n.created_at, n.read_at, n.emailed_at, h.email 
		FROM notification_events n JOIN hikers h on h.id = n.hiker_id
		WHERE n.emailed_at is NULL ORDER BY n.created_at LIMIT $1
`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, apperr.Database("failed to list unsent events", "select unsent events failed", err)
	}
	defer rows.Close()

	unsents := []*notification.Unsent{}
	for rows.Next() {
		u := notification.Unsent{}
		if err = rows.Scan(&u.Event.ID, &u.Event.HikerID, &u.Event.Kind, &u.Event.Payload,
			&u.Event.CreatedAt, &u.Event.ReadAt, &u.Event.EmailedAt, &u.Email); err != nil {
			return nil, apperr.Database("failed to list unsent events", "scan upcoming event failed", err)
		}
		unsents = append(unsents, &u)
	}
	if err = rows.Err(); err != nil {
		return nil, apperr.Database("failed to list unsent events", "iterate upcoming event failed", err)
	}
	return unsents, nil
}

// MarkEmailed stamps emailed_at on the event, recording that its email was
// sent. A missing row is tolerated (already sent or deleted) — never an error,
// so the dispatcher won't re-send on a benign miss.
func (s *NotificationStore) MarkEmailed(ctx context.Context, id uuid.UUID) error {
	q := "UPDATE notification_events SET emailed_at = now() WHERE id = $1"
	ct, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return apperr.Database("failed to mark as emailed", "set emailed now failed", err)
	}
	if ct.RowsAffected() == 0 {
		logger.Warn(ctx, "no email found to set as mark emailed")
	}
	return nil
}

// Insert stores a new notification event; id and created_at default in the table.
func (s *NotificationStore) Insert(ctx context.Context, e *notification.Event) error {
	q := `
	INSERT INTO notification_events (hiker_id, kind, payload)
	VALUES ($1, $2, $3)
`
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return apperr.Internal("cannot store notification", "payload marshal failed", err)
	}
	_, err = s.pool.Exec(ctx, q, e.HikerID, string(e.Kind), payload)
	if err != nil {
		return apperr.Database("cannot store notification", "failed to insert notification", err)
	}
	return nil

}

// NewNotificationStore returns a NotificationStore backed by pool.
func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{
		pool: pool,
	}
}

var _ notification.Storage = (*NotificationStore)(nil)
