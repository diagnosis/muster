package api

import (
	"context"

	"github.com/diagnosis/muster/internal/message"
	"github.com/google/uuid"
)

type fakeMessageService struct {
	message    *message.Message
	messages   []*message.Message
	gotConvID  uuid.UUID
	gotHikerID uuid.UUID
	gotMsgID   uuid.UUID
	gotBody    string
	err        error
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

func newFakeMessageService() *fakeMessageService {
	return &fakeMessageService{}
}

var _ messageService = (*fakeMessageService)(nil)
