package message

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/muster/internal/events"
	"github.com/diagnosis/muster/internal/notification"
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
	CanDM(ctx context.Context, initiator, other uuid.UUID) (bool, error)

	GetOrCreateDM(ctx context.Context, lo, hi, initiator uuid.UUID) (*Conversation, bool, error)
	UpdateDMStatus(ctx context.Context, convID uuid.UUID, status DMStatus, declinedBy *uuid.UUID) error
	ListConversations(ctx context.Context, hikerID uuid.UUID) ([]*ConversationSummary, error)
	GetConversationView(ctx context.Context, convID uuid.UUID) (*ConversationView, error)
	HikerName(ctx context.Context, hikerId uuid.UUID) (string, error)
}

const maxBodyRunes = 500
const maxPerMinute = 10

// Service legislates messaging rules; stores execute them.
type Service struct {
	store         Storage
	broadcaster   events.Broadcaster
	now           func() time.Time
	notifications notification.Storage
}

// NewService returns a Service over the given store and broadcaster.
func NewService(store Storage, broadcaster events.Broadcaster, notifications notification.Storage) *Service {
	return &Service{store: store, broadcaster: broadcaster, now: time.Now, notifications: notifications}
}

type poke struct {
	ConversationID   uuid.UUID        `json:"conversation_id"`
	ConversationKind ConversationKind `json:"conversation_kind"`
	OutingID         *uuid.UUID       `json:"outing_id,omitempty"`
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

	if conv.Kind == ConversationKindDM {
		switch *conv.DmStatus {
		case DMStatusAccepted:
			// fall through to rate limit
		case DMStatusDeclined:
			return nil, apperr.Forbidden("conversation is closed", "dm declined")
		case DMStatusPending:
			if conv.DmInitiator == nil || hikerID != *conv.DmInitiator {
				return nil, apperr.Forbidden("waiting for the other hiker to accept", "dm pending, non-initiator")
			}
			n1, verr := s.store.CountMessagesSince(ctx, conversationID, hikerID, time.Time{})
			if verr != nil {
				return nil, verr
			}
			if n1 > 0 {
				return nil, apperr.Forbidden("one message until they accept", "dm pending, opening message already sent")
			}
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
		ConversationID:   conv.ID,
		ConversationKind: conv.Kind,
		OutingID:         conv.OutingID,
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

// StartDM opens (or returns) the DM between initiator and other. Requires a shared
// outing, or initiator hosting an outing other has requested. New DMs start pending
// and poke both parties with dm.requested; an existing DM is returned silently.
func (s *Service) StartDM(ctx context.Context, h1, h2 uuid.UUID) (*Conversation, error) {
	if h1 == h2 {
		return nil, apperr.BadRequest("cannot dm yourself", "user cannot dm themselves")
	}
	lo, hi := h1, h2
	if h2.String() < h1.String() {
		lo, hi = h2, h1
	}
	ok, err := s.store.CanDM(ctx, h1, h2)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.Forbidden("dm is not allowed", "dm is not allowed")
	}
	conv, created, err := s.store.GetOrCreateDM(ctx, lo, hi, h1)
	if err != nil {
		return nil, err
	}
	if created {
		if err = s.broadcast(ctx, conv, "dm.requested"); err != nil {
			return nil, err
		}
		s.notify(ctx, h1, h2, conv, notification.KindDMRequested)
	}
	return conv, nil

}

// AcceptDM moves a pending DM to accepted. Only the non-initiating party may accept.
func (s *Service) AcceptDM(ctx context.Context, convID, actor uuid.UUID) error {
	conv, err := s.store.GetConversation(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Kind != ConversationKindDM {
		return apperr.BadRequest("this conversation isn't a direct message", "accept on non-dm conversation")
	}
	if actor != *conv.DmA && actor != *conv.DmB {
		return apperr.Forbidden("you're not part of this conversation", "accept by non-party")
	}
	if actor == *conv.DmInitiator {
		return apperr.Forbidden("only the other hiker can accept", "accept by initiator")
	}
	if *conv.DmStatus != DMStatusPending {
		return apperr.Conflict("this conversation isn't waiting for acceptance", fmt.Sprintf("accept on status %s", *conv.DmStatus))
	}
	if err = s.store.UpdateDMStatus(ctx, convID, DMStatusAccepted, nil); err != nil {
		return err
	}
	s.notify(ctx, actor, *conv.DmInitiator, conv, notification.KindDMAccepted)
	return s.broadcast(ctx, conv, "dm.accepted")
}

// DeclineDM moves a pending or accepted DM to declined and records the actor as
// dm_declined_by; either party may decline (this is also "close").
func (s *Service) DeclineDM(ctx context.Context, convID, actor uuid.UUID) error {
	conv, err := s.store.GetConversation(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Kind != ConversationKindDM {
		return apperr.BadRequest("this conversation isn't a direct message", "decline on non-dm conversation")
	}
	if actor != *conv.DmA && actor != *conv.DmB {
		return apperr.Forbidden("you're not part of this conversation", "decline by non-party")
	}
	if *conv.DmStatus == DMStatusDeclined {
		return apperr.Conflict("this conversation isn't waiting for decline", fmt.Sprintf("decline on status %s", *conv.DmStatus))
	}
	err = s.store.UpdateDMStatus(ctx, convID, DMStatusDeclined, &actor)
	if err != nil {
		return err
	}
	return s.broadcast(ctx, conv, "dm.declined")

}

// ReopenDM moves a declined DM back to accepted. Only dm_declined_by may reopen;
// the column is cleared on success.
func (s *Service) ReopenDM(ctx context.Context, convID, actor uuid.UUID) error {
	conv, err := s.store.GetConversation(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Kind != ConversationKindDM {
		return apperr.BadRequest("this conversation isn't a direct message", "reopen on non-dm conversation")
	}
	if actor != *conv.DmA && actor != *conv.DmB {
		return apperr.Forbidden("you're not part of this conversation", "reopen by non-party")
	}
	if *conv.DmStatus != DMStatusDeclined {
		return apperr.Conflict("conversation is not declined", "reopen non-declined conv")
	}
	if conv.DmDeclinedBy == nil || *conv.DmDeclinedBy != actor {
		return apperr.Forbidden("you're not the one declined this conversation", "reopen by non-decliner")
	}

	if err = s.store.UpdateDMStatus(ctx, convID, DMStatusAccepted, nil); err != nil {
		return err
	}
	if actor != *conv.DmA {
		s.notify(ctx, actor, *conv.DmA, conv, notification.KindDMReopened)
	} else {
		s.notify(ctx, actor, *conv.DmB, conv, notification.KindDMReopened)
	}

	return s.broadcast(ctx, conv, "dm.reopened")
}

// ListConversations returns the hiker's inbox: every outing chat they belong to
// and every DM they're a party of, newest activity first. Scoping is done by the
// store; there is no policy here.
func (s *Service) ListConversations(ctx context.Context, hikerID uuid.UUID) ([]*ConversationSummary, error) {
	return s.store.ListConversations(ctx, hikerID)
}

// GetConversation returns one conversation the hiker belongs to: an outing chat
// they're on the roster of, or a DM they're a party of. Forbidden for anyone else,
// NotFound when the conversation doesn't exist.
func (s *Service) GetConversation(ctx context.Context, convID, hikerID uuid.UUID) (*ConversationView, error) {
	conv, err := s.store.GetConversationView(ctx, convID)
	if err != nil {
		return nil, err
	}
	ok, err := s.store.IsMember(ctx, convID, hikerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.Forbidden("you're not part of this conversation", "get by non-member")
	}
	return conv, nil
}

func (s *Service) notify(ctx context.Context, actor, hikerID uuid.UUID, conv *Conversation, kind notification.Kind) {
	actorName, err := s.store.HikerName(ctx, actor)
	if err != nil {
		logger.Warn(ctx, "failed to capture actor name", "err", err)
	}
	e := &notification.Event{
		HikerID: hikerID,
		Kind:    kind,
		Payload: map[string]any{"conversation_id": conv.ID.String(), "from_name": actorName},
	}
	if err = s.notifications.Insert(ctx, e); err != nil {
		logger.Warn(ctx, "failed to send notification", "err", err)
	} else {
		data, _ := json.Marshal(map[string]any{"kind": kind, "conversation_id": conv.ID})
		s.broadcaster.BroadcastToUser(hikerID, events.Event{Type: "notification.created", Data: string(data)})
	}
}
