package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gofuckbiz/poltergeist"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	// Allowed origins ("*" for all)
	AllowOrigins []string
	// Allowed HTTP methods
	AllowMethods []string
	// Allowed headers
	AllowHeaders []string
	// Exposed headers
	ExposeHeaders []string
	// Allow credentials
	AllowCredentials bool
	// Max age for preflight cache (seconds)
	MaxAge int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware with default config
func CORS() poltergeist.MiddlewareFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a CORS middleware with custom config
func CORSWithConfig(config *CORSConfig) poltergeist.MiddlewareFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			origin := c.Header("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				// Set CORS headers
				if len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*" {
					c.SetHeader("Access-Control-Allow-Origin", "*")
				} else {
					c.SetHeader("Access-Control-Allow-Origin", origin)
					c.SetHeader("Vary", "Origin")
				}

				if config.AllowCredentials {
					c.SetHeader("Access-Control-Allow-Credentials", "true")
				}

				if len(config.ExposeHeaders) > 0 {
					c.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
				}

				// Handle preflight request
				if c.Method() == http.MethodOptions {
					c.SetHeader("Access-Control-Allow-Methods", allowMethods)
					c.SetHeader("Access-Control-Allow-Headers", allowHeaders)
					c.SetHeader("Access-Control-Max-Age", maxAge)
					return c.NoContent()
				}
			}

			return next(c)
		}
	}
}

// AllowAllCORS returns a CORS middleware that allows all origins
func AllowAllCORS() poltergeist.MiddlewareFunc {
	return CORSWithConfig(&CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: false,
		MaxAge:           86400,
	})
}
