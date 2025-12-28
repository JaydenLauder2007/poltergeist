package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/poltergeist-framework/poltergeist"
)

// LogFormat defines the log output format
type LogFormat int

const (
	// LogFormatText outputs logs in text format
	LogFormatText LogFormat = iota
	// LogFormatJSON outputs logs in JSON format
	LogFormatJSON
)

// LogConfig holds logging middleware configuration
type LogConfig struct {
	// Output format (text or JSON)
	Format LogFormat
	// Skip certain paths from logging
	SkipPaths []string
	// Custom logger
	Logger *log.Logger
	// Include request body in logs
	IncludeBody bool
	// Include headers in logs
	IncludeHeaders bool
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Format:         LogFormatText,
		SkipPaths:      []string{"/health", "/healthz", "/ping"},
		Logger:         log.Default(),
		IncludeBody:    false,
		IncludeHeaders: false,
	}
}

// Logger returns a logging middleware with default config
func Logger() poltergeist.MiddlewareFunc {
	return LoggerWithConfig(DefaultLogConfig())
}

// LoggerWithConfig returns a logging middleware with custom config
func LoggerWithConfig(config *LogConfig) poltergeist.MiddlewareFunc {
	if config == nil {
		config = DefaultLogConfig()
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			// Skip logging for certain paths
			if skipPaths[c.Path()] {
				return next(c)
			}

			start := time.Now()
			path := c.Path()
			method := c.Method()
			clientIP := c.ClientIP()

			// Execute handler
			err := next(c)

			// Calculate latency
			latency := time.Since(start)

			// Get status code
			statusCode := 200
			if err != nil {
				statusCode = 500
			}

			// Format and log
			if config.Format == LogFormatJSON {
				config.Logger.Printf(`{"time":"%s","method":"%s","path":"%s","status":%d,"latency":"%s","ip":"%s"}`,
					time.Now().Format(time.RFC3339),
					method,
					path,
					statusCode,
					latency,
					clientIP,
				)
			} else {
				statusColor := getStatusColor(statusCode)
				methodColor := getMethodColor(method)
				config.Logger.Printf("%s %s%s%s %s%3d%s %12v | %s",
					time.Now().Format("2006/01/02 - 15:04:05"),
					methodColor, method, colorReset,
					statusColor, statusCode, colorReset,
					latency,
					path,
				)
			}

			return err
		}
	}
}

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
)

func getStatusColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return colorGreen
	case code >= 300 && code < 400:
		return colorCyan
	case code >= 400 && code < 500:
		return colorYellow
	default:
		return colorRed
	}
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return colorBlue
	case "POST":
		return colorCyan
	case "PUT":
		return colorYellow
	case "DELETE":
		return colorRed
	case "PATCH":
		return colorGreen
	default:
		return colorWhite
	}
}

// RequestLogger is a simple request logger that prints to stdout
func RequestLogger() poltergeist.MiddlewareFunc {
	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			start := time.Now()
			err := next(c)
			fmt.Printf("[%s] %s %s - %v\n",
				time.Now().Format("15:04:05"),
				c.Method(),
				c.Path(),
				time.Since(start),
			)
			return err
		}
	}
}

