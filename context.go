package poltergeist

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// =============================================================================
// CONTEXT - Request/Response handling with helpers
// =============================================================================

// Context represents the request context with helper methods
type Context struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	Params     map[string]string
	statusCode int
	written    bool
	keys       map[string]any
	mu         sync.RWMutex

	// Realtime connections
	WS  *WSConn    // WebSocket connection (if upgraded)
	SSE *SSEWriter // SSE writer (if streaming)

	// Internal
	pipeline *EventPipeline
}

// NewContext creates a new Context instance (exported for testing)
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer:     w,
		Request:    r,
		Params:     make(map[string]string),
		statusCode: http.StatusOK,
		keys:       make(map[string]any),
	}
}

// reset reuses the context for pooling (performance optimization)
func (c *Context) reset(w http.ResponseWriter, r *http.Request) {
	c.Writer = w
	c.Request = r
	c.Params = make(map[string]string)
	c.statusCode = http.StatusOK
	c.written = false
	c.keys = make(map[string]any)
	c.WS = nil
	c.SSE = nil
}

// =============================================================================
// REQUEST HELPERS - Reading request data
// =============================================================================

// Bind parses JSON request body into the provided struct
func (c *Context) Bind(v any) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	defer c.Request.Body.Close()
	return json.Unmarshal(body, v)
}

// --- Query Parameters ---

// Query returns a query parameter by key
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// QueryDefault returns a query parameter or default value if empty
func (c *Context) QueryDefault(key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}

// QueryInt returns a query parameter as integer
func (c *Context) QueryInt(key string) (int, error) {
	return strconv.Atoi(c.Query(key))
}

// QueryIntDefault returns a query parameter as integer or default value
func (c *Context) QueryIntDefault(key string, defaultValue int) int {
	if v, err := c.QueryInt(key); err == nil {
		return v
	}
	return defaultValue
}

// QueryBool returns a query parameter as boolean
func (c *Context) QueryBool(key string) bool {
	v := strings.ToLower(c.Query(key))
	return v == "true" || v == "1" || v == "yes"
}

// --- Path Parameters ---

// Param returns a path parameter by key
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// ParamInt returns a path parameter as integer
func (c *Context) ParamInt(key string) (int, error) {
	return strconv.Atoi(c.Param(key))
}

// --- Headers ---

// Header returns a request header value
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) {
	c.Writer.Header().Set(key, value)
}

// ContentType returns the Content-Type header
func (c *Context) ContentType() string {
	return c.Header(HeaderContentType)
}

// --- Request Info ---

// Method returns the HTTP method
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the request path
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// FullURL returns the full request URL
func (c *Context) FullURL() string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + c.Request.RequestURI
}

// ClientIP extracts the client IP address from request
// Checks proxy headers first, then falls back to RemoteAddr
func (c *Context) ClientIP() string {
	// Check proxy headers in order of preference
	for _, header := range []string{HeaderXForwardedFor, HeaderXRealIP} {
		if ip := c.Header(header); ip != "" {
			// X-Forwarded-For can contain multiple IPs, take the first
			if header == HeaderXForwardedFor {
				if idx := strings.Index(ip, ","); idx != -1 {
					ip = strings.TrimSpace(ip[:idx])
				}
			}
			return ip
		}
	}

	// Fall back to RemoteAddr (strip port if present)
	addr := c.Request.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// =============================================================================
// RESPONSE HELPERS - Writing responses (DRY: common write pattern)
// =============================================================================

// writeResponse is the internal DRY helper for all response methods
func (c *Context) writeResponse(code int, contentType string, data []byte) error {
	c.SetHeader(HeaderContentType, contentType)
	c.Writer.WriteHeader(code)
	c.written = true
	_, err := c.Writer.Write(data)
	return err
}

// Status sets the response status code (chainable)
func (c *Context) Status(code int) *Context {
	c.statusCode = code
	return c
}

// JSON sends a JSON response
func (c *Context) JSON(code int, v any) error {
	c.SetHeader(HeaderContentType, ContentTypeJSON)
	c.Writer.WriteHeader(code)
	c.written = true
	return json.NewEncoder(c.Writer).Encode(v)
}

// String sends a plain text response
func (c *Context) String(code int, s string) error {
	return c.writeResponse(code, ContentTypeText, []byte(s))
}

// HTML sends an HTML response
func (c *Context) HTML(code int, html string) error {
	return c.writeResponse(code, ContentTypeHTML, []byte(html))
}

// Bytes sends raw bytes response with custom content type
func (c *Context) Bytes(code int, contentType string, data []byte) error {
	return c.writeResponse(code, contentType, data)
}

// NoContent sends a 204 No Content response
func (c *Context) NoContent() error {
	c.Writer.WriteHeader(http.StatusNoContent)
	c.written = true
	return nil
}

// Redirect sends a redirect response
func (c *Context) Redirect(code int, url string) error {
	http.Redirect(c.Writer, c.Request, url, code)
	c.written = true
	return nil
}

// File serves a file from the filesystem
func (c *Context) File(filepath string) {
	http.ServeFile(c.Writer, c.Request, filepath)
	c.written = true
}

// =============================================================================
// ERROR RESPONSES - Convenient error helpers
// =============================================================================

// Error sends a JSON error response
func (c *Context) Error(code int, message string) error {
	return c.JSON(code, H{"error": message})
}

// BadRequest sends a 400 Bad Request response
func (c *Context) BadRequest(message string) error {
	return c.Error(http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized response
func (c *Context) Unauthorized(message string) error {
	return c.Error(http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response
func (c *Context) Forbidden(message string) error {
	return c.Error(http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response
func (c *Context) NotFound(message string) error {
	return c.Error(http.StatusNotFound, message)
}

// InternalServerError sends a 500 Internal Server Error response
func (c *Context) InternalServerError(message string) error {
	return c.Error(http.StatusInternalServerError, message)
}

// =============================================================================
// CONTEXT STORE - Request-scoped key-value storage
// =============================================================================

// Set stores a value in the context (thread-safe)
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	c.keys[key] = value
	c.mu.Unlock()
}

// Get retrieves a value from the context (thread-safe)
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	value, exists := c.keys[key]
	c.mu.RUnlock()
	return value, exists
}

// MustGet retrieves a value from the context, panics if not found
func (c *Context) MustGet(key string) any {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist in context")
}

// GetString retrieves a string value from the context
func (c *Context) GetString(key string) string {
	if value, exists := c.Get(key); exists {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt retrieves an int value from the context
func (c *Context) GetInt(key string) int {
	if value, exists := c.Get(key); exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// Written returns true if response has been written
func (c *Context) Written() bool {
	return c.written
}
