# Chat Service

Real-time chat microservice for an education platform using HTTP long polling.

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Fiber v2
- **Database**: PostgreSQL
- **Cache**: Redis
- **ORM**: GORM

## Quick Start

### Prerequisites

1. **Install Go 1.21+**: https://go.dev/dl/
2. **Install Docker** (for PostgreSQL & Redis): https://docker.com

### Setup

```bash
# 1. Start databases
docker-compose up -d

# 2. Download dependencies
go mod tidy

# 3. Copy environment file
cp .env.example .env
# Edit .env with your configuration

# 4. Run the service
go run ./cmd/server/main.go
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server port (default: 6004) |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_ACCESS_SECRET` | Shared JWT secret with auth-service |
| `INTERNAL_SERVICE_SECRET` | Secret for service-to-service calls |
| `NOTIFICATION_SERVICE_URL` | URL of notification-service |
| `CLOUDINARY_*` | Cloudinary credentials for media uploads |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/conversations` | Create group chat |
| POST | `/api/v1/conversations/direct` | Create direct chat |
| GET | `/api/v1/conversations` | Get user's conversations |
| POST | `/api/v1/conversations/:id/messages` | Send message |
| GET | `/api/v1/conversations/:id/poll` | Long poll for messages |
| POST | `/api/v1/typing` | Set typing indicator |
| POST | `/api/v1/media/presign` | Get upload URL |

## Architecture

```
cmd/server/main.go      # Entry point
internal/
├── config/             # Configuration
├── handlers/           # HTTP handlers
├── middleware/         # Auth middleware
├── models/             # GORM models
├── repositories/       # Database layer
├── router/             # Routes
└── services/           # Business logic
pkg/
├── cache/              # Redis client
└── database/           # PostgreSQL connection
migrations/             # SQL schema
```
