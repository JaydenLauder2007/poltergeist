package poltergeist

import "sync"

// =============================================================================
// EVENT TYPES - Event-driven architecture constants
// =============================================================================

// EventType represents the type of event in the pipeline
type EventType string

// Standard event types
const (
	EventBeforeRequest EventType = "before_request" // Before request processing
	EventAfterRequest  EventType = "after_request"  // After request processing
	EventError         EventType = "on_error"       // On error occurrence
	EventServerStart   EventType = "server_start"   // Server started
	EventServerStop    EventType = "server_stop"    // Server stopping
	EventWSConnect     EventType = "ws_connect"     // WebSocket connected
	EventWSDisconnect  EventType = "ws_disconnect"  // WebSocket disconnected
	EventWSMessage     EventType = "ws_message"     // WebSocket message received
	EventSSEConnect    EventType = "sse_connect"    // SSE client connected
	EventSSEDisconnect EventType = "sse_disconnect" // SSE client disconnected
)

// =============================================================================
// EVENT HANDLER - Handler function type
// =============================================================================

// EventHandler represents an event handler function
type EventHandler func(ctx *Context)

// =============================================================================
// EVENT PIPELINE - Event-driven request lifecycle
// =============================================================================

// EventPipeline manages event handlers for request lifecycle
type EventPipeline struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// NewEventPipeline creates a new event pipeline
func NewEventPipeline() *EventPipeline {
	return &EventPipeline{
		handlers: make(map[EventType][]EventHandler),
	}
}

// --- Core Methods ---

// On registers an event handler for an event type
func (p *EventPipeline) On(event EventType, handler EventHandler) *EventPipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[event] = append(p.handlers[event], handler)
	return p
}

// Off removes all handlers for an event type
func (p *EventPipeline) Off(event EventType) *EventPipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.handlers, event)
	return p
}

// Emit triggers an event with context
func (p *EventPipeline) Emit(event EventType, ctx *Context) {
	p.mu.RLock()
	handlers := p.handlers[event]
	p.mu.RUnlock()

	for _, handler := range handlers {
		if ctx != nil {
			handler(ctx)
		}
	}
}

// EmitAsync triggers an event asynchronously
func (p *EventPipeline) EmitAsync(event EventType, ctx *Context) {
	p.mu.RLock()
	handlers := p.handlers[event]
	p.mu.RUnlock()

	for _, handler := range handlers {
		if ctx != nil {
			go handler(ctx)
		}
	}
}

// HasHandlers returns true if the event has registered handlers
func (p *EventPipeline) HasHandlers(event EventType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.handlers[event]) > 0
}

// Clear removes all event handlers
func (p *EventPipeline) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = make(map[EventType][]EventHandler)
}

// =============================================================================
// CONVENIENCE METHODS - Fluent API for common events
// =============================================================================

// BeforeRequest registers a handler for before request events
func (p *EventPipeline) BeforeRequest(handler EventHandler) *EventPipeline {
	return p.On(EventBeforeRequest, handler)
}

// AfterRequest registers a handler for after request events
func (p *EventPipeline) AfterRequest(handler EventHandler) *EventPipeline {
	return p.On(EventAfterRequest, handler)
}

// OnError registers a handler for error events
func (p *EventPipeline) OnError(handler EventHandler) *EventPipeline {
	return p.On(EventError, handler)
}

// OnServerStart registers a handler for server start events
func (p *EventPipeline) OnServerStart(handler func()) *EventPipeline {
	return p.On(EventServerStart, func(ctx *Context) {
		handler()
	})
}

// OnServerStop registers a handler for server stop events
func (p *EventPipeline) OnServerStop(handler func()) *EventPipeline {
	return p.On(EventServerStop, func(ctx *Context) {
		handler()
	})
}

// OnWSConnect registers a handler for WebSocket connect events
func (p *EventPipeline) OnWSConnect(handler EventHandler) *EventPipeline {
	return p.On(EventWSConnect, handler)
}

// OnWSDisconnect registers a handler for WebSocket disconnect events
func (p *EventPipeline) OnWSDisconnect(handler EventHandler) *EventPipeline {
	return p.On(EventWSDisconnect, handler)
}

// OnWSMessage registers a handler for WebSocket message events
func (p *EventPipeline) OnWSMessage(handler EventHandler) *EventPipeline {
	return p.On(EventWSMessage, handler)
}

// OnSSEConnect registers a handler for SSE connect events
func (p *EventPipeline) OnSSEConnect(handler EventHandler) *EventPipeline {
	return p.On(EventSSEConnect, handler)
}

// OnSSEDisconnect registers a handler for SSE disconnect events
func (p *EventPipeline) OnSSEDisconnect(handler EventHandler) *EventPipeline {
	return p.On(EventSSEDisconnect, handler)
}
