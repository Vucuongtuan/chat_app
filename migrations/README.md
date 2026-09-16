# Migrations Directory

The `migrations` directory contains database schema migrations. Migrations are versioned SQL scripts that define and evolve the database structure.

## Purpose

Database migrations enable:
- Version control of database schema
- Reproducible database setup
- Easy rollback of schema changes
- Team collaboration on database changes
- Automated deployment of schema changes

## Migration Strategy

The application uses GORM's AutoMigrate feature combined with manual migration files for complex changes.

## Automatic Migrations (GORM)

GORM automatically creates/updates tables on startup:

```go
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &model.User{},
        &model.Conversation{},
        &model.Message{},
        &model.ConversationMember{},
    )
}
```

## Manual Migration Files

For complex operations, create manual migration files:

### Naming Convention

```
<timestamp>_<description>.sql
Example: 20240914_create_users_table.sql
         20240914_add_avatar_column.sql
```

### Timestamp Format

Use Unix timestamp or sequence: `YYYYMMDDHHMMSS`

## Migration Examples

### Create Table
```sql
-- 20240914_create_users_table.sql
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    avatar VARCHAR(255),
    bio TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_email (email),
    INDEX idx_username (username),
    INDEX idx_deleted_at (deleted_at)
);
```

### Add Column
```sql
-- 20240914_add_last_seen_to_users.sql
ALTER TABLE users ADD COLUMN last_seen_at TIMESTAMP NULL;
```

### Create Index
```sql
-- 20240914_create_conversation_index.sql
CREATE INDEX idx_user_id_created_at 
ON conversations(user_id, created_at DESC);
```

### Create Foreign Key
```sql
-- 20240914_add_message_constraints.sql
ALTER TABLE messages
ADD CONSTRAINT fk_messages_conversation_id
FOREIGN KEY (conversation_id) REFERENCES conversations(id);
```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    avatar VARCHAR(255),
    bio TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
```

### Conversations Table
```sql
CREATE TABLE conversations (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    is_group_chat BOOLEAN DEFAULT FALSE,
    owner_id VARCHAR(36) NOT NULL,
    last_message_at TIMESTAMP NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);
```

### Messages Table
```sql
CREATE TABLE messages (
    id VARCHAR(36) PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL,
    sender_id VARCHAR(36) NOT NULL,
    content TEXT NOT NULL,
    attachments JSON,
    edited_at TIMESTAMP NULL,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (sender_id) REFERENCES users(id),
    INDEX idx_conversation_created (conversation_id, created_at DESC)
);
```

### Conversation Members Table
```sql
CREATE TABLE conversation_members (
    id VARCHAR(36) PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY unique_member (conversation_id, user_id)
);
```

## Migration Execution

### Automatic (Application Startup)
```go
if err := database.Migrate(db); err != nil {
    log.Fatal("Migration failed:", err)
}
```

### Manual Migration Tool
```bash
# Apply migrations
go run ./cmd/migrate/main.go up

# Rollback migrations
go run ./cmd/migrate/main.go down

# Check migration status
go run ./cmd/migrate/main.go status
```

## Best Practices

1. **Naming** - Use clear, descriptive migration names
2. **Idempotency** - Migrations should be safe to run multiple times
   ```sql
   CREATE TABLE IF NOT EXISTS ...
   ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...
   ```

3. **Atomic Operations** - Each migration should be a single logical unit
4. **No Breaking Changes** - Add columns as nullable before making required
5. **Backup Before Migration** - Always backup production database
6. **Test Migrations** - Test on development first
7. **Document Changes** - Include comments explaining why

## Rollback Procedure

1. Create reverse migration script
2. Test on development database
3. Execute on staging
4. Execute on production
5. Keep old migration files for history

### Example Rollback
```sql
-- 20240914_create_users_table_rollback.sql
DROP TABLE IF EXISTS users;
```

## CI/CD Integration

Include migration in deployment pipeline:

```yaml
# Example GitHub Actions
- name: Run Migrations
  run: |
    go run ./cmd/migrate/main.go up
```

## Troubleshooting

### Migration Failed
1. Check error message in logs
2. Verify database connectivity
3. Ensure previous migrations completed
4. Check for locking tables

### Rollback Issue
1. Verify rollback migration syntax
2. Check for foreign key constraints
3. Ensure data backup exists

## Resources

- GORM Migration Guide: https://gorm.io/docs/migration.html
- MySQL Reference: https://dev.mysql.com/doc/
