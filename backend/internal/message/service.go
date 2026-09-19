package message

import (
	"context"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type Storage interface {
	GetConversation(ctx context.Context, id uuid.UUID)(*Conversation, error)
	IsMember(ctx context.Context, conversationID, hikerID uuid.UUID)(bool, error)
}

type Service struct {
	store Storage
}
func NewService(store Storage)*Service{
	return &Service{store: store}
}

func (s *Service) PostMessage(ctx context.Context, conversationID, hikerID uuid.UUID, body string)error{
	_, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return err
	}

	member,err := s.store.IsMember(ctx, conversationID, hikerID)
	if err != nil {
		return err
	}
	if !member{
		return apperr.Forbidden("forbidden", "user not part of the roster", err)
	}

	return apperr.Internal("not implemented", "not implemented")
}
