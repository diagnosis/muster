package postgres

import (
	"context"
	"encoding/json"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/notification"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationStore persists notification events in Postgres.
type NotificationStore struct {
	pool *pgxpool.Pool
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
