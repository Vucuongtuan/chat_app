# Database Package

The `database` package handles database connection, initialization, and configuration. It provides a centralized place to manage database setup and utilities.

## Files

### `mysql.go`
Implements MySQL database connection and initialization.

**Functions:**
- `NewDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error)` - Initialize database connection
- `Migrate(db *gorm.DB) error` - Run database migrations
- `Health(db *gorm.DB) error` - Check database connectivity
- `Close(db *gorm.DB) error` - Close database connection

**Responsibilities:**
- Create MySQL connection using GORM
- Configure connection pool
- Apply database settings
- Run auto migrations
- Handle connection errors

## Configuration

Database configuration from environment variables:

```go
type DatabaseConfig struct {
    Host     string // DB_HOST
    Port     string // DB_PORT
    User     string // DB_USER
    Password string // DB_PASSWORD
    Name     string // DB_NAME
}
```

## Connection String Format

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True
```

## Connection Pool Settings

```go
sqlDB := db.DB()
sqlDB.SetMaxIdleConns(10)           // Max idle connections
sqlDB.SetMaxOpenConns(100)          // Max open connections
sqlDB.SetConnMaxLifetime(time.Hour) // Connection lifetime
```

## GORM Configuration

```go
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
    NowFunc: func() time.Time {
        return time.Now().UTC()
    },
    Logger: logger.Default.LogMode(logger.Info),
})
```

## Auto Migrations

GORM automatically creates tables based on model structs:

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

## Usage Example

```go
// In main.go
cfg := config.Load()
db, err := database.NewDatabase(cfg.Database)
if err != nil {
    log.Fatal("Failed to connect database:", err)
}

// Run migrations
if err := database.Migrate(db); err != nil {
    log.Fatal("Failed to run migrations:", err)
}

// Check health
if err := database.Health(db); err != nil {
    log.Fatal("Database unhealthy:", err)
}

defer database.Close(db)

// Use db in repositories
userRepo := repository.NewUserRepository(db)
```

## Error Handling

Common database errors:

- `Connection refused` - Database server not running
- `Access denied` - Invalid credentials
- `Unknown database` - Database doesn't exist
- `Deadlock detected` - Concurrent transaction conflict
- `MySQL has gone away` - Connection lost

## Health Check

Health check endpoint verifies database connectivity:

```go
func Health(db *gorm.DB) error {
    sqlDB := db.DB()
    return sqlDB.Ping()
}
```

## Query Logging

Enable query logging in development:

```go
db = db.Session(&gorm.Session{
    Logger: logger.Default.LogMode(logger.Info),
})
```

Logs all SQL queries executed:
```
[0.154ms] SELECT * FROM users WHERE id = ?
```

## Transaction Support

GORM handles transactions:

```go
tx := db.BeginTx(ctx, nil)
if err := tx.Create(&user).Error; err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

## Performance Tips

1. Use indexes on frequently queried columns
2. Batch operations when possible
3. Use projection (SELECT specific columns)
4. Implement connection pooling
5. Regular query optimization
