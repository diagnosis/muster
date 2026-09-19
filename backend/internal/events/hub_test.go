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

	h.BroadcastToUser(hiker1ID, Event{Type: "message.created", Data: "x"})

	select {
	case got := <-c1.Send:

		if got.Type != "message.created" {
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

	h.Unregister(c1)
	h.BroadcastToUser(hiker1ID, Event{Type: "message.created", Data: "Y"})
	select {
	case ev, ok := <-c1.Send:
		if ok {
			t.Fatalf("hiker1 received %v", ev)
		}
	case <-time.After(time.Second):
		t.Fatalf("c1.Send not closed")
	}
}

func Test_Hub_DoubleUnregister(t *testing.T) {
	h := NewHub()
	hiker := uuid.New()
	c1 := &Client{
		HikerID: hiker,
		Send:    make(chan Event, 1),
	}
	h.Register(c1)
	set := h.clients[hiker]
	if len(set) != 1 {
		t.Fatalf("expected registered 1 got %d", len(set))
	}
	h.Unregister(c1)
	if len(set) != 0 {
		t.Errorf("expected registered 0 got %d", len(set))
	}
	h.Unregister(c1)
	if len(set) != 0 {
		t.Errorf("expected registered 0 got %d", len(set))
	}
}

func Test_Hub_TwoTabs(t *testing.T) {
	h := NewHub()
	hiker := uuid.New()
	c1 := &Client{
		HikerID: hiker,
		Send:    make(chan Event, 1),
	}
	c1b := &Client{
		HikerID: hiker,
		Send:    make(chan Event, 1),
	}
	h.Register(c1)
	h.Register(c1b)
	h.Unregister(c1)
	h.BroadcastToUser(hiker, Event{
		Type: "message.created",
		Data: "XYZ",
	})
	select {
	case ev, ok := <-c1b.Send:
		if !ok {
			t.Fatal("c11 send closed")
		}
		if ev.Type != "message.created" {
			t.Errorf("expected type message.created got %s", ev.Type)
		}
		if ev.Data != "XYZ" {
			t.Errorf("expected data: XYZ got %s", ev.Data)
		}
	case <-time.After(time.Second):
		t.Fatalf("time out")
	}
	select {
	case _, ok := <-c1.Send:
		if ok {
			t.Error("expected hiker not received event")
		}
	case <-time.After(time.Second):
		t.Fatalf("time out/ chan not closed")
	}
}

func Test_Hub_FullBuffer(t *testing.T) {
	h := NewHub()
	hiker := uuid.New()
	c := &Client{
		HikerID: hiker,
		Send:    make(chan Event, 1),
	}
	h.Register(c)
	done := make(chan struct{})
	e1 := Event{
		Type: "message.created",
		Data: "XYZ",
	}
	e2 := Event{
		Type: "message.created",
		Data: "REW",
	}
	go func() {
		h.BroadcastToUser(hiker, e1)
		h.BroadcastToUser(hiker, e2)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hub blocked on full buffer")
	}
	select {
	case e := <-c.Send:
		if e.Data != "XYZ" {
			t.Errorf("expected XYZ got %s", e.Data)
		}
	case <-time.After(time.Second):
		t.Fatalf("nothing arrived")
	}
	select {
	case e := <-c.Send:
		t.Fatalf("expected drop, but got %v", e)
	default:
	}

}

func Test_Hub_CloseAll(t *testing.T) {
	h := NewHub()
	hiker1 := uuid.New()
	hiker2 := uuid.New()
	c1 := &Client{
		HikerID: hiker1,
		Send:    make(chan Event, 1),
	}
	c2 := &Client{
		HikerID: hiker2,
		Send:    make(chan Event, 1),
	}
	h.Register(c1)
	h.Register(c2)
	h.CloseAll()

	h.BroadcastToUser(hiker1, Event{
		Type: "message.created",
		Data: "IO",
	})
	h.BroadcastToUser(hiker2, Event{Type: "message.created", Data: "OI"})

	select {
	case _, ok := <-c1.Send:
		if ok {
			t.Fatalf("expected chan was closed")
		}
	case <-time.After(time.Second):
		t.Fatalf("not closed")
	}
	select {
	case _, ok := <-c2.Send:
		if ok {
			t.Fatalf("expected chan was closed")
		}
	case <-time.After(time.Second):
		t.Fatalf("not closed")
	}

}

func Test_Hub_CountHikers(t *testing.T) {
	h := NewHub()
	hiker1 := uuid.New()
	c1 := &Client{
		HikerID: hiker1,
		Send:    make(chan Event, 1),
	}
	if h.Count(hiker1) != 0 {
		t.Fatalf("expected count 0 got %d", h.Count(hiker1))
	}
	h.Register(c1)
	if h.Count(hiker1) != 1 {
		t.Fatalf("expected count 1 got %d", h.Count(hiker1))
	}
}

func Test_Hub_Unregister(t *testing.T) {
	h := NewHub()
	hiker := uuid.New()
	c := &Client{
		HikerID: hiker,
		Send:    make(chan Event, 1),
	}
	h.Register(c)
	if h.Count(hiker) != 1 {
		t.Fatalf("expected count 1 got %d", h.Count(hiker))
	}
	h.Unregister(c)
	if h.Count(hiker) != 0 {
		t.Fatalf("expected count 0 got %d", h.Count(hiker))
	}
}
