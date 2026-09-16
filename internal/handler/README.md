# Handler Package

The `handler` package contains HTTP request handlers. These are the entry points for incoming API requests and are responsible for parsing requests and delegating to service layer.

## Files

### `auth_handler.go`
Handles authentication-related HTTP requests.

**Functions:**
- `Register()` - Register new user
- `Login()` - User login
- `Logout()` - User logout
- `RefreshToken()` - Refresh JWT token

**Example Endpoints:**
```
POST /auth/register
POST /auth/login
POST /auth/logout
POST /auth/refresh
```

### `user_handler.go`
Handles user-related HTTP requests.

**Functions:**
- `GetUser()` - Get user profile by ID
- `UpdateUser()` - Update user information
- `DeleteUser()` - Delete user account
- `GetAllUsers()` - List all users (admin only)
- `SearchUsers()` - Search users by name/email

**Example Endpoints:**
```
GET /users/:id
PUT /users/:id
DELETE /users/:id
GET /users
GET /users/search?q=name
```

### `chat_handler.go`
Handles chat and messaging HTTP requests.

**Functions:**
- `SendMessage()` - Send a message
- `GetMessages()` - Get messages from a conversation
- `GetConversations()` - Get user's conversations
- `CreateConversation()` - Create new conversation
- `DeleteMessage()` - Delete a message

**Example Endpoints:**
```
POST /messages
GET /conversations/:id/messages
GET /conversations
POST /conversations
DELETE /messages/:id
```

## Handler Responsibilities

1. **Parse Requests** - Extract and validate request data
2. **Call Services** - Delegate business logic to service layer
3. **Handle Errors** - Convert service errors to HTTP responses
4. **Return Responses** - Format and send HTTP responses

## Example Handler Structure

```go
func SendMessage(c *gin.Context) {
    // 1. Parse request
    var req CreateMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse(err))
        return
    }

    // 2. Call service
    msg, err := messageService.Send(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, ErrorResponse(err))
        return
    }

    // 3. Return response
    c.JSON(201, SuccessResponse(msg))
}
```

## Middleware Application

Handlers may use middleware for:
- Authentication verification
- Request logging
- CORS handling
- Rate limiting
