package outing

import (
	"github.com/diagnosis/muster/internal/events"
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
func (f *fakeBroadcaster) sentTo(hikerID uuid.UUID) []events.Event {
	me := []events.Event{}
	for _, s := range f.sent {
		if s.HikerID == hikerID {
			me = append(me, s.Event)
		}
	}
	return me
}

var _ events.Broadcaster = (*fakeBroadcaster)(nil)
