package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/gofuckbiz/poltergeist"
)

// RecoveryConfig holds recovery middleware configuration
type RecoveryConfig struct {
	// Print stack trace
	PrintStack bool
	// Stack trace size (default: 4096)
	StackSize int
	// Custom logger
	Logger *log.Logger
	// Custom recovery handler
	RecoveryHandler func(c *poltergeist.Context, err interface{})
	// Enable HTML error page in development
	EnableDevPage bool
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		PrintStack:    true,
		StackSize:     4096,
		Logger:        log.Default(),
		EnableDevPage: false,
	}
}

// Recovery returns a recovery middleware with default config
func Recovery() poltergeist.MiddlewareFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig())
}

// RecoveryWithConfig returns a recovery middleware with custom config
func RecoveryWithConfig(config *RecoveryConfig) poltergeist.MiddlewareFunc {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return func(next poltergeist.HandlerFunc) poltergeist.HandlerFunc {
		return func(c *poltergeist.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// Get stack trace
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, false)
					stackStr := string(stack[:length])

					// Log the panic
					if config.PrintStack {
						config.Logger.Printf("[PANIC RECOVERED] %v\n%s", r, stackStr)
					} else {
						config.Logger.Printf("[PANIC RECOVERED] %v", r)
					}

					// Custom recovery handler
					if config.RecoveryHandler != nil {
						config.RecoveryHandler(c, r)
						return
					}

					// Default error response
					if config.EnableDevPage {
						// Development error page
						html := formatDevErrorPage(r, stackStr)
						c.HTML(http.StatusInternalServerError, html)
					} else {
						// Production error response
						c.JSON(http.StatusInternalServerError, map[string]string{
							"error": "Internal Server Error",
						})
					}
				}
			}()

			return next(c)
		}
	}
}

// formatDevErrorPage formats a development error page
func formatDevErrorPage(err interface{}, stack string) string {
	// Format stack trace for HTML
	stackLines := strings.Split(stack, "\n")
	var formattedStack strings.Builder
	for _, line := range stackLines {
		formattedStack.WriteString(fmt.Sprintf("<div class=\"stack-line\">%s</div>", line))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Poltergeist - Error</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
            background: #1a1a2e;
            color: #eee;
            padding: 40px;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { 
            color: #e94560;
            font-size: 2.5rem;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        h1::before { content: '👻'; font-size: 3rem; }
        .error-box {
            background: #16213e;
            border-left: 4px solid #e94560;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
        }
        .error-message {
            font-size: 1.3rem;
            color: #ff6b6b;
            font-family: monospace;
        }
        h2 { 
            color: #0f3460;
            background: #e94560;
            padding: 10px 15px;
            border-radius: 5px;
            margin-bottom: 15px;
        }
        .stack-trace {
            background: #0f0f1a;
            padding: 20px;
            border-radius: 8px;
            overflow-x: auto;
            font-family: 'Fira Code', 'Consolas', monospace;
            font-size: 0.9rem;
            line-height: 1.6;
        }
        .stack-line { 
            padding: 2px 0;
            color: #888;
        }
        .stack-line:nth-child(odd) { color: #aaa; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Panic Recovered</h1>
        <div class="error-box">
            <div class="error-message">%v</div>
        </div>
        <h2>Stack Trace</h2>
        <div class="stack-trace">%s</div>
    </div>
</body>
</html>`, err, formattedStack.String())
}
