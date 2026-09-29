package api

import (
	"context"

	"github.com/diagnosis/muster/internal/message"
	"github.com/google/uuid"
)

type fakeMessageService struct {
	message               *message.Message
	messages              []*message.Message
	conversation          *message.Conversation
	conversationSummaries []*message.ConversationSummary
	gotConvID             uuid.UUID
	gotHikerID            uuid.UUID
	gotOtherID            uuid.UUID
	gotMsgID              uuid.UUID
	gotBody               string
	err                   error
}

func (f *fakeMessageService) PostMessage(ctx context.Context, convID, hikerID uuid.UUID, body string) (*message.Message, error) {
	f.gotConvID, f.gotHikerID, f.gotBody = convID, hikerID, body
	return f.message, f.err
}

func (f *fakeMessageService) ListMessages(ctx context.Context, convID, hikerID uuid.UUID) ([]*message.Message, error) {
	f.gotConvID, f.gotHikerID = convID, hikerID
	return f.messages, f.err
}

func (f *fakeMessageService) DeleteMessage(ctx context.Context, msgID, hikerID uuid.UUID) error {
	f.gotMsgID, f.gotHikerID = msgID, hikerID
	return f.err
}

func (f *fakeMessageService) StartDM(ctx context.Context, h1, h2 uuid.UUID) (*message.Conversation, error) {
	f.gotHikerID, f.gotOtherID = h1, h2
	return f.conversation, f.err
}

func (f *fakeMessageService) AcceptDM(ctx context.Context, convID, actor uuid.UUID) error {
	f.gotConvID, f.gotHikerID = convID, actor
	return f.err
}

func (f *fakeMessageService) DeclineDM(ctx context.Context, convID, actor uuid.UUID) error {
	f.gotConvID, f.gotHikerID = convID, actor
	return f.err
}

func (f *fakeMessageService) ReopenDM(ctx context.Context, convID, actor uuid.UUID) error {
	f.gotConvID, f.gotHikerID = convID, actor
	return f.err
}
func (f *fakeMessageService) ListConversations(ctx context.Context, hikerID uuid.UUID) ([]*message.ConversationSummary, error) {
	f.gotHikerID = hikerID
	return f.conversationSummaries, f.err
}

func newFakeMessageService() *fakeMessageService {
	return &fakeMessageService{}
}

var _ messageService = (*fakeMessageService)(nil)
