// Package middleware provides common middleware for Poltergeist framework
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/poltergeist-framework/poltergeist"
)

// Timeout returns a middleware that times out requests
func Timeout(timeout time.Duration) poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Create done channel
			done := make(chan error, 1)

			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				return c.JSON(http.StatusGatewayTimeout, map[string]string{
					"error": "Request Timeout",
				})
			}
		}
	}
}

// Secure adds security headers
func Secure() poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			c.SetHeader("X-Content-Type-Options", "nosniff")
			c.SetHeader("X-Frame-Options", "DENY")
			c.SetHeader("X-XSS-Protection", "1; mode=block")
			c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
			c.SetHeader("Content-Security-Policy", "default-src 'self'")
			return next(c)
		}
	}
}

// gzipWriter wraps response writer for gzip compression
type gzipWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Gzip returns a gzip compression middleware
func Gzip() poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Check if client accepts gzip
			if !strings.Contains(c.Header("Accept-Encoding"), "gzip") {
				return next(c)
			}

			// Create gzip writer
			gz := gzip.NewWriter(c.Writer)
			defer gz.Close()

			c.SetHeader("Content-Encoding", "gzip")
			c.Writer = gzipWriter{Writer: gz, ResponseWriter: c.Writer}

			return next(c)
		}
	}
}

// RequestID adds a unique request ID to each request
func RequestID() poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Check if request already has an ID
			id := c.Header("X-Request-ID")
			if id == "" {
				id = generateRequestID()
			}

			c.SetHeader("X-Request-ID", id)
			c.Set("request_id", id)

			return next(c)
		}
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// Chain combines multiple middlewares into one
func Chain(middlewares ...poltergeist.MiddlewareFunc) poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// If conditionally applies middleware
func If(condition func(c *poltergeist.Context) bool, middleware poltergeist.MiddlewareFunc) poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			if condition(c) {
				return middleware(next)(c)
			}
			return next(c)
		}
	}
}

