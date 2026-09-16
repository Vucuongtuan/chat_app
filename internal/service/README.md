# Service Package

The `service` package contains business logic for the application. Services orchestrate operations between handlers and repositories, implementing domain rules and workflows.

## Files

### `auth_service.go`
Implements authentication and authorization business logic.

**Functions:**
- `Register(ctx context.Context, req RegisterRequest) (*User, error)` - Register new user
- `Login(ctx context.Context, email, password string) (*AuthToken, error)` - Authenticate user
- `ValidateToken(token string) (*User, error)` - Validate JWT token
- `RefreshToken(ctx context.Context, refreshToken string) (*AuthToken, error)` - Issue new token
- `HashPassword(password string) string` - Hash password securely
- `VerifyPassword(hash, password string) bool` - Verify password

**Business Rules:**
- Email must be unique
- Password minimum 8 characters
- Passwords are hashed using bcrypt
- JWT tokens expire after configured time

### `user_service.go`
Implements user management business logic.

**Functions:**
- `GetUser(ctx context.Context, userID string) (*User, error)` - Retrieve user
- `UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error)` - Update user info
- `DeleteUser(ctx context.Context, userID string) error` - Delete user account
- `ListUsers(ctx context.Context, limit, offset int) ([]*User, error)` - List users
- `SearchUsers(ctx context.Context, query string) ([]*User, error)` - Search users
- `UserExists(ctx context.Context, email string) bool` - Check if user exists

**Business Rules:**
- User email is immutable
- Username must be unique and alphanumeric
- Profile updates are validated
- Deleting user also deletes their messages

### `chat_service.go`
Implements chat and messaging business logic.

**Functions:**
- `SendMessage(ctx context.Context, req SendMessageRequest) (*Message, error)` - Send message
- `GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)` - Get messages
- `GetConversations(ctx context.Context, userID string) ([]*Conversation, error)` - Get conversations
- `CreateConversation(ctx context.Context, req CreateConversationRequest) (*Conversation, error)` - Create conversation
- `DeleteMessage(ctx context.Context, messageID string) error` - Delete message
- `ArchiveConversation(ctx context.Context, conversationID string) error` - Archive conversation

**Business Rules:**
- Messages can only be sent to existing conversations
- Only conversation participants can read messages
- Only message author can delete their message
- Messages are never truly deleted (soft delete)
- Conversations with 2+ participants are group chats

## Service Layer Responsibilities

1. **Validate Input** - Ensure data meets business requirements
2. **Implement Business Logic** - Apply domain rules
3. **Coordinate Repositories** - Fetch/save data
4. **Handle Errors** - Convert repository errors to business errors
5. **Maintain Data Consistency** - Manage transactions

## Dependency Injection Example

```go
type UserService struct {
    repo UserRepository
    log  Logger
}

func NewUserService(repo UserRepository, log Logger) *UserService {
    return &UserService{
        repo: repo,
        log:  log,
    }
}
```

## Error Handling

Services define specific error types:
- `ErrNotFound` - Resource not found
- `ErrUnauthorized` - User not authorized
- `ErrValidation` - Input validation failed
- `ErrDuplicate` - Resource already exists
- `ErrInternal` - Internal server error
