package api

import (
	"github.com/diagnosis/muster/internal/events"
	"github.com/diagnosis/muster/internal/message"
	"github.com/google/uuid"
)

type sent struct {
	HikerID uuid.UUID
	Event   events.Event
}

type fakeBroadcaster struct {
	sent []sent
}

func newFakeBroadcaster() *fakeBroadcaster {
	return &fakeBroadcaster{sent: make([]sent, 0)}
}

func (f *fakeBroadcaster) BroadcastToUser(hikerID uuid.UUID, e events.Event) {
	f.sent = append(f.sent, sent{HikerID: hikerID, Event: e})
}

var _ message.Broadcaster = (*fakeBroadcaster)(nil)
