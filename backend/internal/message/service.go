package message

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/events"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
)

// Storage is what the messaging service needs from persistence. Fakes mirror
// the real store's mechanics (seq assignment, not-found errors), never policy.
type Storage interface {
	GetConversation(ctx context.Context, id uuid.UUID) (*Conversation, error)
	IsMember(ctx context.Context, conversationID, hikerID uuid.UUID) (bool, error)
	InsertMessage(ctx context.Context, m *Message, now time.Time) error
	MemberIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	OutingStatus(ctx context.Context, outingID uuid.UUID) (outing.Status, error)
	CountMessagesSince(ctx context.Context, conversationID, hikerID uuid.UUID, since time.Time) (int, error)
	ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*Message, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*Message, error)
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
	OutingHost(ctx context.Context, outingID uuid.UUID) (uuid.UUID, error)
}

const maxBodyRunes = 500
const maxPerMinute = 10

// Broadcaster is the single hub method the service uses; *events.Hub satisfies it.
type Broadcaster interface {
	BroadcastToUser(hikerID uuid.UUID, e events.Event)
}

// Service legislates messaging rules; stores execute them.
type Service struct {
	store       Storage
	broadcaster Broadcaster
	now         func() time.Time
}

// NewService returns a Service over the given store and broadcaster.
func NewService(store Storage, broadcaster Broadcaster) *Service {
	return &Service{store: store, broadcaster: broadcaster, now: time.Now}
}

type poke struct {
	ConversationID uuid.UUID        `json:"conversation_id"`
	Kind           ConversationKind `json:"kind"`
	OutingID       *uuid.UUID       `json:"outing_id,omitempty"`
}

// PostMessage inserts a message from hikerID into conversationID after verifying
// membership, then emits one message.created poke (no body) to every member.
func (s *Service) PostMessage(ctx context.Context, conversationID, hikerID uuid.UUID, body string) (*Message, error) {
	body = strings.TrimSpace(body)
	n := utf8.RuneCountInString(body)
	if n < 1 || n > maxBodyRunes {
		if n > maxBodyRunes {
			return nil, apperr.BadRequest("body cannot be longer than 500 characters", "500+ chars in body")
		}
		return nil, apperr.BadRequest("body cannot be empty", "empty body")
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	member, err := s.store.IsMember(ctx, conversationID, hikerID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, apperr.Forbidden("forbidden", "user not part of the roster", err)
	}
	if conv.OutingID != nil {
		var outingStatusErr error
		status, outingStatusErr := s.store.OutingStatus(ctx, *conv.OutingID)
		if outingStatusErr != nil {
			return nil, outingStatusErr
		}
		if !status.Valid() {
			return nil, apperr.Internal("internal error", "invalid outing status")
		}
		if status != outing.StatusOpen {
			msg := fmt.Sprintf("outing is %s", status)
			return nil, apperr.Forbidden(msg, msg)
		}
	}

	count, err := s.store.CountMessagesSince(ctx, conversationID, hikerID, s.now().Add(-time.Minute))
	if err != nil {
		return nil, err
	}
	if count >= maxPerMinute {
		return nil, apperr.TooManyRequests("too many requests", "too many requests")
	}

	message := &Message{
		ConversationID: conversationID,
		HikerID:        hikerID,
		Body:           body,
	}

	if err = s.store.InsertMessage(ctx, message, s.now()); err != nil {
		return nil, err
	}
	if err = s.broadcast(ctx, conv, "message.created"); err != nil {
		return nil, err
	}

	return message, nil
}

// ListMessages returns message history for members.
func (s *Service) ListMessages(ctx context.Context, convID, hikerID uuid.UUID) ([]*Message, error) {
	if _, err := s.store.GetConversation(ctx, convID); err != nil {
		return nil, err
	}
	ok, err := s.store.IsMember(ctx, convID, hikerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.Forbidden("forbidden", "user not part of the roster", err)
	}
	messages, err := s.store.ListMessages(ctx, convID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// DeleteMessage removes selected messages. Host or author
func (s *Service) DeleteMessage(ctx context.Context, messageID, hikerID uuid.UUID) error {
	message, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	conv, err := s.store.GetConversation(ctx, message.ConversationID)
	if err != nil {
		return err
	}
	allowed := message.HikerID == hikerID
	if !allowed && conv.OutingID != nil {
		hostID, convErr := s.store.OutingHost(ctx, *conv.OutingID)
		if convErr != nil {
			return convErr
		}
		allowed = hostID == hikerID
	}
	if !allowed {
		return apperr.Forbidden("forbidden", "author or host can delete")
	}

	if err = s.store.DeleteMessage(ctx, messageID); err != nil {
		return err
	}
	if err = s.broadcast(ctx, conv, "message.deleted"); err != nil {
		return err
	}

	return nil
}

func (s *Service) broadcast(ctx context.Context, conv *Conversation, eventType string) error {
	memberIDs, err := s.store.MemberIDs(ctx, conv.ID)
	if err != nil {
		return err
	}
	in := poke{
		ConversationID: conv.ID,
		Kind:           conv.Kind,
		OutingID:       conv.OutingID,
	}

	data, err := json.Marshal(in)
	if err != nil {
		return apperr.Internal("internal error", "marshal poke", err)
	}

	for _, id := range memberIDs {
		s.broadcaster.BroadcastToUser(id, events.Event{
			Type: eventType,
			Data: string(data),
		})
	}
	return nil
}
