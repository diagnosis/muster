package message

import (
	"context"
	"sort"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
)

type fakeStore struct {
	converstations  map[uuid.UUID]*Conversation
	members         map[uuid.UUID]map[uuid.UUID]struct{}
	messages        map[uuid.UUID]Message
	seq             int64
	outingStatuses  map[uuid.UUID]outing.Status
	hosts           map[uuid.UUID]uuid.UUID
	dms             map[[2]uuid.UUID]*Conversation
	pendingRequests map[uuid.UUID]map[uuid.UUID]struct{}
	outingTitles    map[uuid.UUID]string
	outingStarts    map[uuid.UUID]time.Time
	names           map[uuid.UUID]string
}

func (f *fakeStore) InsertMessage(ctx context.Context, m *Message, now time.Time) error {
	f.seq++
	m.Seq = f.seq
	m.ID = uuid.New()
	m.CreatedAt = now
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
		converstations:  make(map[uuid.UUID]*Conversation),
		members:         make(map[uuid.UUID]map[uuid.UUID]struct{}),
		messages:        make(map[uuid.UUID]Message),
		seq:             0,
		outingStatuses:  make(map[uuid.UUID]outing.Status),
		hosts:           make(map[uuid.UUID]uuid.UUID),
		dms:             make(map[[2]uuid.UUID]*Conversation),
		pendingRequests: make(map[uuid.UUID]map[uuid.UUID]struct{}),
		outingTitles:    make(map[uuid.UUID]string),
		outingStarts:    make(map[uuid.UUID]time.Time),
		names:           make(map[uuid.UUID]string),
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

func (f *fakeStore) addHiker(m uuid.UUID, name string) {
	if _, ok := f.names[m]; !ok {
		f.names[m] = name
	}

}
func (f *fakeStore) addOutingConversation(outingID, host uuid.UUID, outingStatus outing.Status, members ...uuid.UUID) *Conversation {
	conv := &Conversation{
		ID:       uuid.New(),
		Kind:     ConversationKindOuting,
		OutingID: &outingID,
	}
	f.converstations[conv.ID] = conv
	f.outingStatuses[outingID] = outingStatus
	f.hosts[outingID] = host
	f.outingTitles[outingID] = "outing " + outingID.String()[:8]
	f.outingStarts[outingID] = time.Now().Add(7 * 24 * time.Hour)
	f.addHiker(host, "host "+host.String()[:4])
	memberSet := make(map[uuid.UUID]struct{})
	f.members[conv.ID] = memberSet

	memberSet[host] = struct{}{}
	for _, m := range members {
		memberSet[m] = struct{}{}
		f.addHiker(m, "hiker "+m.String()[:4])
	}
	return conv
}
func (f *fakeStore) addPendingRequest(outingID, hiker uuid.UUID) {
	if f.pendingRequests[outingID] == nil {
		f.pendingRequests[outingID] = make(map[uuid.UUID]struct{})
	}

	set := f.pendingRequests[outingID]
	set[hiker] = struct{}{}
}

func (f *fakeStore) OutingStatus(ctx context.Context, outingID uuid.UUID) (outing.Status, error) {
	v, ok := f.outingStatuses[outingID]
	if !ok {
		return "", apperr.NotFound("outing not found", "outing not found")
	}
	return v, nil
}
func (f *fakeStore) CountMessagesSince(ctx context.Context, conversationID, hikerID uuid.UUID, since time.Time) (int, error) {
	count := 0
	for _, m := range f.messages {
		if m.ConversationID == conversationID && m.HikerID == hikerID && m.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (f *fakeStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*Message, error) {
	mes := []*Message{}
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

func (f *fakeStore) GetMessage(ctx context.Context, messageID uuid.UUID) (*Message, error) {
	v, ok := f.messages[messageID]
	if !ok {
		return nil, apperr.NotFound("message not found", "message not found")
	}
	return &v, nil
}

func (f *fakeStore) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	delete(f.messages, messageID)
	return nil
}
func (f *fakeStore) OutingHost(ctx context.Context, outingID uuid.UUID) (uuid.UUID, error) {
	v, ok := f.hosts[outingID]
	if !ok {
		return uuid.Nil, apperr.NotFound("host not found", "host not found")
	}
	return v, nil
}
func (f *fakeStore) GetOrCreateDM(ctx context.Context, lo, hi, a uuid.UUID) (*Conversation, bool, error) {
	v, ok := f.dms[[2]uuid.UUID{lo, hi}]
	if ok {
		return v, false, nil
	}
	status := DMStatusPending
	c := &Conversation{
		ID:          uuid.New(),
		Kind:        ConversationKindDM,
		DmA:         &lo,
		DmB:         &hi,
		DmInitiator: &a,
		DmStatus:    &status,
		CreatedAt:   time.Now(),
	}
	f.dms[[2]uuid.UUID{lo, hi}] = c
	f.converstations[c.ID] = c
	f.members[c.ID] = map[uuid.UUID]struct{}{lo: {}, hi: {}}
	f.members[c.ID][lo] = struct{}{}
	f.members[c.ID][hi] = struct{}{}
	f.addHiker(lo, "hiker "+lo.String()[:4])
	f.addHiker(hi, "hiker "+hi.String()[:4])
	return c, true, nil
}
func (f *fakeStore) CanDM(ctx context.Context, h1, h2 uuid.UUID) (bool, error) {
	for _, c := range f.converstations {
		if c.Kind != ConversationKindOuting {
			continue
		}
		set := f.members[c.ID]
		_, aIn := set[h1]
		_, bIn := set[h2]
		if aIn && bIn {
			return true, nil
		}
		if f.hosts[*c.OutingID] == h1 {
			if _, pend := f.pendingRequests[*c.OutingID][h2]; pend {
				return true, nil
			}
		}
	}
	return false, nil
}

func (f *fakeStore) setDMStatus(id uuid.UUID, status DMStatus, declinedBy *uuid.UUID) {
	c := f.converstations[id]
	c.DmStatus = &status
	c.DmDeclinedBy = declinedBy
}
func (f *fakeStore) UpdateDMStatus(ctx context.Context, convID uuid.UUID, status DMStatus, declinedBy *uuid.UUID) error {
	if _, ok := f.converstations[convID]; !ok {
		return apperr.NotFound("not found", "not found")
	}
	f.setDMStatus(convID, status, declinedBy)
	return nil
}
func (f *fakeStore) ListConversations(ctx context.Context, hikerID uuid.UUID) ([]*ConversationSummary, error) {

	return nil, apperr.Internal("not implemented", "not implemented")
}

func (f *fakeStore) GetConversationView(ctx context.Context, convID uuid.UUID) (*ConversationView, error) {
	c, ok := f.converstations[convID]
	if !ok {
		return nil, apperr.NotFound("not found", "not found")
	}
	view := &ConversationView{
		ID: c.ID, Kind: c.Kind, OutingID: c.OutingID,
		DmA: c.DmA, DmB: c.DmB, DmInitiator: c.DmInitiator,
		DmStatus: c.DmStatus, DmDeclinedBy: c.DmDeclinedBy,
		CreatedAt: c.CreatedAt,
	}
	if c.Kind == ConversationKindOuting && c.OutingID != nil {
		if t, okk := f.outingTitles[*c.OutingID]; okk {
			view.OutingTitle = &t
		}
		if d, okk := f.outingStarts[*c.OutingID]; okk {
			view.OutingStartsAt = &d
		}
	}
	for id := range f.members[convID] {
		view.Participants = append(view.Participants, Participant{HikerID: id, Name: f.names[id]})
	}
	sort.Slice(view.Participants, func(i, j int) bool {
		return view.Participants[i].HikerID.String() < view.Participants[j].HikerID.String()
	})
	return view, nil
}

var _ Storage = (*fakeStore)(nil)
