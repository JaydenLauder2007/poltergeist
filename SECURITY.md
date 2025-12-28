# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### Do NOT

- Open a public GitHub issue for security vulnerabilities
- Disclose the vulnerability publicly before it's fixed

### Do

1. **Open a private security advisory** on GitHub or contact the maintainer with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional)

2. **Wait for response** - We aim to respond within 48 hours

3. **Coordinate disclosure** - Work with us on the timeline for public disclosure

## What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 1 week
- **Fix Timeline**: Depends on severity, typically 1-4 weeks
- **Credit**: We'll credit you in the security advisory (unless you prefer anonymity)

## Security Best Practices

When using Poltergeist, follow these security recommendations:

### Production Configuration

```go
config := &poltergeist.Config{
    ReadTimeout:      15 * time.Second,
    WriteTimeout:     15 * time.Second,
    IdleTimeout:      60 * time.Second,
    MaxHeaderBytes:   1 << 20, // 1MB
    GracefulShutdown: true,
}
```

### Use Security Middleware

```go
app.Use(middleware.Secure())      // Security headers
app.Use(middleware.RateLimit())   // Rate limiting
app.Use(middleware.Recovery())    // Panic recovery
```

### Validate Input

```go
func createUser(c *poltergeist.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.BadRequest("Invalid request")
    }
    // Validate req fields...
}
```

### Use HTTPS in Production

```go
app.RunTLS(":443", "cert.pem", "key.pem")
```

## Known Security Considerations

1. **CORS**: Default CORS middleware allows all origins. Configure appropriately for production.
2. **Rate Limiting**: Default rate limit is 10 req/sec. Adjust based on your needs.
3. **WebSocket Origins**: Default allows all origins. Configure `CheckOrigin` for production.

Thank you for helping keep Poltergeist secure! 🔒

