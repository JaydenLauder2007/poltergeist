// Package poltergeist is a high-performance, lightweight Go framework
// for building REST APIs and Realtime applications with built-in
// WebSocket, SSE, and auto-generated OpenAPI documentation.
//
// # Quick Start
//
// Create a simple REST API server:
//
//	package main
//
//	import "github.com/gofuckbiz/poltergeist"
//
//	func main() {
//	    app := poltergeist.New()
//
//	    app.GET("/", func(c *poltergeist.Context) error {
//	        return c.JSON(200, poltergeist.H{"message": "Hello, Ghost!"})
//	    })
//
//	    app.Run(":8080")
//	}
//
// # Features
//
//   - Zero-config: Minimal boilerplate, one method to start
//   - Realtime: Built-in WebSocket and SSE support
//   - Event Pipeline: Before/after request hooks
//   - Auto Documentation: OpenAPI/Swagger generation
//   - Rich Middleware: Logger, Recovery, CORS, RateLimit, Auth
//
// # Routing
//
// Register routes using HTTP method shortcuts:
//
//	app.GET("/users", listUsers)
//	app.POST("/users", createUser)
//	app.PUT("/users/:id", updateUser)
//	app.DELETE("/users/:id", deleteUser)
//
// Create route groups with shared middleware:
//
//	api := app.Group("/api", middleware.Logger())
//	api.GET("/users", listUsers)
//
// # WebSocket
//
// Create WebSocket endpoints with hub support:
//
//	hub := poltergeist.NewWSHub()
//	go hub.Run()
//
//	app.WebSocketWithHub("/ws/chat", hub, func(conn *poltergeist.WSConn, _ int, msg []byte) {
//	    hub.BroadcastJSON(poltergeist.H{"message": string(msg)})
//	})
//
// # Server-Sent Events
//
// Create SSE streaming endpoints:
//
//	sseHub := poltergeist.NewSSEHub()
//	go sseHub.Run()
//
//	app.SSEWithHub("/sse/events", sseHub, func(c *poltergeist.Context, sse *poltergeist.SSEWriter) {
//	    sse.SendEvent("welcome", poltergeist.H{"message": "Connected!"})
//	})
//
// # Event Pipeline
//
// Add lifecycle hooks to the request pipeline:
//
//	app.Pipeline().BeforeRequest(func(c *poltergeist.Context) {
//	    c.Set("start", time.Now())
//	})
//
//	app.Pipeline().AfterRequest(func(c *poltergeist.Context) {
//	    duration := time.Since(c.MustGet("start").(time.Time))
//	    log.Printf("Request took: %v", duration)
//	})
//
// For more information, visit https://github.com/gofuckbiz/poltergeist
package poltergeist
