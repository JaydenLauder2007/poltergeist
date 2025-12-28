package poltergeist

import "sync"

// =============================================================================
// BASE HUB - Common functionality for WebSocket and SSE hubs (DRY)
// =============================================================================

// BaseHub provides common hub functionality for managing connections and rooms
// This implements the DRY principle by extracting shared code
type BaseHub struct {
	mu      sync.RWMutex
	rooms   map[string]map[string]bool // room -> set of client IDs
	running bool
}

// newBaseHub creates a new BaseHub
func newBaseHub() *BaseHub {
	return &BaseHub{
		rooms: make(map[string]map[string]bool),
	}
}

// addToRoom adds a client to a room
func (h *BaseHub) addToRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]bool)
	}
	h.rooms[room][clientID] = true
}

// removeFromRoom removes a client from a room
func (h *BaseHub) removeFromRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[room]; ok {
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}

// removeFromAllRooms removes a client from all rooms
func (h *BaseHub) removeFromAllRooms(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for room, clients := range h.rooms {
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}

// getRoomClientIDs returns all client IDs in a room
func (h *BaseHub) getRoomClientIDs(room string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[room]
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(clients))
	for id := range clients {
		ids = append(ids, id)
	}
	return ids
}

// roomCount returns the number of clients in a room
func (h *BaseHub) roomCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.rooms[room]; ok {
		return len(clients)
	}
	return 0
}

// isRunning returns whether the hub is running
func (h *BaseHub) isRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// setRunning sets the running state
func (h *BaseHub) setRunning(running bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running = running
}
