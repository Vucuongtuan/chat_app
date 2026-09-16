# Router Package

The `router` package handles HTTP route configuration and setup. It defines all API endpoints, applies middleware, and configures the Gin router.

## Files

### `router.go`
Main router configuration file.

**Functions:**
- `NewRouter() *gin.Engine` - Initialize and configure router
- `setupRoutes(router *gin.Engine)` - Register all routes
- `setupMiddleware(router *gin.Engine)` - Apply global middleware

**Responsibilities:**
- Initialize Gin router
- Register middleware
- Define API routes
- Configure error handling

## Route Structure

### Authentication Routes
```
POST   /auth/register              - Register new user
POST   /auth/login                 - User login
POST   /auth/logout                - User logout
POST   /auth/refresh               - Refresh JWT token
```

### User Routes
```
GET    /users/:id                  - Get user profile
PUT    /users/:id                  - Update user profile
DELETE /users/:id                  - Delete user account
GET    /users                      - List all users (admin only)
GET    /users/search?q=name        - Search users
```

### Chat Routes
```
POST   /conversations              - Create conversation
GET    /conversations              - Get user's conversations
GET    /conversations/:id          - Get conversation details
PUT    /conversations/:id          - Update conversation
DELETE /conversations/:id          - Delete conversation

POST   /conversations/:id/messages - Send message
GET    /conversations/:id/messages - Get messages
GET    /messages/:id               - Get message details
PUT    /messages/:id               - Edit message
DELETE /messages/:id               - Delete message
```

### Health Check
```
GET    /health                     - Health check endpoint
GET    /health/readiness           - Readiness probe
```

## Router Setup Example

```go
func NewRouter() *gin.Engine {
    router := gin.Default()

    // Middleware
    setupMiddleware(router)

    // Routes
    setupRoutes(router)

    return router
}

func setupMiddleware(r *gin.Engine) {
    r.Use(middleware.Logger())
    r.Use(middleware.RequestID())
    r.Use(middleware.CORS())
}

func setupRoutes(r *gin.Engine) {
    // Auth
    auth := r.Group("/auth")
    {
        auth.POST("/register", handler.Register)
        auth.POST("/login", handler.Login)
        auth.POST("/logout", middleware.JWTAuth(), handler.Logout)
        auth.POST("/refresh", handler.RefreshToken)
    }

    // Users
    users := r.Group("/users")
    users.Use(middleware.JWTAuth())
    {
        users.GET("/:id", handler.GetUser)
        users.PUT("/:id", handler.UpdateUser)
        users.DELETE("/:id", handler.DeleteUser)
    }

    // Chat
    chat := r.Group("/conversations")
    chat.Use(middleware.JWTAuth())
    {
        chat.POST("", handler.CreateConversation)
        chat.GET("", handler.GetConversations)
        // ... more routes
    }
}
```

## Route Groups

Routes are organized into logical groups with shared middleware:

### Public Routes
- No authentication required
- `/auth/login`, `/auth/register`

### Protected Routes
- Require JWT authentication
- `/users`, `/conversations`, `/messages`

### Admin Routes
- Require admin role
- `/admin/*`

## Middleware Application

Different routes apply different middleware:

```go
// No middleware
r.GET("/health", handler.Health)

// CORS only
r.POST("/auth/login", handler.Login)

// Authentication required
r.GET("/users/me", middleware.JWTAuth(), handler.GetCurrentUser)

// Admin only
r.GET("/admin/users", middleware.RequireAdmin(), handler.ListAllUsers)
```

## Error Handling

Router handles various HTTP status codes:

- `200 OK` - Successful request
- `201 Created` - Resource created
- `204 No Content` - No response body
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing/invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists
- `500 Internal Server Error` - Server error

## Response Format

All endpoints return JSON responses with consistent format:

**Success Response:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Operation successful"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "error_code",
  "message": "Human readable error message"
}
```
