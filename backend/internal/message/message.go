package message

import (
	"time"

	"github.com/google/uuid"
)

// ConversationKind selects which shape a conversation row takes (see MESSAGING.md §2).
type ConversationKind string

// Conversation kinds: an outing's roster chat, or a two-person DM.
const (
	ConversationKindOuting ConversationKind = "outing"
	ConversationKindDM     ConversationKind = "dm"
)

// Valid reports whether k is a known conversation kind.
func (m ConversationKind) Valid() bool {
	switch m {
	case ConversationKindDM, ConversationKindOuting:
		return true
	}
	return false
}

// DMStatus is the DM state: pending → accepted | declined; accepted → declined (close);
// declined → accepted only by dm_declined_by (reopen).
type DMStatus string

// DM statuses.
const (
	DMStatusPending  DMStatus = "pending"
	DMStatusAccepted DMStatus = "accepted"
	DMStatusDeclined DMStatus = "declined"
)

// Valid reports whether d is a known DM status.
func (d DMStatus) Valid() bool {
	switch d {
	case DMStatusAccepted, DMStatusDeclined, DMStatusPending:
		return true
	}
	return false
}

// Conversation is one row of conversations. Outing rows set OutingID only;
// dm rows set DmA < DmB, DmInitiator, DmStatus and, when declined, DmDeclinedBy.
type Conversation struct {
	ID           uuid.UUID        `json:"id"`
	Kind         ConversationKind `json:"kind"`
	OutingID     *uuid.UUID       `json:"outing_id"`
	DmA          *uuid.UUID       `json:"dm_a"`
	DmB          *uuid.UUID       `json:"dm_b"`
	DmInitiator  *uuid.UUID       `json:"dm_initiator"`
	DmStatus     *DMStatus        `json:"dm_status"`
	DmDeclinedBy *uuid.UUID       `json:"dm_declined_by"`
	CreatedAt    time.Time        `json:"created_at"`
}

// Message is one row of messages. Seq is the DB-assigned ordering (D6);
// a non-nil DeletedAt marks a soft-deleted message.
type Message struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	HikerID        uuid.UUID  `json:"hiker_id"`
	Seq            int64      `json:"seq"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
}
