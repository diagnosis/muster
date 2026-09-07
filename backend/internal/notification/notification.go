// backend/internal/notification/notification.go

package notification

import (
	"time"

	"github.com/google/uuid"
)

// Kind identifies what a notification event announces. Values byte-match
// the notification_events.kind CHECK constraint — append-only, two homes.
type Kind string

// Notification kinds, one per state transition that affects someone who
// didn't cause it (see backend/docs/notifications-v1.md).
const (
	KindJoinRequestCreated   Kind = "join_request_created"
	KindJoinRequestApproved  Kind = "join_request_approved"
	KindJoinRequestDeclined  Kind = "join_request_declined"
	KindJoinRequestWithdrawn Kind = "join_request_withdrawn"
	KindMemberRemoved        Kind = "member_removed"
	KindOutingCancelled      Kind = "outing_cancelled"
	KindOutingUpdated        Kind = "outing_updated"
)

// Valid reports whether k is a known notification kind.
func (k Kind) Valid() bool {
	switch k {
	case KindJoinRequestCreated, KindJoinRequestApproved, KindJoinRequestDeclined, KindJoinRequestWithdrawn,
		KindMemberRemoved, KindOutingCancelled, KindOutingUpdated:
		return true
	}
	return false
}

// Event is one notification to one recipient: a bell item and, until
// EmailedAt is set, a pending email (outbox pattern).
type Event struct {
	ID        uuid.UUID      `json:"id"`
	HikerID   uuid.UUID      `json:"hiker_id"`
	Kind      Kind           `json:"kind"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
	ReadAt    *time.Time     `json:"read_at"`
	EmailedAt *time.Time     `json:"-"`
}
