package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gofuckbiz/poltergeist"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	// Requests per second
	RPS float64
	// Burst size (max requests in a burst)
	Burst int
	// Key function to identify clients (default: IP-based)
	KeyFunc func(c *poltergeist.Context) string
	// Skip function to bypass rate limiting
	SkipFunc func(c *poltergeist.Context) bool
	// Custom response when rate limited
	LimitHandler func(c *poltergeist.Context) error
	// Cleanup interval for expired limiters
	CleanupInterval time.Duration
	// Expiration time for unused limiters
	ExpirationTime time.Duration
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RPS:   10,
		Burst: 20,
		KeyFunc: func(c *poltergeist.Context) string {
			return c.ClientIP()
		},
		SkipFunc: nil,
		LimitHandler: func(c *poltergeist.Context) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Too Many Requests",
			})
		},
		CleanupInterval: 1 * time.Minute,
		ExpirationTime:  5 * time.Minute,
	}
}

// visitor holds rate limiter and last seen time
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiterStore stores rate limiters per key
type rateLimiterStore struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	config   *RateLimitConfig
}

// newRateLimiterStore creates a new rate limiter store
func newRateLimiterStore(config *RateLimitConfig) *rateLimiterStore {
	store := &rateLimiterStore{
		visitors: make(map[string]*visitor),
		config:   config,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// getVisitor returns or creates a rate limiter for a key
func (s *rateLimiterStore) getVisitor(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, exists := s.visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(s.config.RPS), s.config.Burst)
		s.visitors[key] = &visitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup removes expired visitors
func (s *rateLimiterStore) cleanup() {
	for {
		time.Sleep(s.config.CleanupInterval)
		s.mu.Lock()
		for key, v := range s.visitors {
			if time.Since(v.lastSeen) > s.config.ExpirationTime {
				delete(s.visitors, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimit returns a rate limiting middleware with default config
func RateLimit() poltergeist.MiddlewareFunc {
	return RateLimitWithConfig(DefaultRateLimitConfig())
}

// RateLimitWithConfig returns a rate limiting middleware with custom config
func RateLimitWithConfig(config *RateLimitConfig) poltergeist.MiddlewareFunc {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	store := newRateLimiterStore(config)

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Skip if configured
			if config.SkipFunc != nil && config.SkipFunc(c) {
				return next(c)
			}

			// Get key for this request
			key := config.KeyFunc(c)

			// Get rate limiter for this key
			limiter := store.getVisitor(key)

			// Check if allowed
			if !limiter.Allow() {
				return config.LimitHandler(c)
			}

			return next(c)
		}
	}
}

// RateLimitPerRoute returns a rate limiter specific to a single route
func RateLimitPerRoute(rps float64, burst int) poltergeist.MiddlewareFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too Many Requests",
				})
			}
			return next(c)
		}
	}
}

// SlidingWindowConfig holds sliding window rate limiter config
type SlidingWindowConfig struct {
	// Window size
	Window time.Duration
	// Max requests per window
	MaxRequests int
	// Key function
	KeyFunc func(c *poltergeist.Context) string
}

// slidingWindowStore stores request timestamps
type slidingWindowStore struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	config   *SlidingWindowConfig
}

// SlidingWindowRateLimit implements a sliding window rate limiter
func SlidingWindowRateLimit(config *SlidingWindowConfig) poltergeist.MiddlewareFunc {
	store := &slidingWindowStore{
		requests: make(map[string][]time.Time),
		config:   config,
	}

	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(config.Window)
			store.mu.Lock()
			now := time.Now()
			for key, times := range store.requests {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) < config.Window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(store.requests, key)
				} else {
					store.requests[key] = valid
				}
			}
			store.mu.Unlock()
		}
	}()

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			key := config.KeyFunc(c)
			now := time.Now()

			store.mu.Lock()
			// Clean old requests
			var valid []time.Time
			for _, t := range store.requests[key] {
				if now.Sub(t) < config.Window {
					valid = append(valid, t)
				}
			}

			// Check if rate limited
			if len(valid) >= config.MaxRequests {
				store.mu.Unlock()
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too Many Requests",
				})
			}

			// Add current request
			store.requests[key] = append(valid, now)
			store.mu.Unlock()

			return next(c)
		}
	}
}
