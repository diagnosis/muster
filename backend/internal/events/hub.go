package events

import (
	"sync"

	"github.com/google/uuid"
)

type Event struct {
	Type string
	Data string
}

type Client struct {
	HikerID uuid.UUID
	Send    chan Event
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]struct{}),
	}
}
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.HikerID]; !ok {
		h.clients[c.HikerID] = make(map[*Client]struct{})
	}
	h.clients[c.HikerID][c] = struct{}{}

}
func (h *Hub) UnRegister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	set := h.clients[c.HikerID]
	if _, ok := set[c]; !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, c.HikerID)
	}
	close(c.Send)
}

func (h *Hub) BroadcastToUser(hikerID uuid.UUID, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[hikerID] {
		select {
		case c.Send <- e:
		default:
		}
	}
}
