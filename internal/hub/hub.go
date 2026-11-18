package hub

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is the socket payload delivered to connected devices.
// It mirrors the FCM message shape DisplayLab already sends.
type Event struct {
	Event string                 `json:"event"`
	Token string                 `json:"token"`
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Data  map[string]interface{} `json:"data"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{} // device token -> clients
}

func New() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.Token]; !ok {
		h.clients[c.Token] = make(map[*Client]struct{})
	}
	h.clients[c.Token][c] = struct{}{}
	log.Printf("ws connected token=%s active=%d", truncate(c.Token), len(h.clients[c.Token]))
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.Token]
	if !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, c.Token)
	}
	log.Printf("ws disconnected token=%s remaining=%d", truncate(c.Token), len(set))
}

// Publish sends an event to every socket subscribed to the device token.
// Returns how many clients received it.
func (h *Hub) Publish(ev Event) int {
	payload, err := json.Marshal(ev)
	if err != nil {
		log.Printf("marshal event error: %v", err)
		return 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	set := h.clients[ev.Token]
	sent := 0
	for c := range set {
		select {
		case c.send <- payload:
			sent++
		default:
			// Slow client — drop this message rather than blocking notify path.
			log.Printf("ws drop slow client token=%s", truncate(c.Token))
		}
	}
	return sent
}

func (h *Hub) Stats() (devices, connections int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	devices = len(h.clients)
	for _, set := range h.clients {
		connections += len(set)
	}
	return
}

func truncate(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:6] + "…" + s[len(s)-4:]
}
