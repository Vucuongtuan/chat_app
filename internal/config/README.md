# Config Package

The `config` package handles application configuration loading and management. It reads environment variables and other configuration sources to initialize the application.

## Files

### `config.go`
Main configuration file that loads environment variables and application settings.

**Responsibilities:**
- Load `.env` file and environment variables
- Parse and validate configuration
- Expose configuration values to the application
- Handle default values

**Config Structure:**
```go
type Config struct {
    Database DatabaseConfig
    Server   ServerConfig
    JWT      JWTConfig
    CORS     CORSConfig
}
```

**Usage Example:**
```go
cfg := config.Load()
db_host := cfg.Database.Host
port := cfg.Server.Port
```

## Environment Variables

Required variables (see `.env.example`):
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `SERVER_PORT` - Server listening port
- `SERVER_ENV` - Environment (development/production)
- `JWT_SECRET` - JWT signing secret
- `JWT_EXPIRE` - JWT token expiration time
- `CORS_ALLOWED_ORIGINS` - Comma-separated allowed origins
