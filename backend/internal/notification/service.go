// backend/internal/notification/service.go

package notification

import (
	"context"

	"github.com/google/uuid"
)

// Storage persists notification events and drives the email outbox.
type Storage interface {
	Insert(ctx context.Context, e *Event) error
	ListUnsent(ctx context.Context, limit int) ([]*Unsent, error)
	MarkEmailed(ctx context.Context, id uuid.UUID) error
	ListForHiker(ctx context.Context, hikerID uuid.UUID, limit, offset int) ([]*Event, error)
	MarkRead(ctx context.Context, hikerID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, hikerID uuid.UUID) error
	UnreadCount(ctx context.Context, hikerID uuid.UUID) (int, error)
}
