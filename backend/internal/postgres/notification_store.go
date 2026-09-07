package postgres

import (
	"context"

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
	return apperr.Database("not implemented", "yet to implement")
}

// NewNotificationStore returns a NotificationStore backed by pool.
func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{
		pool: pool,
	}
}

var _ notification.Storage = (*NotificationStore)(nil)
