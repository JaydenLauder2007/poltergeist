package poltergeist

import (
	"net/http"
	"path"
	"strings"
	"sync"
)

// =============================================================================
// TYPES - Core routing types
// =============================================================================

// HandlerFunc defines the handler function signature
type HandlerFunc func(*Context) error

// MiddlewareFunc defines the middleware function signature
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Route represents a single registered route
type Route struct {
	Method      string
	Path        string
	Handler     HandlerFunc
	Middlewares []MiddlewareFunc

	// Metadata (for documentation generation)
	RouteName        string
	RouteDescription string
	RouteTags        []string
	RequestBody      any
	ResponseBody     any
}

// =============================================================================
// ROUTER - Main routing engine
// =============================================================================

// Router handles HTTP request routing
type Router struct {
	routes           []*Route
	middlewares      []MiddlewareFunc
	groups           []*RouteGroup
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	pool             sync.Pool
	pipeline         *EventPipeline
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	r := &Router{
		routes:   make([]*Route, 0),
		groups:   make([]*RouteGroup, 0),
		pipeline: NewEventPipeline(),
	}
	r.pool.New = func() any {
		return &Context{}
	}
	return r
}

// --- Middleware ---

// Use adds global middleware to the router
func (r *Router) Use(middlewares ...MiddlewareFunc) *Router {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// --- Groups ---

// Group creates a new route group with shared prefix and middleware
func (r *Router) Group(prefix string, middlewares ...MiddlewareFunc) *RouteGroup {
	group := &RouteGroup{
		prefix:      prefix,
		middlewares: middlewares,
		router:      r,
	}
	r.groups = append(r.groups, group)
	return group
}

// --- Pipeline ---

// Pipeline returns the event pipeline for hooks
func (r *Router) Pipeline() *EventPipeline {
	return r.pipeline
}

// --- Error Handlers ---

// NotFound sets the custom 404 handler
func (r *Router) NotFound(handler HandlerFunc) *Router {
	r.notFound = handler
	return r
}

// MethodNotAllowed sets the custom 405 handler
func (r *Router) MethodNotAllowed(handler HandlerFunc) *Router {
	r.methodNotAllowed = handler
	return r
}

// --- Route Registration ---

// addRoute is the internal method for registering routes (DRY)
func (r *Router) addRoute(method, routePath string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	route := &Route{
		Method:      method,
		Path:        routePath,
		Handler:     handler,
		Middlewares: middlewares,
	}
	r.routes = append(r.routes, route)
	return route
}

// HTTP method shortcuts (all delegate to addRoute - DRY)

func (r *Router) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodGet, path, handler, middlewares...)
}

func (r *Router) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodPost, path, handler, middlewares...)
}

func (r *Router) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodPut, path, handler, middlewares...)
}

func (r *Router) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodDelete, path, handler, middlewares...)
}

func (r *Router) PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodPatch, path, handler, middlewares...)
}

func (r *Router) OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodOptions, path, handler, middlewares...)
}

func (r *Router) HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return r.addRoute(http.MethodHead, path, handler, middlewares...)
}

// Any registers a route for all standard HTTP methods
func (r *Router) Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	for _, method := range AllHTTPMethods {
		r.addRoute(method, path, handler, middlewares...)
	}
}

// Static serves static files from a directory
func (r *Router) Static(urlPath, dirPath string) {
	r.GET(urlPath+"/*filepath", func(c *Context) error {
		filepath := c.Param("filepath")
		http.ServeFile(c.Writer, c.Request, path.Join(dirPath, filepath))
		return nil
	})
}

// Routes returns all registered routes
func (r *Router) Routes() []*Route {
	return r.routes
}

// =============================================================================
// HTTP HANDLER - Implements http.Handler interface
// =============================================================================

// ServeHTTP handles incoming HTTP requests
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Get context from pool (performance optimization)
	c := r.pool.Get().(*Context)
	c.reset(w, req)
	c.pipeline = r.pipeline
	defer r.pool.Put(c)

	// Emit BeforeRequest event
	r.emitEvent(EventBeforeRequest, c)

	// Find and execute matching route
	if err := r.handleRequest(c, req); err != nil {
		r.handleError(c, err)
	}

	// Emit AfterRequest event
	r.emitEvent(EventAfterRequest, c)
}

// handleRequest finds and executes the matching route (KISS: extracted for clarity)
func (r *Router) handleRequest(c *Context, req *http.Request) error {
	reqPath := req.URL.Path

	// Find matching route
	route, params := r.findRoute(req.Method, reqPath)

	if route == nil {
		return r.handleNoMatch(c, reqPath)
	}

	// Set path parameters
	c.Params = params

	// Build and execute middleware chain
	handler := r.buildMiddlewareChain(route)
	return handler(c)
}

// findRoute searches for a matching route (KISS: single responsibility)
func (r *Router) findRoute(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.Method != method {
			continue
		}
		if matched, params := matchPath(route.Path, path); matched {
			return route, params
		}
	}
	return nil, nil
}

// handleNoMatch handles 404/405 responses (KISS: extracted for clarity)
func (r *Router) handleNoMatch(c *Context, reqPath string) error {
	// Check if path exists with different method (405 vs 404)
	for _, route := range r.routes {
		if matched, _ := matchPath(route.Path, reqPath); matched {
			// Path exists, method doesn't match
			if r.methodNotAllowed != nil {
				return r.methodNotAllowed(c)
			}
			return c.Error(http.StatusMethodNotAllowed, "Method Not Allowed")
		}
	}

	// Path doesn't exist
	if r.notFound != nil {
		return r.notFound(c)
	}
	return c.Error(http.StatusNotFound, "Not Found")
}

// buildMiddlewareChain creates the middleware execution chain (DRY)
func (r *Router) buildMiddlewareChain(route *Route) HandlerFunc {
	handler := route.Handler

	// Apply route-specific middlewares (reverse order)
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	// Apply global middlewares (reverse order)
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	return handler
}

// handleError handles errors from handlers
func (r *Router) handleError(c *Context, err error) {
	c.Set("error", err)
	r.emitEvent(EventError, c)
	if !c.Written() {
		c.InternalServerError(err.Error())
	}
}

// emitEvent safely emits pipeline events
func (r *Router) emitEvent(event EventType, c *Context) {
	if r.pipeline != nil {
		r.pipeline.Emit(event, c)
	}
}

// =============================================================================
// PATH MATCHING - Route pattern matching engine
// =============================================================================

// matchPath matches a route pattern against a request path
// Supports:
//   - Exact matches: /users
//   - Parameters: /users/:id
//   - Wildcards: /static/*filepath
func matchPath(pattern, requestPath string) (bool, map[string]string) {
	params := make(map[string]string)

	// Fast path: exact match
	if pattern == requestPath {
		return true, params
	}

	patternParts := splitPath(pattern)
	pathParts := splitPath(requestPath)

	// Check for wildcard at the end
	hasWildcard, wildcardName := checkWildcard(patternParts)
	if hasWildcard {
		patternParts = patternParts[:len(patternParts)-1]
	}

	// Validate part counts
	if !validatePartCounts(patternParts, pathParts, hasWildcard) {
		return false, nil
	}

	// Match each part
	for i, patternPart := range patternParts {
		if i >= len(pathParts) {
			return false, nil
		}

		pathPart := pathParts[i]

		if strings.HasPrefix(patternPart, ":") {
			// Parameter - capture value
			params[strings.TrimPrefix(patternPart, ":")] = pathPart
		} else if patternPart != pathPart {
			// Literal mismatch
			return false, nil
		}
	}

	// Handle wildcard capture
	if hasWildcard {
		remainingParts := pathParts[len(patternParts):]
		params[wildcardName] = strings.Join(remainingParts, "/")
	}

	return true, params
}

// splitPath splits a path into parts (helper for DRY)
func splitPath(p string) []string {
	return strings.Split(strings.Trim(p, "/"), "/")
}

// checkWildcard checks if pattern ends with wildcard (helper for KISS)
func checkWildcard(parts []string) (bool, string) {
	if len(parts) == 0 {
		return false, ""
	}
	lastPart := parts[len(parts)-1]
	if strings.HasPrefix(lastPart, "*") {
		return true, strings.TrimPrefix(lastPart, "*")
	}
	return false, ""
}

// validatePartCounts validates pattern/path part counts (helper for KISS)
func validatePartCounts(patternParts, pathParts []string, hasWildcard bool) bool {
	if hasWildcard {
		return len(pathParts) >= len(patternParts)
	}
	return len(patternParts) == len(pathParts)
}

// =============================================================================
// ROUTE GROUP - Grouped routes with shared prefix and middleware
// =============================================================================

// RouteGroup represents a group of routes with shared configuration
type RouteGroup struct {
	prefix      string
	middlewares []MiddlewareFunc
	router      *Router
	parent      *RouteGroup
}

// Use adds middleware to the group
func (g *RouteGroup) Use(middlewares ...MiddlewareFunc) *RouteGroup {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// Group creates a nested group
func (g *RouteGroup) Group(prefix string, middlewares ...MiddlewareFunc) *RouteGroup {
	newGroup := &RouteGroup{
		prefix:      g.prefix + prefix,
		middlewares: append(append([]MiddlewareFunc{}, g.middlewares...), middlewares...),
		router:      g.router,
		parent:      g,
	}
	g.router.groups = append(g.router.groups, newGroup)
	return newGroup
}

// addRoute is the internal method for registering routes in a group (DRY)
func (g *RouteGroup) addRoute(method, routePath string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	fullPath := g.prefix + routePath
	allMiddlewares := append(append([]MiddlewareFunc{}, g.middlewares...), middlewares...)
	return g.router.addRoute(method, fullPath, handler, allMiddlewares...)
}

// HTTP method shortcuts (all delegate to addRoute - DRY)

func (g *RouteGroup) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodGet, path, handler, middlewares...)
}

func (g *RouteGroup) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodPost, path, handler, middlewares...)
}

func (g *RouteGroup) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodPut, path, handler, middlewares...)
}

func (g *RouteGroup) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodDelete, path, handler, middlewares...)
}

func (g *RouteGroup) PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodPatch, path, handler, middlewares...)
}

func (g *RouteGroup) OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodOptions, path, handler, middlewares...)
}

func (g *RouteGroup) HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return g.addRoute(http.MethodHead, path, handler, middlewares...)
}

// Any registers a route for all standard HTTP methods
func (g *RouteGroup) Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	for _, method := range AllHTTPMethods {
		g.addRoute(method, path, handler, middlewares...)
	}
}

// =============================================================================
// ROUTE METADATA - Fluent API for documentation
// =============================================================================

// Name sets the route name (for documentation)
func (r *Route) Name(name string) *Route {
	r.RouteName = name
	return r
}

// Desc sets the route description (for documentation)
func (r *Route) Desc(description string) *Route {
	r.RouteDescription = description
	return r
}

// Tag adds tags to the route (for documentation)
func (r *Route) Tag(tags ...string) *Route {
	r.RouteTags = append(r.RouteTags, tags...)
	return r
}

// Request sets the request body type (for documentation)
func (r *Route) Request(body any) *Route {
	r.RequestBody = body
	return r
}

// Response sets the response body type (for documentation)
func (r *Route) Response(body any) *Route {
	r.ResponseBody = body
	return r
}
