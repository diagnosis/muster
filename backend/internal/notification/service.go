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
}
