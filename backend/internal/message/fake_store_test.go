package message

import (
	"context"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

type fakeStore struct {
	converstations map[uuid.UUID]*Conversation
	members        map[uuid.UUID]map[uuid.UUID]struct{}
	messages       map[uuid.UUID]Message
	seq            int64
}

func (f *fakeStore) InsertMessage(ctx context.Context, m *Message) error {
	f.seq++
	m.Seq = f.seq
	m.ID = uuid.New()
	m.CreatedAt = time.Now()
	f.messages[m.ID] = *m
	return nil
}

func (f *fakeStore) MemberIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
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

func newFakeStore() *fakeStore {
	return &fakeStore{
		converstations: make(map[uuid.UUID]*Conversation),
		members:        make(map[uuid.UUID]map[uuid.UUID]struct{}),
		messages:       make(map[uuid.UUID]Message),
		seq:            0,
	}
}

func (f *fakeStore) GetConversation(ctx context.Context, id uuid.UUID) (*Conversation, error) {
	if v, ok := f.converstations[id]; !ok {
		return nil, apperr.NotFound("conversation not found", "not found")
	} else {
		return v, nil
	}

}

func (f *fakeStore) IsMember(ctx context.Context, conversationID, hikerID uuid.UUID) (bool, error) {
	set := f.members[conversationID]
	if _, ok := set[hikerID]; ok {
		return true, nil
	}
	return false, nil
}

func (f *fakeStore) addOutingConversation(outingID, host uuid.UUID, members ...uuid.UUID) *Conversation {
	conv := &Conversation{
		ID:       uuid.New(),
		Kind:     ConversationKindOuting,
		OutingID: &outingID,
	}
	f.converstations[conv.ID] = conv

	memberSet := make(map[uuid.UUID]struct{})
	f.members[conv.ID] = memberSet

	memberSet[host] = struct{}{}
	for _, m := range members {
		memberSet[m] = struct{}{}
	}
	return conv
}

var _ Storage = (*fakeStore)(nil)
