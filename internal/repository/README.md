# Repository Package

The `repository` package handles all database operations. It provides an abstraction layer between services and the database, enabling easy switching of database implementations.

## Files

### `user_repository.go`
Implements database operations for user data.

**Functions:**
- `Create(ctx context.Context, user *User) error` - Insert new user
- `GetByID(ctx context.Context, id string) (*User, error)` - Get user by ID
- `GetByEmail(ctx context.Context, email string) (*User, error)` - Get user by email
- `GetByUsername(ctx context.Context, username string) (*User, error)` - Get user by username
- `Update(ctx context.Context, user *User) error` - Update user record
- `Delete(ctx context.Context, id string) error` - Delete user (soft delete)
- `List(ctx context.Context, limit, offset int) ([]*User, error)` - List all users
- `Search(ctx context.Context, query string) ([]*User, error)` - Search users

**Database Queries:**
- CREATE/INSERT user
- SELECT by id, email, username
- UPDATE user fields
- DELETE (mark as deleted)
- WHERE email/username LIKE pattern

### `chat_repository.go`
Implements database operations for messages and conversations.

**Functions:**
- `CreateMessage(ctx context.Context, msg *Message) error` - Insert message
- `GetMessage(ctx context.Context, id string) (*Message, error)` - Get message by ID
- `GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)` - Get messages in conversation
- `DeleteMessage(ctx context.Context, id string) error` - Delete message (soft delete)
- `CreateConversation(ctx context.Context, conv *Conversation) error` - Create conversation
- `GetConversation(ctx context.Context, id string) (*Conversation, error)` - Get conversation
- `GetUserConversations(ctx context.Context, userID string) ([]*Conversation, error)` - Get user's conversations
- `UpdateConversation(ctx context.Context, conv *Conversation) error` - Update conversation

**Database Tables:**
- `users` - User accounts
- `conversations` - Chat conversations
- `messages` - Chat messages
- `conversation_members` - User participation in conversations

## Repository Pattern Benefits

1. **Abstraction** - Decouple business logic from database
2. **Testability** - Mock repositories in tests
3. **Flexibility** - Change database without changing services
4. **Maintainability** - Centralize database logic

## Interface Definition Example

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

## Query Examples

```go
// Using GORM ORM
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

## Context Usage

All repository functions accept `context.Context` for:
- Cancellation support
- Timeout handling
- Request tracing
- Database query deadlines
