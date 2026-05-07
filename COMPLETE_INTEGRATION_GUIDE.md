# Integration Guide - All Services Working Together

## SYSTEM OVERVIEW DIAGRAM

```mermaid
graph TB
    subgraph "Client Layer"
        Web["Web Browser"]
        iOS["iOS App"]
        Android["Android App"]
    end
    
    subgraph "API Gateway Layer"
        Gateway["API Gateway<br/>Express.js<br/>Port 6000"]
    end
    
    subgraph "Authentication & Authorization"
        Auth["Auth Service<br/>Node.js + Prisma<br/>Port 6001"]
    end
    
    subgraph "Real-time Communication"
        Chat["Chat Service<br/>Go + Fiber<br/>Ports 6004/14/24"]
        WS["WS Gateway<br/>Go + Fiber<br/>Ports 6005/15/25"]
    end
    
    subgraph "Core Services"
        Courses["Courses Service<br/>Go + Fiber<br/>Ports 8085/86"]
        Payment["Payment Service<br/>Go + Fiber<br/>Port 8090"]
        Notif["Notification Service<br/>Node.js + Firebase<br/>Port 6003"]
        Recommend["Recommendation Svc<br/>Python + FastAPI<br/>Port 8095"]
    end
    
    subgraph "Data Layer"
        DB["PostgreSQL<br/>Port 5432<br/>34 Tables"]
        RedisDB["Redis<br/>Port 6379<br/>Caching & Sessions"]
        Kafka["Kafka<br/>Port 9092<br/>Event Streaming"]
    end
    
    subgraph "External Services"
        FCM["Firebase FCM<br/>Push Notifications"]
        Paymob["Paymob API<br/>Payment Gateway"]
        Resend["Resend<br/>Email Service"]
        Cloudinary["Cloudinary<br/>Media CDN"]
        Gemini["Google Gemini<br/>AI Model"]
        Arcjet["Arcjet<br/>Security"]
    end
    
    Web -->|HTTPS| Gateway
    iOS -->|HTTPS| Gateway
    Android -->|HTTPS| Gateway
    iOS -->|WebSocket| WS
    Android -->|WebSocket| WS
    
    Gateway -->|Route| Auth
    Gateway -->|Route| Notif
    Gateway -->|Route| Chat
    Gateway -->|Route| Courses
    Gateway -->|Route| Payment
    Gateway -->|Route| Recommend
    
    Auth -->|Query/Store| DB
    Chat -->|Query/Store| DB
    Courses -->|Query/Store| DB
    Payment -->|Query/Store| DB
    Notif -->|Query/Store| DB
    Recommend -->|Query/Store| DB
    
    Auth -->|Cache| RedisDB
    Chat -->|Cache| RedisDB
    Courses -->|Cache| RedisDB
    Payment -->|Cache| RedisDB
    Recommend -->|Cache| RedisDB
    WS -->|Pub/Sub| RedisDB
    
    Auth -->|Emit| Kafka
    Chat -->|Emit| Kafka
    Courses -->|Emit| Kafka
    Payment -->|Emit| Kafka
    
    Notif -->|Consume| Kafka
    WS -->|Consume| Kafka
    
    Notif -->|Send| FCM
    Notif -->|Send| Resend
    Payment -->|Send| Resend
    
    Payment -->|Process| Paymob
    
    Chat -->|Upload| Cloudinary
    Courses -->|Upload| Cloudinary
    Auth -->|Upload| Cloudinary
    
    Recommend -->|Call| Gemini
    
    Gateway -->|Security| Arcjet
```

---

## COMPLETE USER JOURNEY - FROM SIGNUP TO COURSE COMPLETION

```mermaid
sequenceDiagram
    participant User as User
    participant Gateway as Gateway
    participant Auth as Auth
    participant DB as DB
    participant Redis as Redis
    participant Kafka as Kafka
    participant Notif as Notif
    participant FCM as FCM
    
    User->>Gateway: POST /auth/register
    Gateway->>Auth: Forward request
    Auth->>Auth: Validate input
    Auth->>Auth: Hash password (Bcrypt)
    Auth->>DB: Create user
    Auth->>RedisDB: Store session
    Auth->>Auth: Generate JWT
    Auth->>Notif: Send verification email
    Auth->>Kafka: Emit user.registered
    Auth->>Gateway: Return tokens
    Gateway->>User: JWT + Refresh Token
    
    User->>Gateway: GET /courses
    Gateway->>Courses: Forward request
    Courses->>DB: Query courses
    Courses->>Gateway: Return course list
    Gateway->>User: Course list
    
    User->>Gateway: POST /cart/add
    Gateway->>Payment: Forward request
    Payment->>DB: Add to cart
    Payment->>RedisDB: Cache cart
    Payment->>Gateway: Cart updated
    Gateway->>User: Success
    
    User->>Gateway: POST /cart/checkout
    Gateway->>Payment: Forward request
    Payment->>Courses: Auto-enroll user
    Courses->>DB: Create enrollment
    Payment->>Paymob: Create payment order
    Payment->>Gateway: Payment URL
    Gateway->>User: Redirect to Paymob
    
    User->>Paymob: Complete payment
    Paymob->>Payment: Webhook callback
    Payment->>DB: Update order status
    Payment->>Courses: Activate enrollment
    Payment->>Kafka: Emit payment.completed
    Payment->>Notif: Send receipt email
    Kafka->>Notif: Consume event
    Notif->>FCM: Send push notification
    FCM->>User: "Payment successful!"
    
    User->>Gateway: GET /courses/my
    Gateway->>Courses: Forward request
    Courses->>DB: Query enrollments
    Courses->>Gateway: Return enrolled courses
    Gateway->>User: My courses
    
    User->>Gateway: GET /courses/:id/lessons
    Gateway->>Courses: Forward request
    Courses->>DB: Query lessons
    Courses->>Gateway: Return lessons
    Gateway->>User: Lesson list
    
    User->>Gateway: Watch lesson video
    Gateway->>Courses: Get video URL
    Courses->>Cloudinary: Get signed URL
    Cloudinary->>User: Video stream
    
    Teacher->>Gateway: POST /lessons/:id/start
    Gateway->>Courses: Forward request
    Courses->>DB: Update lesson status
    Courses->>RedisDB: Start QR rotation
    Courses->>Kafka: Emit lesson.started
    Kafka->>Notif: Consume event
    Notif->>FCM: Send "Lesson started"
    FCM->>User: Notification
    
    User->>Gateway: POST /attendance/scan
    Gateway->>Courses: Forward request
    Courses->>Auth: Verify device
    Courses->>RedisDB: Get QR token
    Courses->>Courses: Verify signature
    Courses->>Courses: Check geofence
    Courses->>DB: Record attendance
    Courses->>Kafka: Emit attendance.recorded
    Kafka->>Notif: Consume event
    Notif->>FCM: Send "Attendance recorded"
    FCM->>User: Attendance confirmed
```

---

## KAFKA EVENT FLOW - ALL TOPICS

```mermaid
graph TB
    subgraph "Event Producers"
        Auth["Auth Service<br/>user.registered<br/>user.login"]
        Courses["Courses Service<br/>lesson.started<br/>lesson.ended<br/>lesson.canceled<br/>lesson.rescheduled<br/>attendance.recorded<br/>attendance.finalized<br/>absence.requested<br/>absence.reviewed<br/>progress.updated"]
        Chat["Chat Service<br/>message.created<br/>typing.started<br/>typing.stopped"]
        Payment["Payment Service<br/>payment.completed<br/>payment.failed<br/>subscription.created<br/>subscription.renewed<br/>subscription.canceled"]
    end
    
    subgraph "Kafka Topics"
        T1["user.registered.v1"]
        T2["lesson.started.v1"]
        T3["attendance.recorded.v1"]
        T4["message.created.v1"]
        T5["payment.completed.v1"]
        T6["subscription.created.v1"]
    end
    
    subgraph "Event Consumers"
        Notif["Notification Service<br/>- Send push notifications<br/>- Send emails<br/>- Store notification history"]
        WS["WS Gateway<br/>- Broadcast to connected clients<br/>- Update presence<br/>- Real-time updates"]
        Courses2["Courses Service<br/>- Update progress<br/>- Finalize attendance<br/>- Send reminders"]
    end
    
    Auth -->|Produce| T1
    Courses -->|Produce| T2
    Courses -->|Produce| T3
    Chat -->|Produce| T4
    Payment -->|Produce| T5
    Payment -->|Produce| T6
    
    T1 -->|Consume| Notif
    T2 -->|Consume| Notif
    T2 -->|Consume| WS
    T3 -->|Consume| Notif
    T3 -->|Consume| WS
    T4 -->|Consume| Notif
    T4 -->|Consume| WS
    T5 -->|Consume| Notif
    T5 -->|Consume| Courses2
    T6 -->|Consume| Notif
    T6 -->|Consume| Courses2
```

---

## REDIS USAGE ACROSS SERVICES

```mermaid
graph TB
    RedisDB["Redis<br/>Port 6379"]
    
    Auth["Auth Service"]
    Chat["Chat Service"]
    Courses["Courses Service"]
    Payment["Payment Service"]
    Recommend["Recommendation Service"]
    WS["WS Gateway"]
    
    Auth -->|session:{token}| RedisDB
    Auth -->|ratelimit:login:{ip}| RedisDB
    Auth -->|verify:{email}| RedisDB
    Auth -->|reset:{token}| RedisDB
    
    Chat -->|typing:{conv_id}:{user_id}| RedisDB
    Chat -->|presence:user:{user_id}| RedisDB
    Chat -->|messages:{conv_id}:recent| RedisDB
    
    Courses -->|attendance:lesson:{id}:active_qr| RedisDB
    Courses -->|attendance:lesson:{id}:nonce:{nonce}| RedisDB
    Courses -->|attendance:lock:scan:{lesson_id}:{student_id}| RedisDB
    Courses -->|ratelimit:scan:{user_id}| RedisDB
    
    Payment -->|cart:{user_id}| RedisDB
    Payment -->|payment:idempotency:{key}| RedisDB
    Payment -->|subscription:lock:{id}| RedisDB
    
    Recommend -->|recommendation:v1:{user_id}| RedisDB
    Recommend -->|course:{id}| RedisDB
    Recommend -->|trending:courses| RedisDB
    
    WS -->|ws:user:{user_id}:connections| RedisDB
    WS -->|ws:broadcast| RedisDB
    WS -->|ws:user:{user_id}| RedisDB
```

---

## DATABASE RELATIONSHIPS - COMPLETE VIEW

```mermaid
erDiagram
    USERS ||--o{ SESSIONS : has
    USERS ||--o{ COURSE_ENROLLMENTS : enrolls
    USERS ||--o{ MESSAGES : sends
    USERS ||--o{ NOTIFICATIONS : receives
    USERS ||--o{ CARTS : owns
    USERS ||--o{ PAYMENT_ORDERS : places
    USERS ||--o{ SUBSCRIPTIONS : subscribes
    USERS ||--o{ CHAT_SESSIONS : creates
    
    COURSES ||--o{ LESSONS : contains
    COURSES ||--o{ COURSE_ENROLLMENTS : has
    COURSES ||--o{ CART_ITEMS : in
    COURSES ||--o{ PAYMENT_ORDER_ITEMS : in
    COURSES ||--o{ SUBSCRIPTIONS : billed
    
    LESSONS ||--o{ ATTENDANCE_SESSIONS : has
    LESSONS ||--o{ ATTENDANCE_RECORDS : tracks
    LESSONS ||--o{ ABSENCE_REQUESTS : has
    
    ATTENDANCE_SESSIONS ||--o{ ATTENDANCE_QR_TOKENS : generates
    ATTENDANCE_SESSIONS ||--o{ ATTENDANCE_RECORDS : records
    
    CARTS ||--o{ CART_ITEMS : contains
    
    PAYMENT_ORDERS ||--o{ PAYMENT_ORDER_ITEMS : contains
    PAYMENT_ORDERS ||--o{ PAYMENT_TRANSACTIONS : has
    PAYMENT_ORDERS ||--o{ SUBSCRIPTIONS : creates
    
    CONVERSATIONS ||--o{ MESSAGES : contains
    CONVERSATIONS ||--o{ CONVERSATION_PARTICIPANTS : has
    
    CHAT_SESSIONS ||--o{ CHAT_MESSAGES : contains
```

---

## SERVICE DEPENDENCIES & COMMUNICATION

```mermaid
graph TB
    Gateway["API Gateway"]
    
    Auth["Auth Service"]
    Notif["Notification Service"]
    Chat["Chat Service"]
    WS["WS Gateway"]
    Courses["Courses Service"]
    Payment["Payment Service"]
    Recommend["Recommendation Service"]
    
    Gateway -->|Routes to| Auth
    Gateway -->|Routes to| Notif
    Gateway -->|Routes to| Chat
    Gateway -->|Routes to| Courses
    Gateway -->|Routes to| Payment
    Gateway -->|Routes to| Recommend
    
    Auth -->|Validates JWT| Chat
    Auth -->|Validates JWT| Courses
    Auth -->|Validates JWT| Payment
    Auth -->|Validates JWT| Recommend
    Auth -->|Validates JWT| WS
    
    Courses -->|Queries courses| Payment
    Courses -->|Activates enrollment| Payment
    
    Payment -->|Queries courses| Courses
    Payment -->|Queries user data| Auth
    
    Recommend -->|Queries user data| Auth
    Recommend -->|Queries courses| Courses
    
    Notif -->|Consumes events| Kafka["Kafka"]
    WS -->|Consumes events| Kafka
    
    Auth -->|Emits events| Kafka
    Chat -->|Emits events| Kafka
    Courses -->|Emits events| Kafka
    Payment -->|Emits events| Kafka
    
    Notif -->|Sends push| FCM["Firebase FCM"]
    Notif -->|Sends email| Resend["Resend"]
    
    Payment -->|Processes payment| Paymob["Paymob API"]
    
    Chat -->|Uploads media| Cloudinary["Cloudinary"]
    Courses -->|Uploads media| Cloudinary
    Auth -->|Uploads images| Cloudinary
    
    Recommend -->|Calls AI| Gemini["Google Gemini"]
```

---

## COMPLETE DATA FLOW - PAYMENT TO COURSE ACCESS

```
1. STUDENT ADDS COURSE TO CART
   Student → Gateway → Payment Service
   ↓
   Payment Service:
   - Validates course exists (calls Courses Service)
   - Creates cart item in PostgreSQL
   - Caches cart in Redis
   ↓
   Response: Cart updated

2. STUDENT CHECKS OUT
   Student → Gateway → Payment Service
   ↓
   Payment Service:
   - Validates all courses in cart
   - Calls Courses Service to auto-enroll (unpaid)
   - Creates payment order in PostgreSQL
   - Generates Paymob payment URL
   ↓
   Response: Payment URL

3. STUDENT COMPLETES PAYMENT
   Student → Paymob → Completes payment
   ↓
   Paymob → Webhook → Payment Service
   ↓
   Payment Service:
   - Verifies HMAC signature
   - Updates order status to PAID
   - Calls Courses Service to activate enrollment
   - Creates subscriptions for MONTHLY items
   - Clears cart from Redis
   - Emits Kafka event: payment.completed
   - Sends receipt email via Resend
   ↓
   Kafka Consumer (Notification Service):
   - Receives payment.completed event
   - Creates notification in PostgreSQL
   - Sends push notification via Firebase FCM
   - Sends email confirmation

4. STUDENT ACCESSES COURSE
   Student → Gateway → Courses Service
   ↓
   Courses Service:
   - Validates JWT token (calls Auth Service)
   - Checks enrollment in PostgreSQL
   - Checks is_paid = true
   - Returns course content
   ↓
   Response: Course lessons and materials

5. STUDENT WATCHES LESSON
   Student → Gateway → Courses Service
   ↓
   Courses Service:
   - Validates access
   - Returns video URL from Cloudinary
   ↓
   Student → Cloudinary → Streams video

6. TEACHER STARTS LESSON
   Teacher → Gateway → Courses Service
   ↓
   Courses Service:
   - Updates lesson status to LIVE
   - Starts QR rotation worker
   - Generates QR token every 30 seconds
   - Signs with HMAC-SHA256
   - Stores in Redis with 35s TTL
   - Emits Kafka event: lesson.started
   ↓
   Kafka Consumer (Notification Service):
   - Sends push notification to enrolled students

7. STUDENT SCANS QR CODE
   Student → Gateway → Courses Service
   ↓
   Courses Service:
   - Verifies JWT signature
   - Calls Auth Service to verify device
   - Verifies QR signature
   - Checks geofence (Haversine)
   - Validates device fingerprint
   - Checks for emulator
   - Acquires Redis lock
   - Records attendance in PostgreSQL
   - Emits Kafka event: attendance.recorded
   ↓
   Kafka Consumer (Notification Service):
   - Sends push notification: "Attendance recorded"
   ↓
   Kafka Consumer (WS Gateway):
   - Broadcasts to teacher's connected clients
   ↓
   Response: Attendance confirmed

8. TEACHER ENDS LESSON
   Teacher → Gateway → Courses Service
   ↓
   Courses Service:
   - Updates lesson status to COMPLETED
   - Auto-marks absent students
   - Calculates progress
   - Emits Kafka event: lesson.ended
   ↓
   Kafka Consumer (Notification Service):
   - Sends notifications to students
```

---

## SCALABILITY & LOAD DISTRIBUTION

```mermaid
graph TB
    LB["Load Balancer"]
    
    LB -->|Route| Gateway1["API Gateway Instance 1"]
    LB -->|Route| Gateway2["API Gateway Instance 2"]
    
    Gateway1 -->|Route| Auth["Auth Service Single Instance"]
    Gateway2 -->|Route| Auth
    
    Gateway1 -->|Route| Chat1["Chat Service Instance 1 Port 6004"]
    Gateway1 -->|Route| Chat2["Chat Service Instance 2 Port 6014"]
    Gateway1 -->|Route| Chat3["Chat Service Instance 3 Port 6024"]
    
    Gateway2 -->|Route| Chat1
    Gateway2 -->|Route| Chat2
    Gateway2 -->|Route| Chat3
    
    Gateway1 -->|Route| WS1["WS Gateway Instance 1 Port 6005"]
    Gateway1 -->|Route| WS2["WS Gateway Instance 2 Port 6015"]
    Gateway1 -->|Route| WS3["WS Gateway Instance 3 Port 6025"]
    
    Gateway2 -->|Route| WS1
    Gateway2 -->|Route| WS2
    Gateway2 -->|Route| WS3
    
    Gateway1 -->|Route| Courses1["Courses Service Instance 1 Port 8085"]
    Gateway1 -->|Route| Courses2["Courses Service Instance 2 Port 8086"]
    
    Gateway2 -->|Route| Courses1
    Gateway2 -->|Route| Courses2
    
    Gateway1 -->|Route| Payment["Payment Service Single Instance"]
    Gateway2 -->|Route| Payment
    
    Gateway1 -->|Route| Recommend["Recommendation Service Single Instance"]
    Gateway2 -->|Route| Recommend
    
    Chat1 -->|Shared| DB["PostgreSQL Single Instance"]
    Chat2 -->|Shared| DB
    Chat3 -->|Shared| DB
    
    Chat1 -->|Shared| RedisDB["Redis Single Instance"]
    Chat2 -->|Shared| RedisDB
    Chat3 -->|Shared| RedisDB
    
    Chat1 -->|Shared| Kafka["Kafka Single Instance"]
    Chat2 -->|Shared| Kafka
    Chat3 -->|Shared| Kafka
```

---

## SECURITY LAYERS

```mermaid
graph TB
    Client["Client"]
    
    Client -->|HTTPS| Layer1["Layer 1: TLS/SSL Encryption in transit"]
    
    Layer1 -->|Request| Layer2["Layer 2: API Gateway Arcjet Bot Detection VPN Blocking CORS Protection"]
    
    Layer2 -->|Request| Layer3["Layer 3: JWT Verification Signature validation Token expiry check"]
    
    Layer3 -->|Request| Layer4["Layer 4: Service Authorization Role-based access control Resource ownership check"]
    
    Layer4 -->|Request| Layer5["Layer 5: Data Layer Parameterized queries SQL injection prevention"]
    
    Layer5 -->|Response| Layer6["Layer 6: Response Encryption Gzip compression Sensitive data masking"]
    
    Layer6 -->|Response| Client
    
    Auth["Auth Service"]
    Auth -->|2FA| TOTP["TOTP Verification Backup codes"]
    Auth -->|Device| Fingerprint["Device Fingerprinting Trusted devices"]
    Auth -->|Password| Bcrypt["Bcrypt Hashing Work factor 12"]
    
    Payment["Payment Service"]
    Payment -->|Tokens| Paymob["Paymob Tokenization PCI-DSS Compliant"]
    Payment -->|Webhook| HMAC["HMAC Verification Signature validation"]
    
    Courses["Courses Service"]
    Courses -->|QR| QRSig["HMAC-SHA256 QR Signature"]
    Courses -->|Attendance| Geofence["Geofence Validation Device Fingerprint Emulator Detection"]
```

---

## MONITORING & OBSERVABILITY

```mermaid
graph TB
    Services["All Services"]
    
    Services -->|Logs| ELK["ELK Stack Elasticsearch Logstash Kibana"]
    
    Services -->|Metrics| Prometheus["Prometheus Metrics Collection"]
    
    Prometheus -->|Visualize| Grafana["Grafana Dashboards"]
    
    Services -->|Traces| Jaeger["Jaeger Distributed Tracing"]
    
    Services -->|Health| HealthCheck["Health Check Endpoint /health"]
    
    HealthCheck -->|Monitored by| Monitoring["Monitoring System Alerts on failures"]
    
    Monitoring -->|Alert| Team["DevOps Team"]
```

---

## DEPLOYMENT ARCHITECTURE

```mermaid
graph TB
    subgraph "Development"
        DevDocker["Docker Compose All services locally"]
    end
    
    subgraph "Staging"
        StagingK8s["Kubernetes Cluster Staging namespace"]
    end
    
    subgraph "Production"
        ProdK8s["Kubernetes Cluster Production namespace"]
        ProdLB["Load Balancer"]
        ProdDB["PostgreSQL Replicated"]
        ProdRedisDB["Redis Cluster"]
        ProdKafka["Kafka Cluster"]
    end
    
    DevDocker -->|CI/CD| StagingK8s
    StagingK8s -->|Approved| ProdK8s
    
    ProdLB -->|Route| ProdK8s
    ProdK8s -->|Query| ProdDB
    ProdK8s -->|Cache| ProdRedisDB
    ProdK8s -->|Events| ProdKafka
```

---

## SUMMARY: HOW EVERYTHING WORKS TOGETHER

1. **Client sends request** → API Gateway
2. **API Gateway** routes to appropriate service
3. **Service validates JWT** with Auth Service
4. **Service queries PostgreSQL** for data
5. **Service caches in Redis** for performance
6. **Service emits Kafka event** for other services
7. **Notification Service consumes event** and sends notifications
8. **WS Gateway consumes event** and broadcasts to connected clients
9. **Response sent back** to client via API Gateway
10. **External services** (FCM, Paymob, Resend, Cloudinary, Gemini) handle specialized tasks

**Result**: A fully integrated, scalable, real-time education platform!
