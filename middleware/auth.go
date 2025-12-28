package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/poltergeist-framework/poltergeist"
)

// BasicAuthConfig holds Basic Auth configuration
type BasicAuthConfig struct {
	// Validator function
	Validator func(username, password string, c *poltergeist.Context) bool
	// Realm name
	Realm string
	// Skip function
	SkipFunc func(c *poltergeist.Context) bool
}

// BasicAuth returns a Basic Auth middleware
func BasicAuth(validator func(username, password string, c *poltergeist.Context) bool) poltergeist.MiddlewareFunc {
	return BasicAuthWithConfig(&BasicAuthConfig{
		Validator: validator,
		Realm:     "Restricted",
	})
}

// BasicAuthWithConfig returns a Basic Auth middleware with custom config
func BasicAuthWithConfig(config *BasicAuthConfig) poltergeist.MiddlewareFunc {
	realm := config.Realm
	if realm == "" {
		realm = "Restricted"
	}

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Skip if configured
			if config.SkipFunc != nil && config.SkipFunc(c) {
				return next(c)
			}

			auth := c.Header("Authorization")
			if auth == "" {
				return unauthorized(c, realm)
			}

			// Check Basic prefix
			if !strings.HasPrefix(auth, "Basic ") {
				return unauthorized(c, realm)
			}

			// Decode credentials
			payload, err := base64.StdEncoding.DecodeString(auth[6:])
			if err != nil {
				return unauthorized(c, realm)
			}

			// Split username:password
			pair := strings.SplitN(string(payload), ":", 2)
			if len(pair) != 2 {
				return unauthorized(c, realm)
			}

			// Validate
			if !config.Validator(pair[0], pair[1], c) {
				return unauthorized(c, realm)
			}

			// Store username in context
			c.Set("username", pair[0])

			return next(c)
		}
	}
}

func unauthorized(c *poltergeist.Context, realm string) error {
	c.SetHeader("WWW-Authenticate", `Basic realm="`+realm+`"`)
	return c.JSON(http.StatusUnauthorized, map[string]string{
		"error": "Unauthorized",
	})
}

// BasicAuthWithUsers returns a Basic Auth middleware with a static user map
func BasicAuthWithUsers(users map[string]string) poltergeist.MiddlewareFunc {
	return BasicAuth(func(username, password string, c *poltergeist.Context) bool {
		expectedPassword, ok := users[username]
		if !ok {
			return false
		}
		// Constant time comparison to prevent timing attacks
		return subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1
	})
}

// BearerAuthConfig holds Bearer token auth configuration
type BearerAuthConfig struct {
	// Token validator function
	Validator func(token string, c *poltergeist.Context) bool
	// Skip function
	SkipFunc func(c *poltergeist.Context) bool
	// Error message
	ErrorMessage string
}

// BearerAuth returns a Bearer token auth middleware
func BearerAuth(validator func(token string, c *poltergeist.Context) bool) poltergeist.MiddlewareFunc {
	return BearerAuthWithConfig(&BearerAuthConfig{
		Validator:    validator,
		ErrorMessage: "Invalid or missing token",
	})
}

// BearerAuthWithConfig returns a Bearer token auth middleware with custom config
func BearerAuthWithConfig(config *BearerAuthConfig) poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Skip if configured
			if config.SkipFunc != nil && config.SkipFunc(c) {
				return next(c)
			}

			auth := c.Header("Authorization")
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": config.ErrorMessage,
				})
			}

			// Check Bearer prefix
			if !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": config.ErrorMessage,
				})
			}

			token := auth[7:]

			// Validate token
			if !config.Validator(token, c) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": config.ErrorMessage,
				})
			}

			// Store token in context
			c.Set("token", token)

			return next(c)
		}
	}
}

// APIKeyConfig holds API key auth configuration
type APIKeyConfig struct {
	// Header name (default: X-API-Key)
	HeaderName string
	// Query parameter name (default: api_key)
	QueryName string
	// Validator function
	Validator func(key string, c *poltergeist.Context) bool
	// Skip function
	SkipFunc func(c *poltergeist.Context) bool
}

// APIKeyAuth returns an API key auth middleware
func APIKeyAuth(validator func(key string, c *poltergeist.Context) bool) poltergeist.MiddlewareFunc {
	return APIKeyAuthWithConfig(&APIKeyConfig{
		HeaderName: "X-API-Key",
		QueryName:  "api_key",
		Validator:  validator,
	})
}

// APIKeyAuthWithConfig returns an API key auth middleware with custom config
func APIKeyAuthWithConfig(config *APIKeyConfig) poltergeist.MiddlewareFunc {
	headerName := config.HeaderName
	if headerName == "" {
		headerName = "X-API-Key"
	}
	queryName := config.QueryName
	if queryName == "" {
		queryName = "api_key"
	}

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Skip if configured
			if config.SkipFunc != nil && config.SkipFunc(c) {
				return next(c)
			}

			// Try header first
			key := c.Header(headerName)
			if key == "" {
				// Try query parameter
				key = c.Query(queryName)
			}

			if key == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "API key required",
				})
			}

			// Validate key
			if !config.Validator(key, c) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid API key",
				})
			}

			// Store key in context
			c.Set("api_key", key)

			return next(c)
		}
	}
}

// StaticAPIKey returns an API key middleware with a static key
func StaticAPIKey(validKey string) poltergeist.MiddlewareFunc {
	return APIKeyAuth(func(key string, c *poltergeist.Context) bool {
		return subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1
	})
}

