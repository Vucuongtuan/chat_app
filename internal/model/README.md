# Model Package

The `model` package defines the core data structures used throughout the application. These models represent the domain entities and are mapped to database tables.

## Files

### `user.go`
Defines the User entity.

**Model Fields:**
- `ID` - Unique user identifier (UUID)
- `Username` - Unique username (alphanumeric)
- `Email` - Unique email address
- `PasswordHash` - Bcrypt hashed password
- `FullName` - User's full name
- `Avatar` - Avatar/profile picture URL
- `Bio` - User biography/status
- `IsActive` - Account activation status
- `CreatedAt` - Account creation timestamp
- `UpdatedAt` - Last profile update timestamp
- `DeletedAt` - Soft delete timestamp

**Database Table:** `users`

**Validations:**
- Email format validation
- Username alphanumeric + underscore only
- Password minimum 8 characters
- No duplicate email/username

**Example:**
```go
type User struct {
    ID           string
    Username     string
    Email        string
    PasswordHash string
    FullName     string
    Avatar       string
    Bio          string
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    *time.Time
}
```

### `message.go`
Defines the Message entity.

**Model Fields:**
- `ID` - Unique message identifier (UUID)
- `ConversationID` - Conversation this message belongs to
- `SenderID` - User who sent the message
- `Content` - Message text content
- `Attachments` - File attachments (JSON array)
- `EditedAt` - Last edit timestamp (NULL if never edited)
- `CreatedAt` - Message sent timestamp
- `DeletedAt` - Soft delete timestamp

**Database Table:** `messages`

**Constraints:**
- SenderID must be participant of conversation
- Content cannot be empty
- Maximum content length: 5000 characters

**Example:**
```go
type Message struct {
    ID             string
    ConversationID string
    SenderID       string
    Content        string
    Attachments    []string
    EditedAt       *time.Time
    CreatedAt      time.Time
    DeletedAt      *time.Time
}
```

### `auth.go` (Optional)
Defines authentication-related models.

**Structures:**
- `AuthToken` - JWT token response
- `LoginRequest` - Login credentials
- `RegisterRequest` - Registration data
- `RefreshTokenRequest` - Token refresh data

**AuthToken Fields:**
- `AccessToken` - JWT access token
- `RefreshToken` - Refresh token
- `ExpiresIn` - Token expiration time in seconds
- `TokenType` - "Bearer"

**Example:**
```go
type AuthToken struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"`
}
```

### `conversation.go`
Defines the Conversation entity.

**Model Fields:**
- `ID` - Unique conversation identifier
- `Name` - Conversation name (NULL for DMs)
- `Description` - Conversation description
- `IsGroupChat` - Whether this is a group conversation
- `Owner` - Creator of conversation
- `Members` - Participants ([]*User)
- `LastMessage` - Last message in conversation
- `LastMessageAt` - Timestamp of last message
- `CreatedAt` - Conversation creation timestamp
- `UpdatedAt` - Last update timestamp
- `DeletedAt` - Soft delete timestamp

**Database Tables:**
- `conversations` - Conversation metadata
- `conversation_members` - Many-to-many user participation

## Model Responsibilities

1. **Data Representation** - Define entity structure
2. **Validation** - Implement field constraints
3. **Serialization** - Convert to/from JSON
4. **Database Mapping** - GORM tags for ORM

## GORM Tags Example

```go
type User struct {
    ID        string `gorm:"column:id;primaryKey"`
    Email     string `gorm:"column:email;uniqueIndex"`
    Username  string `gorm:"column:username;uniqueIndex"`
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}
```

## Relationships

Models define relationships for data associations:
- User ↔ Message (One-to-Many)
- User ↔ Conversation (Many-to-Many)
- Conversation ↔ Message (One-to-Many)
