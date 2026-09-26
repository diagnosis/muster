package events

import (
	"sync"

	"github.com/google/uuid"
)

// Event is a poke delivered over the user stream: a type the client switches
// on and a small data payload. It never carries a message body (D3).
type Event struct {
	Type string
	Data string
}

// Client is one open connection for one hiker (a tab, a phone). Send is owned
// by the Hub: the Hub closes it on Unregister or CloseAll, and only the stream
// handler reads from it.
type Client struct {
	HikerID uuid.UUID
	Send    chan Event
}

// Hub is the registry of open connections keyed by hiker. It holds no roster
// and no policy; callers decide who receives what. Safe for concurrent use.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]struct{}
}

// NewHub returns an empty Hub ready for use.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]struct{}),
	}
}

// Register adds c to the registry. Registering the same client twice is a no-op.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.HikerID]; !ok {
		h.clients[c.HikerID] = make(map[*Client]struct{})
	}
	h.clients[c.HikerID][c] = struct{}{}

}

// Unregister removes c and closes its Send channel, which is how the stream
// handler learns to exit. Calling it for a client that is not registered,
// including a second call for the same client, is a no-op.
func (h *Hub) Unregister(c *Client) {
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

// BroadcastToUser delivers e to every registered client of hikerID. It never
// blocks: if a client's Send buffer is full the event is dropped for that
// client (D3: pokes are idempotent, the client refetches on reconnect/focus).
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

// CloseAll closes every client's Send channel and empties the registry. It is
// called on shutdown before http.Server.Shutdown so stream handlers return and
// the server has no active connections left to wait on.
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, v := range h.clients {
		for c := range v {
			close(c.Send)
		}
		delete(h.clients, k)
	}

}

// Count reports the number of open connections registered for hikerID.
func (h *Hub) Count(hikerID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[hikerID])
}
