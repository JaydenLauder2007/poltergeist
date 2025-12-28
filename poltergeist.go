// Package poltergeist provides a high-performance, lightweight Go framework
// for building REST API and Realtime applications with WebSocket and SSE support.
//
// Poltergeist focuses on minimal boilerplate, zero-config setup, and built-in
// support for event-driven architecture.
//
// Example usage:
//
//	package main
//
//	import "github.com/poltergeist-framework/poltergeist"
//
//	func main() {
//	    app := poltergeist.New()
//
//	    app.GET("/", func(c *poltergeist.Context) error {
//	        return c.JSON(200, map[string]string{"message": "Hello, Ghost!"})
//	    })
//
//	    app.Run(":8080")
//	}
package poltergeist

// Version is the current version of Poltergeist
const Version = "1.0.0"

// Shortcuts for common types

// H is a shortcut for map[string]any, useful for JSON responses
type H map[string]any

// M is an alias for H
type M = H

// StatusCodes are shortcuts for common HTTP status codes
const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusAccepted            = 202
	StatusNoContent           = 204
	StatusMovedPermanently    = 301
	StatusFound               = 302
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusConflict            = 409
	StatusUnprocessableEntity = 422
	StatusTooManyRequests     = 429
	StatusInternalServerError = 500
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
)

// Default creates a new server with common middleware (Logger, Recovery)
func Default() *Server {
	s := New()
	// Middleware will be applied via the middleware package
	return s
}

// Quick creates and starts a server with minimal setup
func Quick(addr string, routes map[string]HandlerFunc) error {
	s := New()
	for path, handler := range routes {
		s.GET(path, handler)
	}
	return s.Run(addr)
}

