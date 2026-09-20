package message

import (
	"time"

	"github.com/google/uuid"
)



type ConversationKind string
const (
	ConversationKindOuting  ConversationKind = "outing"
	ConversationKindDM ConversationKind = "dm"
)
func (m ConversationKind) Valid() bool{
	switch m {
	case ConversationKindDM, ConversationKindOuting: return true
	}
	return false
}
type DMStatus string
const (
	DMStatusPending DMStatus = "pending"
	DMStatusAccepted DMStatus = "accepted"
	DMStatusDeclined DMStatus = "declined"
)
func (d DMStatus) Valid() bool{
	switch d {
	case DMStatusAccepted, DMStatusDeclined, DMStatusPending:return true
	}
	return false
}

type Conversation struct {
	ID uuid.UUID `json:"id"`
	Kind ConversationKind `json:"kind"`
	OutingID *uuid.UUID `json:"outing_id"`
	DmA *uuid.UUID `json:"dm_a"`
	DmB *uuid.UUID `json:"dm_b"`
	DmInitiator *uuid.UUID `json:"dm_initiator"`
	DmStatus *DMStatus `json:"dm_status"`
	DmDeclinedBy *uuid.UUID `json:"dm_declined_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	HikerID uuid.UUID `json:"hiker_id"`
	Seq int64 `json:"seq"`
	Body string `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}