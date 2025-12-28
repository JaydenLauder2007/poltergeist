package poltergeist

// =============================================================================
// INTERFACES - Dependency Inversion & Interface Segregation (SOLID I, D)
// =============================================================================

// ResponseWriter defines the interface for writing responses
// This allows for easier testing and extension
type ResponseWriter interface {
	JSON(code int, v any) error
	String(code int, s string) error
	HTML(code int, html string) error
	Bytes(code int, contentType string, data []byte) error
	NoContent() error
	Error(code int, message string) error
}

// RequestReader defines the interface for reading requests
type RequestReader interface {
	Bind(v any) error
	Query(key string) string
	QueryDefault(key, defaultValue string) string
	QueryInt(key string) (int, error)
	QueryIntDefault(key string, defaultValue int) int
	Param(key string) string
	ParamInt(key string) (int, error)
	Header(key string) string
}

// RouteRegistrar defines the interface for registering routes
// Implements Interface Segregation - clients only depend on what they need
type RouteRegistrar interface {
	GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route
	Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
}

// MiddlewareUser defines interface for adding middleware
type MiddlewareUser interface {
	Use(middlewares ...MiddlewareFunc)
}

// Broadcaster defines the interface for broadcasting messages
// DRY: Common interface for WSHub and SSEHub
type Broadcaster interface {
	Broadcast(data []byte)
	BroadcastToRoom(room string, data []byte)
}

// RoomManager defines the interface for room management
// DRY: Common interface for WSHub and SSEHub
type RoomManager interface {
	JoinRoom(clientID string, room string)
	LeaveRoom(clientID string, room string)
	RoomCount(room string) int
}

// Ensure interfaces are implemented (compile-time check)
var (
	_ ResponseWriter = (*Context)(nil)
	_ RequestReader  = (*Context)(nil)
	_ RouteRegistrar = (*Router)(nil)
	_ RouteRegistrar = (*RouteGroup)(nil)
	_ RouteRegistrar = (*Server)(nil)
)
