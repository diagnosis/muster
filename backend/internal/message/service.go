package message

import (
	"context"
	"encoding/json"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/events"
	"github.com/google/uuid"
)

type Storage interface {
	GetConversation(ctx context.Context, id uuid.UUID) (*Conversation, error)
	IsMember(ctx context.Context, conversationID, hikerID uuid.UUID) (bool, error)
	InsertMessage(ctx context.Context, m *Message) error
	MemberIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
}
type Broadcaster interface {
	BroadcastToUser(hikerID uuid.UUID, e events.Event)
}

type Service struct {
	store       Storage
	broadcaster Broadcaster
}

func NewService(store Storage, broadcaster Broadcaster) *Service {
	return &Service{store: store, broadcaster: broadcaster}
}

type poke struct {
	ConversationID uuid.UUID        `json:"conversation_id"`
	Kind           ConversationKind `json:"kind"`
	OutingID       *uuid.UUID       `json:"outing_id,omitempty'"`
}

func (s *Service) PostMessage(ctx context.Context, conversationID, hikerID uuid.UUID, body string) error {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return err
	}

	member, err := s.store.IsMember(ctx, conversationID, hikerID)
	if err != nil {
		return err
	}
	if !member {
		return apperr.Forbidden("forbidden", "user not part of the roster", err)
	}
	message := &Message{
		ConversationID: conversationID,
		HikerID:        hikerID,
		Body:           body,
	}

	if err = s.store.InsertMessage(ctx, message); err != nil {
		return err
	}
	memberIDs, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	in := poke{
		ConversationID: conversationID,
		Kind:           conv.Kind,
		OutingID:       conv.OutingID,
	}

	data, err := json.Marshal(in)
	if err != nil {
		return apperr.Internal("internal error", "marshal poke", err)
	}


	for _, id := range memberIDs {
		s.broadcaster.BroadcastToUser(id, events.Event{
			Type: "message.created",
			Data: string(data),
		})
	}

	return nil
}
