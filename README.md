# Chat App

A simple chat application built with Go, Gin, and MySQL.

## Project Structure

```
chatapp/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration
│   ├── handler/                 # HTTP handlers
│   ├── service/                 # Business logic
│   ├── repository/              # Data access layer
│   ├── model/                   # Data models
│   ├── middleware/              # Middleware functions
│   └── router/                  # Router setup
├── pkg/
│   ├── database/                # Database connection
│   ├── jwt/                     # JWT utilities
│   └── response/                # Response utilities
├── migrations/                  # Database migrations
├── .env                         # Environment variables
├── .air.toml                    # Air config (hot reload)
├── Dockerfile                   # Docker image
├── docker-compose.yml           # Docker compose setup
└── README.md                    # This file
```

## Prerequisites

- Go 1.26.5+
- MySQL 8.0+
- Docker & Docker Compose (optional)

## Setup

### 1. Clone the repository
```bash
cd chatapp
```

### 2. Install dependencies
```bash
go mod download
```

### 3. Configure environment variables
```bash
cp .env.example .env
# Edit .env with your configuration
```

### 4. Setup database
```bash
# Create database and run migrations
# (migrations go here)
```

## Development

### Run with air (hot reload)
```bash
air
```

### Run directly
```bash
go run ./cmd/server/main.go
```

### Build
```bash
go build -o bin/main ./cmd/server
```

## Docker

### Build and run with Docker Compose
```bash
docker-compose up -d
```

### Stop containers
```bash
docker-compose down
```

## API Endpoints

(To be documented)

## License

MIT
