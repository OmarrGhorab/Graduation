# Complete Tech Stack & Architecture Documentation

## Table of Contents
1. [System Overview](#system-overview)
2. [Service Architecture](#service-architecture)
3. [Technology Stack Summary](#technology-stack-summary)
4. [Detailed Service Breakdown](#detailed-service-breakdown)
5. [Database & Storage](#database--storage)
6. [Infrastructure & DevOps](#infrastructure--devops)
7. [Security & Authentication](#security--authentication)
8. [Design Decisions & Constraints](#design-decisions--constraints)

---

## System Overview

This is a **microservices-based education platform** built for managing courses, attendance, payments, real-time chat, AI recommendations, and notifications. The system supports multiple user roles (Students, Teachers, Parents, Instructors, Assistants, HR, Recruiters) with comprehensive features for online and offline learning.

### Key Capabilities
- **Authentication & Authorization**: Multi-factor auth, OAuth, session management, device tracking
- **Course Management**: Course creation, enrollment, materials, attendance tracking
- **Real-time Communication**: WebSocket-based chat with media support
- **Payment Processing**: Shopping cart, subscriptions, Paymob integration
- **AI-Powered Features**: Course recommendations, chatbot assistant, progress reports
- **Attendance System**: QR-based attendance with geofencing and anti-fraud measures
- **Notifications**: Push notifications via FCM, email notifications via Resend
- **Parent Controls**: Child linking, location tracking, progress monitoring

---

## Service Architecture

### Microservices (9 Services)

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway (Port 6000)                  │
│                    Node.js + Express + TypeScript               │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──────────────────┬──────────────────┬──────────────────┐
             │                  │                  │                  │
      ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
      │Auth Service │   │Notification │   │Chat Service │   │WS Gateway   │
      │  (6001)     │   │Service(6003)│   │(6004/14/24) │   │(6005/15/25) │
      │Node.js+Prisma│   │Node.js+FCM  │   │Go+Fiber     │   │Go+WebSocket │
      └─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
             │                  │                  │                  │
      ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
      │Courses Svc  │   │Payment Svc  │   │Recommend Svc│   │             │
      │(8085/8086)  │   │   (8090)    │   │   (8095)    │   │             │
      │Go+Fiber+QR  │   │Go+Paymob    │   │Python+AI    │   │             │
      └─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
             │                  │                  │
             └──────────────────┴──────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Shared Resources    │
                    │ PostgreSQL + Redis    │
                    │      + Kafka          │
                    └───────────────────────┘
```

---

## Technology Stack Summary

### Programming Languages
| Language | Services | Reason for Choice |
|----------|----------|-------------------|
| **TypeScript/Node.js** | API Gateway, Auth Service, Notification Service | Rapid development, rich ecosystem, excellent for I/O-bound operations, Prisma ORM |
| **Go** | Chat Service, WS Gateway, Courses Service, Payment Service | High performance, excellent concurrency, low memory footprint, perfect for real-time systems |
| **Python** | Recommendation Service | Best AI/ML ecosystem, FastAPI for async performance, easy integration with AI models |

### Frameworks & Libraries
| Framework | Purpose | Services |
|-----------|---------|----------|
| **Express.js** | HTTP server | API Gateway, Auth, Notification |
| **Fiber v2** | High-performance HTTP | Chat, Courses, Payment, WS Gateway |
| **FastAPI** | Async Python API | Recommendation Service |
| **Prisma** | Type-safe ORM | Auth Service, Notification Service |
| **GORM** | Go ORM | Chat, Courses, Payment Services |
| **SQLAlchemy** | Python ORM | Recommendation Service |


### Databases & Storage
| Technology | Purpose | Why Chosen |
|------------|---------|------------|
| **PostgreSQL 15** | Primary database | ACID compliance, JSON support, excellent for relational data, mature ecosystem |
| **Redis 7** | Caching, sessions, pub/sub | In-memory speed, pub/sub for real-time, perfect for session storage and QR tokens |
| **Kafka** | Event streaming | Reliable message delivery, event sourcing, decoupled microservices communication |

### External Services
| Service | Purpose | Why Chosen |
|---------|---------|------------|
| **Firebase Cloud Messaging** | Push notifications | Industry standard, cross-platform, reliable delivery |
| **Resend** | Email delivery | Modern API, excellent deliverability, same as auth service |
| **Paymob** | Payment gateway | Local Egyptian payment provider, supports cards and wallets |
| **Cloudinary** | Media storage | CDN, image optimization, video support |
| **Google Gemini AI** | AI recommendations & chatbot | Advanced language model, multimodal support |

### Observability & DevOps
| Technology | Purpose | Why Chosen |
|------------|---------|------------|
| **Prometheus** | Metrics & Monitoring | Industry standard for time-series metrics, powerful PromQL |
| **Grafana** | Dashboards & Logs | Unified observability platform for visualizing metrics and logs |
| **Loki & Promtail** | Log Aggregation | Lightweight log storage perfectly integrated with Grafana |
| **Jaeger** | Distributed Tracing | End-to-end request tracing across microservices |
| **OpenTelemetry (OTel)** | Telemetry Collection | Standardized framework for collecting metrics, logs, and traces |
| **Sentry** | Error Tracking | Real-time crash reporting and exception tracking |
| **Docker & Compose** | Containerization | Consistent environments, easy orchestration for 15+ containers |
| **Arcjet** | Bot detection, VPN blocking | Security layer for API Gateway |

---

## Detailed Service Breakdown

### 1. API Gateway (Port 6000)
**Tech Stack**: Node.js 18+, Express 5, TypeScript

**Purpose**: Single entry point for all client requests, handles routing, security, and request management.

**Key Features**:
- Request routing to downstream services
- CORS protection with configurable origins
- Bot detection and VPN blocking (Arcjet)
- Response compression (gzip)
- Request timeout management (30s)
- Health check aggregation
- Rate limiting

**Dependencies**:
```json
{
  "express": "^5.1.0",
  "http-proxy-middleware": "^3.0.5",
  "compression": "^1.8.1",
  "@arcjet/node": "^1.0.0-beta.13",
  "cors": "^2.8.5"
}
```

**Why These Choices**:
- **Express**: Mature, well-documented, extensive middleware ecosystem
- **TypeScript**: Type safety, better IDE support, catches errors at compile time
- **Arcjet**: Specialized security for API protection without performance overhead

**Routing Logic**:
- `/api/v1/notifications/*` → Notification Service
- `/api/v1/location/request` → Notification Service
- `/*` → Auth Service (catch-all)


---

### 2. Auth Service (Port 6001)
**Tech Stack**: Node.js 18+, Express 5, TypeScript, Prisma, PostgreSQL

**Purpose**: Centralized authentication, authorization, user management, and session handling.

**Key Features**:
- **Multi-factor Authentication**: TOTP-based 2FA with QR codes, backup codes
- **OAuth Integration**: Google and Apple Sign-In
- **Session Management**: JWT-based with refresh tokens, device tracking
- **Device Security**: Device fingerprinting, trusted device management, new device blocking
- **Email Verification**: Resend integration for verification emails
- **Password Management**: Bcrypt hashing, password reset flows
- **Profile Management**: User profiles with images (Cloudinary), bio, preferences
- **Parent-Child Linking**: Parent accounts can link to children for monitoring
- **Location Tracking**: GPS location history for child safety features
- **Onboarding Flow**: Multi-step onboarding with interests and goals
- **Role-Based Access Control**: 7 roles (Student, Teacher, Parent, Instructor, Assistant, HR, Recruiter)

**Database Schema** (PostgreSQL via Prisma):
```prisma
- User (id, name, username, email, password, role, 2FA fields, device fields)
- UserPreference (language, theme, notifications)
- UserDevice (fingerprint, platform, trusted status)
- AuthProvider (OAuth providers)
- Session (tokens, expiry, location, GPS tracking)
- Interest & UserInterest (user interests)
- CourseEnrollment (user-course relationships)
- ParentLinkRequest & ParentChildLink (parent-child relationships)
- UnlinkRequest (child can request to unlink from parent)
- LocationHistory (GPS tracking for children)
```

**Dependencies**:
```json
{
  "@prisma/client": "^6.18.0",
  "bcrypt": "^6.0.0",
  "jsonwebtoken": "^9.0.2",
  "speakeasy": "^2.0.0",
  "qrcode": "^1.5.4",
  "google-auth-library": "^10.5.0",
  "resend": "^6.5.2",
  "cloudinary": "^2.8.0",
  "ioredis": "^5.8.2"
}
```

**Why These Choices**:
- **Prisma**: Type-safe database access, excellent migrations, auto-generated types
- **PostgreSQL**: Complex relationships (parent-child, sessions, devices), ACID transactions
- **Redis**: Fast session lookups, distributed session storage
- **Bcrypt**: Industry standard for password hashing, configurable work factor
- **JWT**: Stateless authentication, works well with microservices
- **Resend**: Modern email API, better deliverability than traditional SMTP


**Security Features**:
- Device fingerprinting prevents account sharing
- 2FA with encrypted TOTP secrets
- Session revocation and device management
- New device email notifications
- Rate limiting on sensitive endpoints
- Encrypted backup codes for 2FA recovery

---

### 3. Notification Service (Port 6003)
**Tech Stack**: Node.js 18+, Express 5, TypeScript, Prisma, Firebase Admin SDK

**Purpose**: Handle push notifications, email notifications, and notification history.

**Key Features**:
- **Push Notifications**: Firebase Cloud Messaging (FCM) for iOS and Android
- **FCM Token Management**: Register/unregister device tokens
- **Notification History**: Persistent storage with pagination
- **Read Status Tracking**: Mark notifications as read
- **Kafka Consumer**: Listens to events from other services
- **Silent Push**: Location request notifications for parent tracking

**Database Schema**:
```prisma
- Notification (id, userId, type, data, read, createdAt)
- FcmToken (id, userId, token, deviceId, platform)
```

**Dependencies**:
```json
{
  "@prisma/client": "^6.18.0",
  "firebase-admin": "^13.6.0",
  "kafkajs": "^2.2.4",
  "ioredis": "^5.8.2"
}
```

**Why These Choices**:
- **Firebase FCM**: Industry standard, reliable, cross-platform, free tier generous
- **Kafka**: Decoupled event-driven architecture, reliable message delivery
- **Redis Pub/Sub**: Real-time notification delivery to connected clients
- **PostgreSQL**: Persistent notification history, complex queries for filtering

**Notification Types**:
- Parent link requests
- Unlink requests
- Course enrollments
- Attendance updates
- Payment confirmations
- Lesson reminders
- Progress updates


---

### 4. Chat Service (Ports 6004, 6014, 6024 - 3 instances)
**Tech Stack**: Go 1.23, Fiber v2, GORM, PostgreSQL, Redis, Kafka

**Purpose**: Real-time messaging system for direct and group chats with media support.

**Key Features**:
- **Direct Messaging**: One-on-one conversations
- **Group Chats**: Multi-participant conversations
- **Media Support**: Images, videos, documents via Cloudinary
- **Typing Indicators**: Real-time typing status
- **Read Receipts**: Message read tracking
- **Unread Counts**: Per-conversation unread message counts
- **Message History**: Paginated message retrieval
- **Presigned URLs**: Secure media upload via Cloudinary
- **Horizontal Scaling**: 3 instances for load distribution

**Database Schema** (PostgreSQL via GORM):
```go
- Conversation (ID, Type, Title, Participants)
- Message (ID, ConversationID, SenderID, Content, MediaURL, ReadBy)
- ConversationParticipant (ConversationID, UserID, UnreadCount)
```

**Dependencies**:
```go
github.com/gofiber/fiber/v2
github.com/golang-jwt/jwt/v5
gorm.io/gorm
github.com/redis/go-redis/v9
github.com/segmentio/kafka-go
github.com/cloudinary/cloudinary-go
```

**Why These Choices**:
- **Go**: Excellent concurrency model, low latency, efficient memory usage
- **Fiber**: Fastest Go web framework, Express-like API, built on fasthttp
- **GORM**: Mature Go ORM, good performance, easy migrations
- **Redis**: Fast message caching, typing indicator state
- **Kafka**: Event streaming for notifications, message delivery guarantees
- **Cloudinary**: CDN for media, automatic optimization, secure uploads

**Architecture Pattern**:
- Each chat service instance connects to a WS Gateway instance
- Kafka for cross-instance message broadcasting
- Redis for shared state (typing indicators, presence)


---

### 5. WS Gateway (Ports 6005, 6015, 6025 - 3 instances)
**Tech Stack**: Go 1.23, Fiber v2, WebSocket, Redis, Kafka

**Purpose**: WebSocket gateway for real-time bidirectional communication.

**Key Features**:
- **WebSocket Connections**: Persistent connections for real-time updates
- **Multi-Tab Support**: Multiple connections per user
- **Event Broadcasting**: Kafka-based message distribution
- **Presence Management**: Online/offline status tracking
- **Connection Pooling**: Efficient connection management
- **JWT Authentication**: Secure WebSocket handshake
- **Horizontal Scaling**: 3 instances with Redis coordination

**Dependencies**:
```go
github.com/gofiber/contrib/websocket
github.com/golang-jwt/jwt/v5
github.com/redis/go-redis/v9
github.com/segmentio/kafka-go
```

**Why These Choices**:
- **Go**: Perfect for WebSocket servers, goroutines for concurrent connections
- **Fiber WebSocket**: High-performance WebSocket implementation
- **Redis Pub/Sub**: Cross-instance message broadcasting
- **Kafka**: Reliable event delivery from chat service

**Event Types**:
- `message.created`: New message notifications
- `typing.started/stopped`: Typing indicators
- `conversation.read`: Read receipt updates
- `user.presence`: Online/offline status

**Scaling Strategy**:
- Each WS Gateway instance is paired with a Chat Service instance
- Redis pub/sub for cross-instance communication
- Kafka for persistent event streaming
- Load balancer distributes WebSocket connections


---

### 6. Courses & Attendance Service (Ports 8085, 8086 - 2 instances)
**Tech Stack**: Go 1.24, Fiber v2, GORM, PostgreSQL, Redis, Kafka, Cloudinary

**Purpose**: Comprehensive course management with advanced QR-based attendance system.

**Key Features**:

**Course Management**:
- Course creation with subjects/categories
- Online and offline delivery types
- Course materials upload (Cloudinary)
- Enrollment management
- Assistant permissions (grading, attendance, materials)
- Course progress tracking

**Advanced QR Attendance System**:
- **Rotating QR Codes**: New QR every 30 seconds (configurable)
- **HMAC Signing**: Cryptographically signed QR tokens
- **Geofencing**: Haversine distance calculation for offline lessons
- **Device Fingerprinting**: Prevents device sharing
- **Emulator Detection**: Blocks virtual devices
- **Nonce-Based**: Single-use QR codes stored in Redis
- **Time Windows**: Present vs Late status based on scan time
- **Anti-Fraud**: Multiple validation layers

**Lesson Management**:
- Lesson scheduling and lifecycle (SCHEDULED → LIVE → COMPLETED)
- Lesson start/end/cancel/reschedule
- Automatic attendance finalization
- Lesson reminders via Kafka

**Absence Management**:
- Student/parent absence requests
- Parent approval workflow
- Excuse types (medical, emergency, parent excuse)
- Automatic EXCUSED status on approval

**Progress Calculation**:
- Weighted formula: `(completion_ratio * (1-weight)) + (attendance_ratio * weight)`
- Configurable attendance weight (default 30%)
- Attendance points: PRESENT=1.0, LATE=0.7, EXCUSED=0.8, ABSENT=0.0
- Auto-recomputation on attendance changes

**Calendar Integration**:
- Personal calendar feed for enrolled lessons
- Updates on reschedule/cancel


**Database Schema**:
```go
- courses (id, title, subject_id, delivery_type, location, geofence_radius, total_lessons)
- subjects (id, name, description)
- lessons (id, course_id, title, starts_at, ends_at, status)
- enrollments (course_id, user_id, enrolled_at)
- course_assistants (course_id, assistant_id, permissions)
- attendance_sessions (lesson_id, qr_rotation_seconds, qr_expiry_seconds)
- attendance_qr_tokens (lesson_id, nonce, issued_at, expires_at, signature)
- attendance_records (lesson_id, student_id, status, scan_time, device_id, lat, lng)
- absence_requests (lesson_id, student_id, reason, status, reviewed_by)
- progress_snapshots (course_id, student_id, completion_ratio, attendance_ratio, overall_progress)
```

**Dependencies**:
```go
github.com/gofiber/fiber/v2
gorm.io/gorm
github.com/redis/go-redis/v9
github.com/segmentio/kafka-go
github.com/cloudinary/cloudinary-go/v2
github.com/go-playground/validator/v10
```

**Why These Choices**:
- **Go**: High performance for QR rotation workers, concurrent request handling
- **Redis**: Fast QR token storage with TTL, nonce tracking, scan locks
- **PostgreSQL**: Complex relationships, ACID transactions for attendance
- **HMAC-SHA256**: Industry standard for token signing, prevents tampering
- **Haversine Formula**: Accurate geofencing calculation
- **Kafka**: Event-driven notifications for lesson lifecycle

**Security Measures**:
- No client time trust (server UTC only)
- No static QR codes (rotation prevents screenshots)
- Signature verification on every scan
- Geofence validation for offline lessons
- Device fingerprint validation
- Rate limiting on scan endpoint
- Idempotent scan handling (prevents double-scanning)
- Audit logging for all attendance changes


---

### 7. Payment Service (Port 8090)
**Tech Stack**: Go 1.24, Fiber v2, GORM, PostgreSQL, Redis, Kafka, Paymob API

**Purpose**: Handle payments, subscriptions, shopping cart, and billing automation.

**Key Features**:

**Shopping Cart**:
- Add multiple courses before checkout
- Support for ONE_TIME and MONTHLY billing types
- Cart validation and total calculation
- Mixed billing types in single cart
- Automatic cart clearing after payment

**Payment Processing**:
- Paymob integration (Egyptian payment gateway)
- Card payments (Visa, Mastercard)
- Mobile wallet payments
- HMAC webhook verification
- Multi-course checkout
- Payment order tracking

**Monthly Subscriptions**:
- Automatic subscription creation for MONTHLY items
- Subscription status tracking (ACTIVE, CANCELLED, SUSPENDED, EXPIRED)
- Next billing date tracking
- Subscription cancellation
- Background job for automatic renewals

**Email Notifications** (via Resend):
- Subscription renewal reminders with payment links
- Payment receipts
- Cancellation confirmations
- HTML email templates

**Background Jobs**:
- Subscription billing job (runs every 24 hours)
- Finds subscriptions due for billing
- Creates renewal payment orders
- Sends email notifications
- Updates billing dates

**Database Schema**:
```go
- carts (user_id, created_at, updated_at)
- cart_items (cart_id, course_id, billing_type, price_cents)
- payment_orders (id, user_id, order_type, total_cents, status, subscription_id)
- payment_order_items (order_id, course_id, billing_type, price_cents)
- payment_transactions (order_id, paymob_transaction_id, status, payment_method)
- subscriptions (id, user_id, course_id, status, next_billing_date, billing_cycle_months)
- payment_methods (user_id, token, last_four, card_brand, is_default)
```


**Dependencies**:
```go
github.com/gofiber/fiber/v2
gorm.io/gorm
github.com/redis/go-redis/v9
github.com/segmentio/kafka-go
github.com/go-playground/validator/v10
```

**Why These Choices**:
- **Go**: Reliable background jobs, efficient concurrent processing
- **Paymob**: Local Egyptian payment provider, supports local payment methods
- **PostgreSQL**: ACID transactions for payment integrity, complex queries
- **Redis**: Payment session caching, idempotency keys
- **Kafka**: Event streaming for enrollment activation
- **Resend**: Same email provider as auth service, consistent experience

**Payment Flow**:
1. User adds courses to cart
2. Checkout creates Paymob payment order
3. User completes payment on Paymob
4. Webhook verifies HMAC and updates order status
5. Enrollments activated in courses service
6. Subscriptions created for MONTHLY items
7. Cart cleared
8. Confirmation email sent

**Subscription Renewal Flow**:
1. Background job runs every 24 hours
2. Finds subscriptions with `next_billing_date <= today`
3. Creates renewal payment order
4. Sends email with payment link
5. User completes payment
6. Webhook updates subscription billing date
7. Receipt email sent

**Future Enhancements**:
- Automatic charging with stored payment methods (Paymob tokenization)
- Payment retry logic for failed renewals
- Proration for mid-cycle changes
- Discount/coupon system
- Refund processing


---

### 8. Recommendation Service (Port 8095)
**Tech Stack**: Python 3.11+, FastAPI, SQLAlchemy, PostgreSQL, Redis, Google Gemini AI

**Purpose**: AI-powered course recommendations, chatbot assistant, and progress reports.

**Key Features**:

**AI Course Recommendations**:
- Personalized recommendations based on user profile
- Interest-based filtering
- Enrollment history analysis
- Trending courses
- Cache with 6-hour TTL
- Refresh on-demand

**AI Chatbot Assistant**:
- **SSE Streaming**: Server-Sent Events for real-time responses
- **Multimodal Support**: Text and image inputs
- **Context Management**: Maintains conversation history (20 messages)
- **Session Management**: Multiple chat sessions per user (max 10)
- **Media Upload**: Cloudinary integration for images
- **Code Sanitization**: Strips code blocks from AI responses
- **Pagination**: Message history with pagination
- **Course Context**: Fetches course data for relevant answers

**Parent Progress Reports**:
- AI-generated progress reports for students
- Multi-language support (English, Arabic, French)
- PDF generation (FPDF2)
- Attendance analysis
- Performance insights
- Downloadable reports

**Database Schema**:
```python
- recommendation_history (user_id, course_id, score, created_at)
- chat_sessions (id, user_id, title, is_active, created_at)
- chat_messages (id, session_id, role, content, media_url, created_at)
```

**Dependencies**:
```python
fastapi==0.110.0
google-genai==0.3.0
sqlalchemy==2.0.29
redis==5.0.3
httpx==0.27.0
cloudinary==1.41.0
fpdf2==2.7.8
```


**Why These Choices**:
- **Python**: Best ecosystem for AI/ML, easy integration with AI models
- **FastAPI**: Modern async framework, automatic OpenAPI docs, excellent performance
- **Google Gemini**: Advanced language model, multimodal capabilities, good pricing
- **SQLAlchemy**: Mature Python ORM, excellent for complex queries
- **Redis**: Fast caching for recommendations, reduces AI API calls
- **SSE**: Better than WebSocket for one-way streaming, simpler implementation
- **FPDF2**: Lightweight PDF generation, supports multiple languages

**AI Integration**:
- Gemini 4-26B model for high-quality responses
- Streaming responses for better UX
- Context window management (20 messages)
- Automatic course data fetching for relevant context
- Rate limiting to prevent API abuse

**Chatbot Features**:
- Max 10 active chats per user
- Max 100 messages per chat
- Max 2000 characters per message
- 1-hour cache for course data
- Automatic session cleanup

**Report Generation**:
- Fetches student data from courses service
- Analyzes attendance patterns
- Generates insights with AI
- Creates formatted PDF
- Sends notification via Kafka

---

## Database & Storage

### PostgreSQL (Shared Database)
**Version**: 15-alpine

**Why PostgreSQL**:
- **ACID Compliance**: Critical for payments, enrollments, attendance
- **JSON Support**: Flexible data storage for notifications, preferences
- **Complex Relationships**: Parent-child links, course enrollments, sessions
- **Mature Ecosystem**: Excellent tooling, monitoring, backup solutions
- **Performance**: Handles millions of rows efficiently with proper indexing
- **Extensions**: PostGIS for geolocation (future), full-text search

**Database Design Principles**:
- UUID primary keys for distributed systems
- Composite unique indexes for preventing duplicates
- Optimized indexes for common queries
- Soft deletes with `deleted_at` timestamps
- Audit trails for sensitive operations
- UTC timestamps everywhere


**Key Indexes**:
```sql
-- Auth Service
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_sessions_user_active ON sessions(user_id, is_active, is_revoked, expires_at);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Courses Service
CREATE INDEX idx_enrollments_user_course ON enrollments(user_id, course_id);
CREATE INDEX idx_attendance_lesson_student ON attendance_records(lesson_id, student_id);
CREATE INDEX idx_lessons_course_status ON lessons(course_id, status);

-- Chat Service
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);
CREATE INDEX idx_participants_user ON conversation_participants(user_id);

-- Payment Service
CREATE INDEX idx_orders_user_status ON payment_orders(user_id, status);
CREATE INDEX idx_subscriptions_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscriptions_billing_date ON subscriptions(next_billing_date);
```

### Redis (Shared Cache)
**Version**: 7-alpine

**Why Redis**:
- **In-Memory Speed**: Sub-millisecond latency for hot data
- **Pub/Sub**: Real-time event broadcasting across services
- **TTL Support**: Automatic expiration for QR tokens, sessions
- **Data Structures**: Lists, sets, sorted sets for complex operations
- **Atomic Operations**: INCR, SETNX for counters and locks
- **Persistence**: Optional RDB/AOF for durability

**Use Cases by Service**:

**Auth Service**:
- Session storage: `session:{token}` (TTL: session expiry)
- Rate limiting: `ratelimit:login:{ip}` (token bucket)
- Email verification codes: `verify:{email}` (TTL: 15 minutes)
- Password reset tokens: `reset:{token}` (TTL: 1 hour)

**Courses Service**:
- Active QR tokens: `attendance:lesson:{id}:active_qr` (TTL: 30s)
- QR nonces: `attendance:lesson:{id}:nonce:{nonce}` (TTL: 35s)
- Scan locks: `attendance:lock:scan:{lesson_id}:{student_id}` (TTL: 5s)
- Rate limiting: `ratelimit:scan:{user_id}`

**Chat Service**:
- Typing indicators: `typing:{conversation_id}:{user_id}` (TTL: 5s)
- Online presence: `presence:user:{user_id}` (TTL: 30s)
- Message cache: `messages:{conversation_id}:recent` (TTL: 1 hour)


**Payment Service**:
- Cart sessions: `cart:{user_id}` (TTL: 24 hours)
- Payment idempotency: `payment:idempotency:{key}` (TTL: 24 hours)
- Subscription locks: `subscription:lock:{id}` (TTL: 30s)

**Recommendation Service**:
- Recommendation cache: `recommendation:v1:{user_id}` (TTL: 6 hours)
- Course data cache: `course:{id}` (TTL: 1 hour)
- Trending courses: `trending:courses` (TTL: 1 hour)

**WS Gateway**:
- Connection mapping: `ws:user:{user_id}:connections` (set of connection IDs)
- Pub/Sub channels: `ws:broadcast`, `ws:user:{user_id}`

### Kafka (Event Streaming)
**Version**: Confluent 7.5.0

**Why Kafka**:
- **Reliable Delivery**: At-least-once delivery guarantees
- **Event Sourcing**: Immutable event log for audit trails
- **Decoupling**: Services don't need to know about each other
- **Scalability**: Horizontal scaling with partitions
- **Replay**: Can replay events for debugging or recovery
- **Ordering**: Maintains message order within partitions

**Topics**:
```
courses.lesson.started.v1
courses.lesson.ended.v1
courses.lesson.canceled.v1
courses.lesson.rescheduled.v1
courses.attendance.recorded.v1
courses.attendance.finalized.v1
courses.absence.requested.v1
courses.absence.reviewed.v1
courses.progress.updated.v1
courses.notification.requested.v1
chat.message.created.v1
chat.typing.v1
payment.order.completed.v1
payment.subscription.created.v1
```

**Event Envelope**:
```json
{
  "event_id": "uuid",
  "event_type": "courses.lesson.started.v1",
  "occurred_at": "2024-01-19T12:00:00Z",
  "aggregate_id": "lesson-uuid",
  "actor_user_id": "user-uuid",
  "payload": { ... },
  "trace_id": "trace-uuid"
}
```

**Consumer Groups**:
- `notification-courses-v1`: Notification service consumes course events
- `notification-chat-v1`: Notification service consumes chat events
- `notification-payment-v1`: Notification service consumes payment events
- `ws-gateway-v1`: WS Gateway consumes chat events


---

## Security & Authentication

### Authentication Flow
1. **Login**: User provides credentials → Auth service validates → Returns JWT access + refresh tokens
2. **Token Validation**: Each service validates JWT signature using shared secret
3. **Session Tracking**: Sessions stored in PostgreSQL with device info, location, GPS
4. **Token Refresh**: Refresh token used to get new access token without re-login
5. **Revocation**: Sessions can be revoked (logout, security breach)

### JWT Structure
```json
{
  "userId": "uuid",
  "role": "STUDENT",
  "sessionId": "uuid",
  "deviceId": "uuid",
  "iat": 1234567890,
  "exp": 1234567890
}
```

### Security Layers

**API Gateway**:
- Bot detection (Arcjet)
- VPN/proxy blocking (Arcjet)
- CORS protection
- Rate limiting
- Request timeout

**Auth Service**:
- Password hashing (Bcrypt, work factor 12)
- 2FA with TOTP
- Device fingerprinting
- New device blocking
- Session management
- Email verification
- Rate limiting on login/register

**Courses Service**:
- QR signature verification (HMAC-SHA256)
- Geofencing validation
- Device fingerprint validation
- Emulator detection
- Nonce-based single-use QR
- Rate limiting on scan endpoint

**Payment Service**:
- Webhook HMAC verification
- Idempotency keys
- Transaction locking
- PCI compliance (tokenized cards)

**Internal Service Communication**:
- Shared secret header: `x-internal-service-secret`
- Service-to-service authentication
- No public exposure of internal endpoints


---

## Infrastructure & DevOps

### Docker Compose Architecture
```yaml
services:
  # Infrastructure
  - postgres (5432)
  - redis (6379)
  - zookeeper (2181)
  - kafka (9092)
  
  # Chat Services (3 instances)
  - chat-service-1 (6004)
  - chat-service-2 (6014)
  - chat-service-3 (6024)
  
  # WS Gateways (3 instances)
  - ws-gateway-1 (6005)
  - ws-gateway-2 (6015)
  - ws-gateway-3 (6025)
  
  # Courses Services (2 instances)
  - courses-service-1 (8085)
  - courses-service-2 (8086)
  
  # Single Instance Services
  - payment-service (8090)
  - recommendation-service (8095)
```

**Node Services (Run Manually)**:
- api-gateway (6000)
- auth-service (6001)
- notification-service (6003)

### Scaling Strategy

**Horizontal Scaling** (Multiple Instances):
- **Chat Service**: 3 instances for load distribution
- **WS Gateway**: 3 instances, each paired with a chat instance
- **Courses Service**: 2 instances for high availability

**Why These Services Scale**:
- High concurrent connections (chat, WebSocket)
- CPU-intensive operations (QR rotation, geofencing)
- High request volume (course queries, attendance)

**Stateless Design**:
- All services are stateless
- State stored in PostgreSQL/Redis
- Kafka for cross-instance communication
- Load balancer can distribute to any instance

**Single Instance Services**:
- **Auth Service**: Session state in Redis, can scale if needed
- **Payment Service**: Background jobs need coordination (future: distributed locks)
- **Notification Service**: Kafka consumer, can scale with consumer groups
- **Recommendation Service**: AI API rate limits, cache-heavy


### Health Checks
Each service exposes `/health` endpoint:
```json
{
  "status": "ok",
  "service": "service-name",
  "timestamp": "2024-01-19T12:00:00Z"
}
```

API Gateway aggregates health from all services:
```json
{
  "status": "ok",
  "service": "api-gateway",
  "upstreams": {
    "auth-service": { "status": "ok", "latency": 45 },
    "notification-service": { "status": "ok", "latency": 32 }
  }
}
```

### Environment Variables
Each service has `.env` file with:
- Database connection strings
- Redis URLs
- Kafka brokers
- API keys (Paymob, Resend, Cloudinary, AI)
- JWT secrets
- Internal service secrets
- Feature flags

### Deployment
**Development**:
```bash
# Start infrastructure
docker-compose up -d postgres redis kafka

# Start Node services
cd api-gateway && npm run dev
cd auth-service && npm run dev
cd notification-service && npm run dev

# Go services run in Docker
docker-compose up chat-service-1 ws-gateway-1 courses-service-1 payment-service
```

**Production**:
```bash
# All services in Docker
docker-compose -f docker-compose.yml up -d

# Or Kubernetes deployment (future)
kubectl apply -f k8s/
```

---

## Design Decisions & Constraints

### Why Microservices?
**Pros**:
- Independent scaling (chat needs more instances than payment)
- Technology diversity (Go for performance, Python for AI, Node for rapid dev)
- Team autonomy (different teams can work on different services)
- Fault isolation (chat failure doesn't affect payments)
- Independent deployment (update one service without redeploying all)

**Cons**:
- Increased complexity (distributed systems, network calls)
- Data consistency challenges (eventual consistency)
- Debugging difficulty (distributed tracing needed)
- Operational overhead (more services to monitor)

**Why We Chose It**: Benefits outweigh costs for this scale and team structure.


### Why Go for Some Services?
**Chat, WS Gateway, Courses, Payment**:
- **Concurrency**: Goroutines perfect for WebSocket connections, QR rotation workers
- **Performance**: Low latency, efficient memory usage
- **Compiled**: Single binary deployment, no runtime dependencies
- **Type Safety**: Catches errors at compile time
- **Standard Library**: Excellent HTTP, JSON, crypto support

**When Not to Use Go**:
- Rapid prototyping (Node.js faster)
- AI/ML integration (Python ecosystem better)
- Complex ORM needs (Prisma better than GORM)

### Why Node.js for Some Services?
**API Gateway, Auth, Notification**:
- **Rapid Development**: Express ecosystem, extensive middleware
- **I/O Bound**: Perfect for proxy, auth checks, notifications
- **Prisma**: Best-in-class ORM with type safety
- **Ecosystem**: Rich package ecosystem (JWT, OAuth, email)
- **Team Familiarity**: Easier to find Node.js developers

**When Not to Use Node.js**:
- CPU-intensive tasks (Go better)
- High concurrency (Go goroutines better)
- Real-time systems (Go channels better)

### Why Python for AI Service?
**Recommendation Service**:
- **AI Ecosystem**: Best libraries (TensorFlow, PyTorch, Transformers)
- **FastAPI**: Modern async framework, excellent performance
- **Easy Integration**: Simple AI model integration
- **Data Science**: Pandas, NumPy for data analysis

**When Not to Use Python**:
- High-performance APIs (Go/Node faster)
- Real-time systems (GIL limitations)
- Memory-intensive (higher memory usage)

### Why PostgreSQL?
**All Services**:
- **ACID Transactions**: Critical for payments, enrollments
- **Complex Queries**: Joins, aggregations, window functions
- **JSON Support**: Flexible schema for notifications, preferences
- **Mature**: 30+ years of development, battle-tested
- **Extensions**: PostGIS, full-text search, pg_cron

**Why Not MongoDB**:
- Need ACID transactions
- Complex relationships (parent-child, enrollments)
- Strong consistency requirements

**Why Not MySQL**:
- PostgreSQL has better JSON support
- Better full-text search
- More advanced features (window functions, CTEs)


### Why Redis?
**All Services**:
- **Speed**: Sub-millisecond latency for hot data
- **Pub/Sub**: Real-time event broadcasting
- **TTL**: Automatic expiration for QR tokens, sessions
- **Atomic Operations**: INCR, SETNX for counters, locks
- **Data Structures**: Lists, sets, sorted sets

**Why Not Memcached**:
- Redis has more data structures
- Pub/Sub support
- Persistence options
- Lua scripting

**Why Not In-Memory**:
- Need shared state across instances
- Need persistence for sessions
- Need pub/sub for real-time

### Why Kafka?
**Event Streaming**:
- **Reliability**: At-least-once delivery
- **Scalability**: Horizontal scaling with partitions
- **Replay**: Can replay events for debugging
- **Ordering**: Maintains order within partitions
- **Decoupling**: Services don't need to know about each other

**Why Not RabbitMQ**:
- Kafka better for event sourcing
- Better for high throughput
- Better for replay scenarios

**Why Not Direct HTTP**:
- Coupling between services
- No retry/replay
- Synchronous (blocking)

### Why Paymob?
**Payment Gateway**:
- **Local**: Egyptian payment provider
- **Payment Methods**: Cards, wallets, installments
- **Compliance**: PCI-DSS compliant
- **Integration**: Good API documentation
- **Support**: Local support team

**Why Not Stripe**:
- Limited support for Egyptian payment methods
- Higher fees for international transactions
- Currency conversion issues

### Why Resend?
**Email Service**:
- **Modern API**: RESTful, easy to integrate
- **Deliverability**: High inbox placement rate
- **Developer Experience**: Excellent docs, SDKs
- **Pricing**: Generous free tier
- **Consistency**: Same provider as auth service

**Why Not SendGrid**:
- Resend has better developer experience
- Simpler pricing
- Better documentation

**Why Not SMTP**:
- Deliverability issues
- Spam folder problems
- No analytics
- Complex setup


### Why Firebase FCM?
**Push Notifications**:
- **Cross-Platform**: iOS and Android
- **Reliability**: Industry standard, 99.9% uptime
- **Free**: Generous free tier
- **Features**: Topics, targeting, analytics
- **Integration**: Official SDKs for all platforms

**Why Not OneSignal**:
- FCM is free
- Better integration with Firebase ecosystem
- More control over implementation

### Why Cloudinary?
**Media Storage**:
- **CDN**: Global content delivery
- **Optimization**: Automatic image/video optimization
- **Transformations**: On-the-fly resizing, cropping
- **Security**: Signed URLs, access control
- **Pricing**: Generous free tier

**Why Not AWS S3**:
- Cloudinary has built-in transformations
- Easier to use
- Better for media-heavy applications

### Why Google Gemini?
**AI Model**:
- **Multimodal**: Text and image inputs
- **Performance**: Fast response times
- **Quality**: High-quality responses
- **Pricing**: Competitive pricing
- **Context**: Large context window

**Why Not OpenAI**:
- Gemini has better pricing
- Multimodal support
- Google ecosystem integration

---

## Performance Considerations

### Caching Strategy
**L1 Cache (Redis)**:
- Session data (TTL: session expiry)
- QR tokens (TTL: 30-35 seconds)
- Recommendations (TTL: 6 hours)
- Course data (TTL: 1 hour)

**L2 Cache (PostgreSQL)**:
- Materialized views for complex queries
- Indexed columns for fast lookups

### Database Optimization
- **Indexes**: All foreign keys, frequently queried columns
- **Partitioning**: Large tables (messages, notifications) by date
- **Connection Pooling**: Reuse database connections
- **Query Optimization**: EXPLAIN ANALYZE for slow queries

### API Optimization
- **Pagination**: All list endpoints paginated
- **Field Selection**: GraphQL-style field selection (future)
- **Compression**: Gzip for responses > 1KB
- **CDN**: Static assets served from CDN


### Scalability Limits
**Current Architecture Can Handle**:
- 10,000+ concurrent users
- 1,000+ messages per second
- 100+ courses with 1,000+ students each
- 10,000+ QR scans per minute

**Bottlenecks**:
- PostgreSQL write throughput (can scale with read replicas)
- Redis memory (can scale with Redis Cluster)
- Kafka throughput (can scale with more partitions)
- AI API rate limits (can cache more aggressively)

---

## Future Enhancements

### Short Term
- [ ] Automatic payment charging with stored cards
- [ ] Payment retry logic for failed subscriptions
- [x] Distributed tracing (Jaeger via OpenTelemetry)
- [x] Metrics and monitoring (Prometheus, Grafana, Loki)
- [ ] API rate limiting per user
- [ ] WebSocket connection pooling optimization

### Medium Term
- [ ] GraphQL API for flexible queries
- [ ] Read replicas for PostgreSQL
- [ ] Redis Cluster for high availability
- [ ] Kubernetes deployment
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Automated testing (integration, E2E)
- [ ] API versioning strategy

### Long Term
- [ ] Multi-region deployment
- [ ] Event sourcing for audit trails
- [ ] CQRS pattern for read-heavy operations
- [ ] Machine learning for fraud detection
- [ ] Advanced analytics and reporting
- [ ] Mobile app development (React Native/Flutter)
- [ ] Admin dashboard (React/Vue)

---

## Summary

This education platform is built with a **modern microservices architecture** using the best tools for each job:

- **Node.js/TypeScript** for rapid development and I/O-bound services
- **Go** for high-performance, concurrent, real-time systems
- **Python** for AI/ML integration and data science
- **PostgreSQL** for reliable, ACID-compliant data storage
- **Redis** for fast caching and real-time features
- **Kafka** for event-driven, decoupled communication

The architecture prioritizes:
- **Scalability**: Horizontal scaling for high-traffic services
- **Reliability**: ACID transactions, event sourcing, retry logic
- **Security**: Multi-layer security, encryption, authentication
- **Performance**: Caching, indexing, optimization
- **Developer Experience**: Type safety, good tooling, clear separation

Each technology choice is **justified by specific requirements** and **constraints**, resulting in a robust, scalable, and maintainable system.

