# API Gateway

A modular, production-ready API Gateway built with Express and TypeScript. This gateway routes requests to downstream microservices (auth-service, notification-service) while providing security, health monitoring, and request management features.

## Table of Contents

- [Architecture](#architecture)
- [Features](#features)
- [Directory Structure](#directory-structure)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
- [API Endpoints](#api-endpoints)

## Architecture

The API Gateway follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                          │
├─────────────────────────────────────────────────────────────┤
│  Middleware Layer                                           │
│  ┌──────────┐ ┌─────────┐ ┌──────┐ ┌──────┐ ┌─────────┐  │
│  │Compression│→│ Timeout │→│ CORS │→│Arcjet│→│  Routes │  │
│  └──────────┘ └─────────┘ └──────┘ └──────┘ └─────────┘  │
├─────────────────────────────────────────────────────────────┤
│  Routing Layer                                              │
│  ┌────────────────┐  ┌──────────────────────────────────┐ │
│  │ Health Check   │  │      Proxy Routes                │ │
│  │ /health        │  │  /api/v1/notifications → notify  │ │
│  └────────────────┘  │  /api/v1/location → notify       │ │
│                      │  /* → auth-service               │ │
│                      └──────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────┴─────────────────────┐
        ↓                                           ↓
┌───────────────────┐                    ┌──────────────────────┐
│  Auth Service     │                    │ Notification Service │
│  :6001            │                    │  :6003               │
└───────────────────┘                    └──────────────────────┘
```

### Key Design Principles

1. **Separation of Concerns**: Each module has a single, well-defined responsibility
2. **Dependency Injection**: External dependencies are injected for testability
3. **Configuration as Code**: All configuration centralized and validated at startup
4. **Fail Fast**: Invalid configuration causes immediate startup failure with clear errors
5. **Testability First**: All modules designed to be unit testable in isolation

## Features

### Security
- **Bot Detection**: Blocks malicious bots while allowing search engines and monitors (via Arcjet)
- **VPN/Proxy Blocking**: Blocks requests from VPNs, proxies, and hosting providers (via Arcjet)
- **CORS Protection**: Configurable origin whitelist with support for development wildcard
- **Request Timeout**: 30-second timeout to prevent hanging requests

### Performance
- **Response Compression**: Gzip compression for responses > 1KB
- **Parallel Health Checks**: Concurrent upstream service health verification
- **Efficient Routing**: Priority-based route matching

### Monitoring
- **Health Check Endpoint**: `/health` endpoint with upstream service status
- **Latency Measurement**: Response time tracking for each upstream service
- **Error Logging**: Comprehensive error logging with stack traces and context

### Reliability
- **Graceful Error Handling**: Consistent error responses with proper HTTP status codes
- **Fail Open**: Arcjet errors don't block requests
- **Timeout Protection**: Prevents processing of timed-out requests

## Directory Structure

```
api-gateway/
├── src/
│   ├── config/
│   │   └── index.ts              # Configuration loading and validation
│   ├── middleware/
│   │   ├── index.ts              # Middleware composition and ordering
│   │   ├── cors.middleware.ts    # CORS configuration
│   │   ├── compression.middleware.ts  # Response compression
│   │   ├── timeout.middleware.ts # Request timeout handling
│   │   ├── arcjet.middleware.ts  # Bot and VPN detection
│   │   └── error.middleware.ts   # Centralized error handling
│   ├── services/
│   │   └── health.service.ts     # Health check logic
│   ├── routes/
│   │   └── index.ts              # Proxy route configuration
│   ├── app.ts                    # Express app setup
│   └── main.ts                   # Application entry point
├── tests/
│   ├── helpers/
│   │   └── mocks.ts              # Test utilities and mocks
│   ├── middleware/               # Middleware unit tests
│   ├── services/                 # Service unit tests
│   ├── routes/                   # Route unit tests
│   └── app.test.ts               # Integration tests
├── .env                          # Environment variables (not in git)
├── .env.example                  # Example environment variables
├── jest.config.cjs               # Jest configuration
├── tsconfig.json                 # TypeScript configuration
└── package.json                  # Dependencies and scripts
```

## Configuration

### Environment Variables

Create a `.env` file in the `api-gateway` directory with the following variables:

```bash
# Server Configuration
PORT=6000                          # Port for the API Gateway
NODE_ENV=development               # Environment: development, production, or test

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173  # Comma-separated list of allowed origins
# Use * for development to allow all origins

# Upstream Services
AUTH_SERVICE_URL=http://localhost:6001           # Auth service URL
NOTIFICATION_SERVICE_URL=http://localhost:6003   # Notification service URL

# Security (Optional)
ARCJET_KEY=your_arcjet_api_key_here  # Arcjet API key (optional, only used in production)
```

### Configuration Validation

The gateway validates all configuration at startup:
- `PORT` must be a number between 1-65535
- `NODE_ENV` must be one of: development, production, test
- `ALLOWED_ORIGINS` must be non-empty
- Service URLs must be valid HTTP/HTTPS URLs

Missing or invalid configuration will cause the gateway to exit with a descriptive error message.

### Default Values

- `NODE_ENV`: defaults to "development"
- `AUTH_SERVICE_URL`: defaults to "http://localhost:6001"
- `NOTIFICATION_SERVICE_URL`: defaults to "http://localhost:6003"
- Arcjet protection: automatically disabled in development, enabled in production (if key provided)

## Development

### Prerequisites

- Node.js 18+ and npm
- Running instances of auth-service and notification-service (for full functionality)

### Installation

```bash
cd api-gateway
npm install
```

### Running the Gateway

```bash
# Development mode with auto-reload
npm run dev

# Production mode
npm start

# Build TypeScript
npm run build
```

The gateway will start on the port specified in your `.env` file (default: 6000).

### Development Tips

1. **Use wildcard CORS in development**: Set `ALLOWED_ORIGINS=*` to allow all origins
2. **Disable Arcjet in development**: Don't set `ARCJET_KEY` to skip bot/VPN protection
3. **Check health endpoint**: Visit `http://localhost:6000/health` to verify upstream services

## Testing

### Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

### Test Coverage

The project maintains 80%+ code coverage with:
- **Unit tests**: Test individual modules in isolation
- **Property-based tests**: Test universal properties across many inputs (using fast-check)
- **Integration tests**: Test the full application setup

### Test Structure

```
tests/
├── middleware/          # Middleware unit tests
│   ├── cors.middleware.test.ts
│   ├── compression.middleware.test.ts
│   ├── timeout.middleware.test.ts
│   ├── arcjet.middleware.test.ts
│   └── error.middleware.test.ts
├── services/            # Service unit tests
│   └── health.service.test.ts
├── routes/              # Route unit tests
│   └── routes.test.ts
└── app.test.ts          # Integration tests
```

### Writing Tests

All tests use Jest with TypeScript support. Mock utilities are provided in `tests/helpers/mocks.ts`:

```typescript
import { mockRequest, mockResponse } from '../helpers/mocks';

test('example test', () => {
  const req = mockRequest({ method: 'GET', path: '/health' });
  const res = mockResponse();
  // ... test logic
});
```

## Deployment

### Production Checklist

1. **Set environment variables**:
   ```bash
   NODE_ENV=production
   PORT=6000
   ALLOWED_ORIGINS=https://yourdomain.com
   AUTH_SERVICE_URL=https://auth.yourdomain.com
   NOTIFICATION_SERVICE_URL=https://notifications.yourdomain.com
   ARCJET_KEY=your_production_arcjet_key
   ```

2. **Build the application**:
   ```bash
   npm run build
   ```

3. **Run tests**:
   ```bash
   npm test
   ```

4. **Start the server**:
   ```bash
   npm start
   ```

### Docker Deployment

```dockerfile
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build TypeScript
RUN npm run build

# Expose port
EXPOSE 6000

# Start the application
CMD ["npm", "start"]
```

### Health Monitoring

Monitor the `/health` endpoint for service health:

```bash
curl http://localhost:6000/health
```

Response format:
```json
{
  "status": "ok",
  "service": "api-gateway",
  "upstreams": {
    "auth-service": {
      "name": "auth-service",
      "status": "ok",
      "latency": 45
    },
    "notification-service": {
      "name": "notification-service",
      "status": "ok",
      "latency": 32
    }
  },
  "timestamp": "2024-01-19T12:00:00.000Z"
}
```

### Graceful Shutdown

The gateway handles SIGTERM and SIGINT signals for graceful shutdown:

```bash
# Graceful shutdown
kill -SIGTERM <pid>

# Or use Ctrl+C
```

## API Endpoints

### Health Check

```
GET /health
```

Returns the health status of the gateway and all upstream services.

**Response (200 OK)**:
```json
{
  "status": "ok",
  "service": "api-gateway",
  "upstreams": {
    "auth-service": { "name": "auth-service", "status": "ok", "latency": 45 },
    "notification-service": { "name": "notification-service", "status": "ok", "latency": 32 }
  },
  "timestamp": "2024-01-19T12:00:00.000Z"
}
```

**Response (503 Service Unavailable)** - When any upstream service is unhealthy:
```json
{
  "status": "error",
  "service": "api-gateway",
  "upstreams": {
    "auth-service": { "name": "auth-service", "status": "error" },
    "notification-service": { "name": "notification-service", "status": "ok", "latency": 32 }
  },
  "timestamp": "2024-01-19T12:00:00.000Z"
}
```

### Proxy Routes

All other routes are proxied to upstream services:

| Route Pattern | Upstream Service | Description |
|--------------|------------------|-------------|
| `/api/v1/notifications/*` | notification-service | Notification management |
| `/api/v1/location/request` | notification-service | Silent push notifications |
| `/*` | auth-service | Authentication and user management (catch-all) |

**Example**:
```bash
# Proxied to auth-service
curl http://localhost:6000/api/auth/login

# Proxied to notification-service
curl http://localhost:6000/api/v1/notifications
```

### Error Responses

All errors return a consistent JSON format:

```json
{
  "error": "TimeoutError",
  "message": "Request exceeded 30000ms timeout",
  "statusCode": 408,
  "timestamp": "2024-01-19T12:00:00.000Z"
}
```

Common error status codes:
- `400` - Bad Request (validation errors)
- `401` - Unauthorized
- `403` - Forbidden (CORS, Arcjet blocking)
- `404` - Not Found
- `408` - Request Timeout
- `500` - Internal Server Error
- `502` - Bad Gateway (upstream service error)
- `503` - Service Unavailable (health check failure)
- `504` - Gateway Timeout (upstream service timeout)

## Troubleshooting

### Gateway won't start

1. Check environment variables are set correctly
2. Verify PORT is not already in use
3. Check logs for configuration validation errors

### CORS errors

1. Verify origin is in `ALLOWED_ORIGINS` list
2. Use `*` for development to allow all origins
3. Check browser console for specific CORS error

### Upstream service unreachable

1. Check health endpoint: `curl http://localhost:6000/health`
2. Verify upstream service URLs in `.env`
3. Ensure upstream services are running

### Arcjet blocking legitimate requests

1. Check Arcjet logs in console
2. Verify Arcjet rules in `arcjet.middleware.ts`
3. Use `DRY_RUN` mode for testing: modify config in code
4. Disable Arcjet in development by not setting `ARCJET_KEY`

## License

[Your License Here]

## Contributing

[Your Contributing Guidelines Here]
