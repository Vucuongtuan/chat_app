# Middleware Package

The `middleware` package contains HTTP middleware functions that process requests before they reach handlers. Middleware provides cross-cutting concerns like authentication, logging, and CORS.

## Files

### `auth.go`
Implements authentication middleware.

**Functions:**
- `JWTAuth()` - Verify JWT token in Authorization header
- `RequireAuth()` - Ensure user is authenticated
- `RequireAdmin()` - Ensure user has admin role
- `OptionalAuth()` - Authenticate if token provided, otherwise allow

**Responsibilities:**
- Extract JWT token from Authorization header
- Validate token signature and expiration
- Extract user claims from token
- Return 401 if token invalid
- Attach user info to request context

**Usage:**
```go
router.POST("/messages", middleware.JWTAuth(), handler.SendMessage)
```

**Header Format:**
```
Authorization: Bearer <jwt_token>
```

### `cors.go`
Implements Cross-Origin Resource Sharing (CORS) middleware.

**Functions:**
- `CORS()` - Handle CORS requests
- `AllowOrigin(origin string) bool` - Validate allowed origins

**Responsibilities:**
- Check request origin against allowed list
- Set CORS headers in response
- Handle preflight OPTIONS requests
- Allow configured headers and methods

**CORS Headers Set:**
- `Access-Control-Allow-Origin` - Allowed origin
- `Access-Control-Allow-Methods` - GET, POST, PUT, DELETE, OPTIONS
- `Access-Control-Allow-Headers` - Content-Type, Authorization
- `Access-Control-Max-Age` - Preflight cache duration

**Configuration:**
Uses `CORS_ALLOWED_ORIGINS` from environment:
```
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

### `logger.go`
Implements request logging middleware.

**Functions:**
- `Logger()` - Log incoming requests and responses
- `RequestID()` - Generate and attach unique request ID

**Logs:**
- Request method and path
- Query parameters
- Response status code
- Response time in milliseconds
- Request ID for tracing

**Example Log:**
```
[INFO] GET /api/users - 200 - 45ms - req_id=abc123def456
```

**Context Values:**
- `request_id` - Unique request identifier
- `user_id` - Authenticated user ID (if authenticated)

## Middleware Order

Middleware should be registered in this order:

```go
// 1. Logging - log all requests
router.Use(middleware.Logger())

// 2. Request ID - attach trace ID
router.Use(middleware.RequestID())

// 3. CORS - handle cross-origin
router.Use(middleware.CORS())

// 4. Error recovery - catch panics
router.Use(gin.Recovery())
```

## Using Middleware

### Global Middleware
```go
router.Use(middleware.Logger())
router.Use(middleware.CORS())
```

### Route-specific Middleware
```go
router.POST("/messages", middleware.JWTAuth(), handler.SendMessage)
router.GET("/admin/users", middleware.RequireAdmin(), handler.ListUsers)
```

### Conditional Middleware
```go
router.GET("/public", handler.PublicHandler)
router.GET("/private", middleware.JWTAuth(), handler.PrivateHandler)
```

## Context Utilities

Middleware stores values in request context:

```go
// In middleware
c.Set("user_id", user.ID)
c.Set("request_id", requestID)

// In handler
userID := c.GetString("user_id")
requestID := c.GetString("request_id")
```

## Error Handling

Middleware should not abort all requests on error. Follow patterns:

```go
if err := validateToken(token); err != nil {
    c.JSON(401, gin.H{"error": "invalid_token"})
    c.Abort()
    return
}
c.Next()
```
