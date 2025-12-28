package events

import "sync"

// EventType represents the type of event
type EventType string

// Standard event types
const (
	BeforeRequest EventType = "before_request"
	AfterRequest  EventType = "after_request"
	OnError       EventType = "on_error"
	ServerStart   EventType = "server_start"
	ServerStop    EventType = "server_stop"
	WSConnect     EventType = "ws_connect"
	WSDisconnect  EventType = "ws_disconnect"
	WSMessage     EventType = "ws_message"
	SSEConnect    EventType = "sse_connect"
	SSEDisconnect EventType = "sse_disconnect"
)

// EventHandler represents an event handler function
type EventHandler func(data any)

// Pipeline is the main event pipeline for event-driven architecture
type Pipeline struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// NewPipeline creates a new event pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		handlers: make(map[EventType][]EventHandler),
	}
}

// On registers an event handler
func (p *Pipeline) On(event EventType, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[event] = append(p.handlers[event], handler)
}

// Off removes all handlers for an event
func (p *Pipeline) Off(event EventType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.handlers, event)
}

// Emit triggers an event with data
func (p *Pipeline) Emit(event EventType, data any) {
	p.mu.RLock()
	handlers := p.handlers[event]
	p.mu.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// EmitAsync triggers an event asynchronously
func (p *Pipeline) EmitAsync(event EventType, data any) {
	p.mu.RLock()
	handlers := p.handlers[event]
	p.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// HasHandlers returns true if the event has handlers
func (p *Pipeline) HasHandlers(event EventType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.handlers[event]) > 0
}

// Clear removes all event handlers
func (p *Pipeline) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = make(map[EventType][]EventHandler)
}

