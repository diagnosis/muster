package api

import (
	"context"
	"sort"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/message"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
)

type messageFakeStore struct {
	converstations map[uuid.UUID]*message.Conversation
	members        map[uuid.UUID]map[uuid.UUID]struct{}
	messages       map[uuid.UUID]message.Message
	seq            int64
	outingStatuses map[uuid.UUID]outing.Status
	hosts          map[uuid.UUID]uuid.UUID
}

func (f *messageFakeStore) InsertMessage(ctx context.Context, m *message.Message, now time.Time) error {
	f.seq++
	m.Seq = f.seq
	m.ID = uuid.New()
	m.CreatedAt = now
	f.messages[m.ID] = *m
	return nil
}

func (f *messageFakeStore) MemberIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	_, err := f.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	set := f.members[conversationID]
	members := []uuid.UUID{}
	for k := range set {
		members = append(members, k)
	}
	return members, nil
}

func newMessageFakeStore() *messageFakeStore {
	return &messageFakeStore{
		converstations: make(map[uuid.UUID]*message.Conversation),
		members:        make(map[uuid.UUID]map[uuid.UUID]struct{}),
		messages:       make(map[uuid.UUID]message.Message),
		seq:            0,
		outingStatuses: make(map[uuid.UUID]outing.Status),
		hosts:          make(map[uuid.UUID]uuid.UUID),
	}
}

func (f *messageFakeStore) GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error) {
	if v, ok := f.converstations[id]; !ok {
		return nil, apperr.NotFound("conversation not found", "not found")
	} else {
		return v, nil
	}

}

func (f *messageFakeStore) IsMember(ctx context.Context, conversationID, hikerID uuid.UUID) (bool, error) {
	set := f.members[conversationID]
	if _, ok := set[hikerID]; ok {
		return true, nil
	}
	return false, nil
}

func (f *messageFakeStore) addOutingConversation(outingID, host uuid.UUID, outingStatus outing.Status, members ...uuid.UUID) *message.Conversation {
	conv := &message.Conversation{
		ID:       uuid.New(),
		Kind:     message.ConversationKindOuting,
		OutingID: &outingID,
	}
	f.converstations[conv.ID] = conv
	f.outingStatuses[outingID] = outingStatus
	f.hosts[outingID] = host
	memberSet := make(map[uuid.UUID]struct{})
	f.members[conv.ID] = memberSet

	memberSet[host] = struct{}{}
	for _, m := range members {
		memberSet[m] = struct{}{}
	}
	return conv
}

func (f *messageFakeStore) OutingStatus(ctx context.Context, outingID uuid.UUID) (outing.Status, error) {
	v, ok := f.outingStatuses[outingID]
	if !ok {
		return "", apperr.NotFound("outing not found", "outing not found")
	}
	return v, nil
}
func (f *messageFakeStore) CountMessagesSince(ctx context.Context, conversationID, hikerID uuid.UUID, since time.Time) (int, error) {
	count := 0
	for _, m := range f.messages {
		if m.ConversationID == conversationID && m.HikerID == hikerID && m.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (f *messageFakeStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*message.Message, error) {
	mes := []*message.Message{}
	for _, m := range f.messages {
		if m.ConversationID == conversationID {
			mes = append(mes, &m)
		}
	}
	sort.Slice(mes, func(i, j int) bool {
		return mes[i].Seq < mes[j].Seq
	})
	return mes, nil
}

func (f *messageFakeStore) GetMessage(ctx context.Context, messageID uuid.UUID) (*message.Message, error) {
	v, ok := f.messages[messageID]
	if !ok {
		return nil, apperr.NotFound("message not found", "message not found")
	}
	return &v, nil
}

func (f *messageFakeStore) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	delete(f.messages, messageID)
	return nil
}
func (f *messageFakeStore) OutingHost(ctx context.Context, outingID uuid.UUID) (uuid.UUID, error) {
	v, ok := f.hosts[outingID]
	if !ok {
		return uuid.Nil, apperr.NotFound("host not found", "host not found")
	}
	return v, nil
}

var _ message.Storage = (*messageFakeStore)(nil)
