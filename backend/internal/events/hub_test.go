package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test_Hub_Register(t *testing.T) {
	h := NewHub()
	hiker1ID := uuid.New()
	hiker2ID := uuid.New()
	c1 := &Client{hiker1ID, make(chan Event, 1)}
	c2 := &Client{hiker2ID, make(chan Event, 1)}

	h.Register(c1)
	h.Register(c2)

	h.BroadcastToUser(hiker1ID, Event{Type: "message_created", Data: "x"})

	select {
	case got := <-c1.Send:

		if got.Type != "message_created" {
			t.Errorf("expected message_created got %s", got.Type)
		}
		if got.Data != "x" {
			t.Errorf("expected x got %s", got.Data)
		}
	case <-time.After(time.Second):
		t.Fatalf("time out")
	}
	select {
	case ev := <-c2.Send:
		t.Fatalf("hiker2 received %v", ev)
	default:
	}

	h.UnRegister(c1)
	h.BroadcastToUser(hiker1ID, Event{Type: "message_created", Data: "Y"})
	select {
	case ev, ok := <-c1.Send:
		if ok { t.Fatalf("hiker1 received %v", ev) }
		case <-time.After(time.Second): t.Fatalf("c1.Send not closed")
	}



}
