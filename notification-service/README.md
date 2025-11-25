# Notification Service

A microservice for handling real-time notifications using Redis pub/sub and Server-Sent Events (SSE).

## Features

- **Real-time notifications** via Redis pub/sub
- **Server-Sent Events (SSE)** for live updates
- **Persistent storage** with Prisma and PostgreSQL
- **RESTful API** for publishing notifications
- **Authentication** middleware for user endpoints
- **Pagination** support for notification history

## Architecture

The notification service follows a microservice pattern:

- **Auth Service** → publishes notifications via API calls
- **Notification Service** → handles Redis pub/sub, database storage, and SSE streaming
- **API Gateway** → routes `/api/v1/notifications/*` to notification service
- **Client Applications** → connect to SSE endpoint for real-time updates

## API Endpoints

### Public (Internal Service Use)
- `POST /api/v1/notifications/publish` - Publish a notification (used by other services)

### Authenticated (User Endpoints)
- `GET /api/v1/notifications/stream` - SSE endpoint for real-time notifications
- `GET /api/v1/notifications` - Get paginated notification history
- `PATCH /api/v1/notifications/read` - Mark notifications as read

## Setup

1. Install dependencies:
```bash
npm install
```

2. Set up environment variables in `.env`:
```env
DATABASE_URL="postgresql://username:password@localhost:5432/graduation_db"
REDIS_URL="redis://localhost:6379"
PORT=6003
JWT_SECRET="your-jwt-secret-key"
ALLOWED_ORIGINS="http://localhost:3000,http://localhost:8080"
```

3. Generate Prisma client:
```bash
npm run generate
```

4. Run database migrations:
```bash
npm run migrate
```

5. Start the service:
```bash
npm run dev
```

## Usage Examples

### Publishing a Notification (from another service)
```javascript
const response = await fetch('http://localhost:6003/api/v1/notifications/publish', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    userId: 'user-123',
    type: 'parent_link_request',
    message: 'You have a new parent link request',
    data: { requestId: 'req-456' }
  })
});
```

### Client SSE Connection
```javascript
const eventSource = new EventSource('http://localhost:3000/api/v1/notifications/stream', {
  headers: { 'Authorization': 'Bearer your-jwt-token' }
});

eventSource.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  console.log('New notification:', notification);
};
```

## Redis Channels

The service uses Redis pub/sub with the following channel pattern:
- `notifications:{userId}` - User-specific notification channel

## Database Schema

```sql
model Notification {
  id        String   @id @default(uuid())
  userId    String
  type      String
  data      Json
  read      Boolean  @default(false)
  createdAt DateTime @default(now())

  @@index([userId, createdAt])
  @@index([userId, read])
  @@index([createdAt])
}
```

## Development

- `npm run dev` - Start with nodemon
- `npm run build` - Compile TypeScript
- `npm start` - Start production server
- `npm run generate` - Generate Prisma client
- `npm run migrate` - Run database migrations
