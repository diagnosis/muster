// backend/internal/notification/service.go

package notification

import "context"

// Storage persists and retrieves notification events.
type Storage interface {
	Insert(ctx context.Context, e *Event) error
}
