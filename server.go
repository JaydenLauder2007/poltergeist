package poltergeist

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// =============================================================================
// CONFIGURATION - Server settings
// =============================================================================

// Config holds server configuration options
type Config struct {
	Addr             string        // Server address (default: ":8080")
	ReadTimeout      time.Duration // Read timeout (default: 30s)
	WriteTimeout     time.Duration // Write timeout (default: 30s)
	IdleTimeout      time.Duration // Idle timeout (default: 120s)
	MaxHeaderBytes   int           // Max header bytes (default: 1MB)
	GracefulShutdown bool          // Enable graceful shutdown (default: true)
	ShutdownTimeout  time.Duration // Shutdown timeout (default: 30s)
	TLSCertFile      string        // TLS certificate file
	TLSKeyFile       string        // TLS key file
	DevMode          bool          // Development mode (verbose logging)
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Addr:             ":8080",
		ReadTimeout:      DefaultReadTimeout,
		WriteTimeout:     DefaultWriteTimeout,
		IdleTimeout:      DefaultIdleTimeout,
		MaxHeaderBytes:   DefaultMaxHeaderBytes,
		GracefulShutdown: true,
		ShutdownTimeout:  DefaultShutdownTimeout,
		DevMode:          false,
	}
}

// =============================================================================
// SERVER - Main Poltergeist server
// =============================================================================

// Server represents the Poltergeist HTTP server
type Server struct {
	router     *Router
	config     *Config
	httpServer *http.Server
}

// New creates a new Poltergeist server with default configuration
func New() *Server {
	return &Server{
		router: NewRouter(),
		config: DefaultConfig(),
	}
}

// NewWithConfig creates a new Poltergeist server with custom configuration
func NewWithConfig(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}
	return &Server{
		router: NewRouter(),
		config: config,
	}
}

// =============================================================================
// ACCESSORS - Get internal components
// =============================================================================

// Router returns the underlying router
func (s *Server) Router() *Router {
	return s.router
}

// Config returns the server configuration
func (s *Server) Config() *Config {
	return s.config
}

// Pipeline returns the event pipeline
func (s *Server) Pipeline() *EventPipeline {
	return s.router.Pipeline()
}

// Routes returns all registered routes
func (s *Server) Routes() []*Route {
	return s.router.Routes()
}

// =============================================================================
// MIDDLEWARE - Global middleware management
// =============================================================================

// Use adds global middleware to the server
func (s *Server) Use(middlewares ...MiddlewareFunc) *Server {
	s.router.Use(middlewares...)
	return s
}

// =============================================================================
// ROUTE REGISTRATION - Delegate to router (DRY: single implementation)
// =============================================================================

// Group creates a new route group
func (s *Server) Group(prefix string, middlewares ...MiddlewareFunc) *RouteGroup {
	return s.router.Group(prefix, middlewares...)
}

// HTTP methods - all delegate to router

func (s *Server) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.GET(path, handler, middlewares...)
}

func (s *Server) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.POST(path, handler, middlewares...)
}

func (s *Server) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.PUT(path, handler, middlewares...)
}

func (s *Server) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.DELETE(path, handler, middlewares...)
}

func (s *Server) PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.PATCH(path, handler, middlewares...)
}

func (s *Server) OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.OPTIONS(path, handler, middlewares...)
}

func (s *Server) HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	return s.router.HEAD(path, handler, middlewares...)
}

func (s *Server) Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.Any(path, handler, middlewares...)
}

// Static serves static files from a directory
func (s *Server) Static(urlPath, dirPath string) {
	s.router.Static(urlPath, dirPath)
}

// NotFound sets the custom 404 handler
func (s *Server) NotFound(handler HandlerFunc) *Server {
	s.router.NotFound(handler)
	return s
}

// MethodNotAllowed sets the custom 405 handler
func (s *Server) MethodNotAllowed(handler HandlerFunc) *Server {
	s.router.MethodNotAllowed(handler)
	return s
}

// =============================================================================
// SERVER LIFECYCLE - Start, stop, and manage server
// =============================================================================

// Run starts the server (blocking)
func (s *Server) Run(addr ...string) error {
	address := s.resolveAddress(addr)
	s.httpServer = s.createHTTPServer(address)

	s.printBanner(address)
	s.router.pipeline.Emit(EventServerStart, nil)

	if s.config.GracefulShutdown {
		return s.runWithGracefulShutdown()
	}
	return s.startServer()
}

// RunTLS starts the server with TLS
func (s *Server) RunTLS(addr, certFile, keyFile string) error {
	s.config.Addr = addr
	s.config.TLSCertFile = certFile
	s.config.TLSKeyFile = keyFile
	return s.Run(addr)
}

// Shutdown stops the server gracefully
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.router.pipeline.Emit(EventServerStop, nil)
	return s.httpServer.Shutdown(ctx)
}

// =============================================================================
// INTERNAL HELPERS - Private methods for server operations
// =============================================================================

// resolveAddress determines the server address to use
func (s *Server) resolveAddress(addr []string) string {
	if len(addr) > 0 && addr[0] != "" {
		return addr[0]
	}
	return s.config.Addr
}

// createHTTPServer creates the underlying http.Server
func (s *Server) createHTTPServer(address string) *http.Server {
	return &http.Server{
		Addr:           address,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}
}

// startServer starts the HTTP(S) server
func (s *Server) startServer() error {
	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		return s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
	}
	return s.httpServer.ListenAndServe()
}

// runWithGracefulShutdown starts server with graceful shutdown support
func (s *Server) runWithGracefulShutdown() error {
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.startServer(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("⚡ Received signal %v, shutting down gracefully...\n", sig)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	s.router.pipeline.Emit(EventServerStop, nil)

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	log.Println("👻 Server stopped gracefully")
	return nil
}

// printBanner prints the startup banner
func (s *Server) printBanner(addr string) {
	banner := `
   ___      _ _                        _     _   
  / _ \___ | | |_ ___ _ __ __ _  ___(_)___| |_ 
 / /_)/ _ \| | __/ _ \ '__/ _' |/ _ \ / __| __|
/ ___/ (_) | | ||  __/ | | (_| |  __/ \__ \ |_ 
\/    \___/|_|\__\___|_|  \__, |\___|_|___/\__|
                          |___/                 
`
	fmt.Print(banner)
	fmt.Printf("⚡ Poltergeist v%s\n", Version)
	fmt.Printf("👻 Server starting on http://localhost%s\n", addr)
	if s.config.DevMode {
		fmt.Println("🔧 Development mode enabled")
	}
	fmt.Println("────────────────────────────────────────")
}

// =============================================================================
// CONVENIENCE FUNCTIONS - Quick start helpers
// =============================================================================

// Run is a convenience function to create and run a server
func Run(addr string, setup func(*Server)) error {
	server := New()
	setup(server)
	return server.Run(addr)
}
