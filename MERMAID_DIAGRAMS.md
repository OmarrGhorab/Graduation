# Mermaid Diagrams - Services & Database Architecture

## 1. OVERALL SYSTEM ARCHITECTURE

```mermaid
graph TB
    Client["Client Applications<br/>(Web, iOS, Android)"]
    
    Client -->|HTTPS| Gateway["API Gateway<br/>Port 6000<br/>Node.js + Express"]
    
    Gateway -->|Route| Auth["Auth Service<br/>Port 6001<br/>Node.js + Prisma"]
    Gateway -->|Route| Notif["Notification Service<br/>Port 6003<br/>Node.js + Firebase"]
    Gateway -->|Route| Chat["Chat Service<br/>Ports 6004/14/24<br/>Go + Fiber (3x)"]
    Gateway -->|Route| Courses["Courses Service<br/>Ports 8085/86<br/>Go + Fiber (2x)"]
    Gateway -->|Route| Payment["Payment Service<br/>Port 8090<br/>Go + Fiber"]
    Gateway -->|Route| Recommend["Recommendation Svc<br/>Port 8095<br/>Python + FastAPI"]
    
    WS["WS Gateway<br/>Ports 6005/15/25<br/>Go + Fiber (3x)"]
    Client -->|WebSocket| WS
    
    Auth -->|Query/Store| DB["PostgreSQL<br/>Port 5432"]
    Notif -->|Query/Store| DB
    Chat -->|Query/Store| DB
    Courses -->|Query/Store| DB
    Payment -->|Query/Store| DB
    Recommend -->|Query/Store| DB
    
    Auth -->|Cache/Session| Redis["Redis<br/>Port 6379"]
    Chat -->|Cache| Redis
    Courses -->|QR Tokens| Redis
    Payment -->|Cart| Redis
    Recommend -->|Cache| Redis
    WS -->|Pub/Sub| Redis
    
    Auth -->|Events| Kafka["Kafka<br/>Port 9092"]
    Notif -->|Consume| Kafka
    Chat -->|Events| Kafka
    Courses -->|Events| Kafka
    Payment -->|Events| Kafka
    WS -->|Consume| Kafka
    
    Notif -->|Push| FCM["Firebase FCM"]
    Notif -->|Email| Resend["Resend"]
    Payment -->|Email| Resend
    
    Payment -->|Process| Paymob["Paymob API"]
    
    Chat -->|Media| Cloudinary["Cloudinary CDN"]
    Courses -->|Media| Cloudinary
    Auth -->|Images| Cloudinary
    
    Recommend -->|AI| Gemini["Google Gemini AI"]
    
    Gateway -->|Security| Arcjet["Arcjet<br/>Bot Detection"]

    subgraph "Observability & Monitoring"
        Prometheus["Prometheus<br/>Port 9090"]
        Grafana["Grafana<br/>Port 3001"]
        Loki["Loki<br/>Port 3100"]
        Jaeger["Jaeger<br/>Port 16686"]
        OTel["OTel Collector<br/>Port 4318"]
        Sentry["Sentry<br/>Cloud"]
    end
    
    Prometheus -->|Scrape Metrics| Gateway
    Prometheus -->|Scrape Metrics| Auth
    Prometheus -->|Scrape Metrics| Chat
    Prometheus -->|Scrape Metrics| Courses
    Prometheus -->|Scrape Metrics| Payment
    Prometheus -->|Scrape Metrics| Recommend
    
    Gateway -->|Send Traces| OTel
    Auth -->|Send Traces| OTel
    Chat -->|Send Traces| OTel
    Courses -->|Send Traces| OTel
    Payment -->|Send Traces| OTel
    Recommend -->|Send Traces| OTel
    
    OTel -->|Export Traces| Jaeger
    Grafana -->|Query| Prometheus
    Grafana -->|Query| Loki
    
    Gateway -->|Report Errors| Sentry
    Auth -->|Report Errors| Sentry
    Chat -->|Report Errors| Sentry
    Courses -->|Report Errors| Sentry
    Payment -->|Report Errors| Sentry
    Recommend -->|Report Errors| Sentry
```

---

## 2. AUTH SERVICE ARCHITECTURE

```mermaid
graph LR
    Client["Client"]
    
    Client -->|POST /auth/register| Handler["HTTP Handler<br/>register"]
    Client -->|POST /auth/login| Handler
    Client -->|POST /auth/2fa/verify| Handler
    Client -->|POST /auth/refresh| Handler
    Client -->|GET /profile| Handler
    
    Handler -->|Validate| Middleware["Auth Middleware<br/>JWT Verification"]
    
    Middleware -->|Use Case| Register["Register Use Case"]
    Middleware -->|Use Case| Login["Login Use Case"]
    Middleware -->|Use Case| TwoFA["2FA Use Case"]
    Middleware -->|Use Case| Profile["Profile Use Case"]
    
    Register -->|Hash| Bcrypt["Bcrypt"]
    Login -->|Verify| Bcrypt
    
    Register -->|Generate| JWT["JWT Token"]
    Login -->|Generate| JWT
    TwoFA -->|Verify| TOTP["TOTP 2FA"]
    
    Register -->|Store| UserRepo["User Repository"]
    Login -->|Query| UserRepo
    Profile -->|Query| UserRepo
    
    UserRepo -->|Persist| DB["PostgreSQL<br/>Users Table"]
    
    Register -->|Cache| SessionRepo["Session Repository"]
    Login -->|Cache| SessionRepo
    
    SessionRepo -->|Store| Redis["Redis<br/>Session Storage"]
    
    Register -->|Send| Email["Email Service<br/>Resend"]
    
    Register -->|Emit| Kafka["Kafka<br/>user.registered"]
```

---

## 3. NOTIFICATION SERVICE ARCHITECTURE

```mermaid
graph LR
    KafkaConsumer["Kafka Consumer<br/>Listen to Events"]
    
    KafkaConsumer -->|course.enrolled| Handler1["Course Enrolled Handler"]
    KafkaConsumer -->|payment.completed| Handler2["Payment Completed Handler"]
    KafkaConsumer -->|attendance.recorded| Handler3["Attendance Handler"]
    KafkaConsumer -->|lesson.started| Handler4["Lesson Started Handler"]
    
    Handler1 -->|Create| Notif["Notification Entity"]
    Handler2 -->|Create| Notif
    Handler3 -->|Create| Notif
    Handler4 -->|Create| Notif
    
    Notif -->|Store| NotifRepo["Notification Repository"]
    NotifRepo -->|Persist| DB["PostgreSQL<br/>Notifications Table"]
    
    Notif -->|Get Token| TokenRepo["FCM Token Repository"]
    TokenRepo -->|Query| DB
    
    Notif -->|Send| FCM["Firebase FCM<br/>Push Notification"]
    Notif -->|Send| Email["Resend<br/>Email"]
    
    Client["Client"]
    Client -->|POST /register-token| Handler5["Register Token Handler"]
    Handler5 -->|Store| TokenRepo
    
    Client -->|GET /notifications| Handler6["Get Notifications Handler"]
    Handler6 -->|Query| NotifRepo
```

---

## 4. CHAT SERVICE ARCHITECTURE

```mermaid
graph LR
    Client1["User A"]
    Client2["User B"]
    
    Client1 -->|Create Chat| Handler1["Create Conversation Handler"]
    Client1 -->|Send Message| Handler2["Send Message Handler"]
    Client1 -->|Typing| Handler3["Typing Indicator Handler"]
    
    Handler1 -->|Create| ConvEntity["Conversation Entity"]
    Handler2 -->|Create| MsgEntity["Message Entity"]
    Handler3 -->|Update| TypingEntity["Typing State"]
    
    ConvEntity -->|Store| ConvRepo["Conversation Repository"]
    MsgEntity -->|Store| MsgRepo["Message Repository"]
    TypingEntity -->|Cache| Redis["Redis<br/>Typing Indicators"]
    
    ConvRepo -->|Persist| DB["PostgreSQL<br/>Conversations"]
    MsgRepo -->|Persist| DB
    
    Handler2 -->|Emit| Kafka["Kafka<br/>message.created"]
    
    Kafka -->|Consume| WS["WS Gateway"]
    WS -->|Broadcast| Client2
    
    Handler2 -->|Upload| Cloudinary["Cloudinary<br/>Media Storage"]
    
    Client2 -->|Mark Read| Handler4["Mark Read Handler"]
    Handler4 -->|Update| MsgRepo
    Handler4 -->|Emit| Kafka2["Kafka<br/>message.read"]
```

---

## 5. COURSES & ATTENDANCE SERVICE ARCHITECTURE

```mermaid
graph TB
    Teacher["Teacher"]
    Student["Student"]
    
    Teacher -->|Create Course| CourseHandler["Create Course Handler"]
    Teacher -->|Start Lesson| LessonHandler["Start Lesson Handler"]
    Teacher -->|End Lesson| EndHandler["End Lesson Handler"]
    
    CourseHandler -->|Create| CourseEntity["Course Entity"]
    LessonHandler -->|Create| LessonEntity["Lesson Entity"]
    
    CourseEntity -->|Store| CourseRepo["Course Repository"]
    LessonEntity -->|Store| LessonRepo["Lesson Repository"]
    
    CourseRepo -->|Persist| DB["PostgreSQL<br/>Courses Table"]
    LessonRepo -->|Persist| DB
    
    LessonHandler -->|Start Worker| QRWorker["QR Rotation Worker<br/>Every 30s"]
    QRWorker -->|Generate| QRToken["QR Token<br/>HMAC-SHA256"]
    QRToken -->|Store| Redis["Redis<br/>QR Tokens<br/>TTL: 35s"]
    
    Student -->|Scan QR| ScanHandler["Scan QR Handler"]
    
    ScanHandler -->|Verify| JWTVerify["JWT Verification"]
    ScanHandler -->|Verify| AuthService["Auth Service<br/>Device Verification"]
    ScanHandler -->|Verify| QRVerify["QR Signature Verify"]
    ScanHandler -->|Validate| Geofence["Geofence Check<br/>Haversine"]
    ScanHandler -->|Check| DeviceFingerprint["Device Fingerprint"]
    ScanHandler -->|Detect| Emulator["Emulator Detection"]
    
    ScanHandler -->|Record| AttendanceRepo["Attendance Repository"]
    AttendanceRepo -->|Persist| DB
    
    ScanHandler -->|Emit| Kafka["Kafka<br/>attendance.recorded"]
    
    EndHandler -->|Auto-Mark| AbsentRepo["Absence Repository"]
    AbsentRepo -->|Persist| DB
    
    EndHandler -->|Calculate| ProgressCalc["Progress Calculator<br/>Weighted Formula"]
    ProgressCalc -->|Store| ProgressRepo["Progress Repository"]
    ProgressRepo -->|Persist| DB
```

---

## 6. PAYMENT SERVICE ARCHITECTURE

```mermaid
graph TB
    Student["Student"]
    
    Student -->|Add to Cart| CartHandler["Add to Cart Handler"]
    Student -->|View Cart| ViewHandler["View Cart Handler"]
    Student -->|Checkout| CheckoutHandler["Checkout Handler"]
    
    CartHandler -->|Create| CartEntity["Cart Entity"]
    CartEntity -->|Store| CartRepo["Cart Repository"]
    CartRepo -->|Persist| DB["PostgreSQL<br/>Carts Table"]
    CartRepo -->|Cache| Redis["Redis<br/>Cart Session"]
    
    CheckoutHandler -->|Validate| Validator["Cart Validator"]
    Validator -->|Query| CourseService["Courses Service<br/>Get Course Details"]
    
    CheckoutHandler -->|Create| OrderEntity["Payment Order Entity"]
    OrderEntity -->|Store| OrderRepo["Order Repository"]
    OrderRepo -->|Persist| DB
    
    CheckoutHandler -->|Auto-Enroll| CourseService
    
    CheckoutHandler -->|Create| PaymobOrder["Paymob Order"]
    PaymobOrder -->|Send| Paymob["Paymob API"]
    
    Paymob -->|Return| PaymentURL["Payment URL"]
    PaymentURL -->|Send| Student
    
    Student -->|Complete Payment| Paymob
    
    Paymob -->|Webhook| WebhookHandler["Webhook Handler"]
    WebhookHandler -->|Verify| HMACVerify["HMAC Verification"]
    
    HMACVerify -->|Update| OrderRepo
    OrderRepo -->|Persist| DB
    
    WebhookHandler -->|Activate| CourseService
    WebhookHandler -->|Create| SubEntity["Subscription Entity"]
    SubEntity -->|Store| SubRepo["Subscription Repository"]
    SubRepo -->|Persist| DB
    
    WebhookHandler -->|Clear| CartRepo
    WebhookHandler -->|Emit| Kafka["Kafka<br/>payment.completed"]
    WebhookHandler -->|Send| Email["Resend<br/>Receipt Email"]
    
    BillingJob["Background Job<br/>Subscription Billing<br/>Daily 2 AM"]
    BillingJob -->|Find Due| SubRepo
    BillingJob -->|Create| RenewalOrder["Renewal Order"]
    BillingJob -->|Send| Email
```

---

## 7. RECOMMENDATION SERVICE ARCHITECTURE

```mermaid
graph LR
    Client["Client"]
    
    Client -->|GET /recommendations| RecHandler["Get Recommendations Handler"]
    Client -->|POST /chatbot/message| ChatHandler["Chat Handler"]
    Client -->|GET /reports/progress| ReportHandler["Report Handler"]
    
    RecHandler -->|Check| RedisCache["Redis Cache<br/>6h TTL"]
    RedisCache -->|Hit| Client
    
    RedisCache -->|Miss| RecService["Recommendation Service"]
    RecService -->|Query| UserData["Get User Data<br/>Auth Service"]
    RecService -->|Query| CourseData["Get Course Data<br/>Courses Service"]
    
    RecService -->|Call| Gemini["Google Gemini AI"]
    Gemini -->|Return| Recommendations["Recommendations"]
    
    Recommendations -->|Store| RecRepo["Recommendation Repository"]
    RecRepo -->|Persist| DB["PostgreSQL<br/>Recommendations"]
    
    Recommendations -->|Cache| RedisCache
    Recommendations -->|Send| Client
    
    ChatHandler -->|Get Session| SessionRepo["Chat Session Repository"]
    SessionRepo -->|Query| DB
    
    ChatHandler -->|Build Context| ContextBuilder["Context Builder<br/>Last 20 messages"]
    ContextBuilder -->|Call| Gemini
    
    Gemini -->|Stream| SSE["Server-Sent Events<br/>Real-time Response"]
    SSE -->|Send| Client
    
    ChatHandler -->|Save| MsgRepo["Message Repository"]
    MsgRepo -->|Persist| DB
    
    ReportHandler -->|Generate| ReportGen["Report Generator<br/>AI Analysis"]
    ReportGen -->|Create PDF| FPDF["FPDF2<br/>PDF Generation"]
    FPDF -->|Send| Client
```

---

## 8. WEBSOCKET GATEWAY ARCHITECTURE

```mermaid
graph TB
    Client1["Client 1"]
    Client2["Client 2"]
    Client3["Client 3"]
    
    Client1 -->|WebSocket| WSGateway["WS Gateway<br/>Fiber WebSocket"]
    Client2 -->|WebSocket| WSGateway
    Client3 -->|WebSocket| WSGateway
    
    WSGateway -->|Store Connection| Redis["Redis<br/>Connection Mapping<br/>ws:user:{id}:connections"]
    
    WSGateway -->|Subscribe| KafkaConsumer["Kafka Consumer<br/>Listen to Events"]
    
    KafkaConsumer -->|message.created| EventHandler1["Message Event Handler"]
    KafkaConsumer -->|typing.started| EventHandler2["Typing Event Handler"]
    KafkaConsumer -->|user.presence| EventHandler3["Presence Event Handler"]
    
    EventHandler1 -->|Get Recipients| Redis
    EventHandler1 -->|Broadcast| Client1
    EventHandler1 -->|Broadcast| Client2
    
    EventHandler2 -->|Get Recipients| Redis
    EventHandler2 -->|Broadcast| Client1
    
    EventHandler3 -->|Update| Redis
    EventHandler3 -->|Broadcast| Client1
    EventHandler3 -->|Broadcast| Client2
    EventHandler3 -->|Broadcast| Client3
```

---

## 9. API GATEWAY ARCHITECTURE

```mermaid
graph TB
    Client["Client"]
    
    Client -->|Request| Gateway["API Gateway<br/>Express.js"]
    
    Gateway -->|Middleware 1| Compression["Compression Middleware<br/>gzip > 1KB"]
    Compression -->|Middleware 2| Timeout["Timeout Middleware<br/>30s"]
    Timeout -->|Middleware 3| CORS["CORS Middleware<br/>Whitelist Origins"]
    CORS -->|Middleware 4| Arcjet["Arcjet Middleware<br/>Bot Detection<br/>VPN Blocking"]
    
    Arcjet -->|Route| Router["Router<br/>Path Matching"]
    
    Router -->|/api/v1/notifications| NotifService["Notification Service<br/>6003"]
    Router -->|/api/v1/location| NotifService
    Router -->|/api/v1/courses| CourseService["Courses Service<br/>8085/8086"]
    Router -->|/api/v1/payments| PaymentService["Payment Service<br/>8090"]
    Router -->|/api/v1/chat| ChatService["Chat Service<br/>6004/6014/6024"]
    Router -->|/api/v1/recommendations| RecService["Recommendation Service<br/>8095"]
    Router -->|/* catch-all| AuthService["Auth Service<br/>6001"]
    
    NotifService -->|Response| Gateway
    CourseService -->|Response| Gateway
    PaymentService -->|Response| Gateway
    ChatService -->|Response| Gateway
    RecService -->|Response| Gateway
    AuthService -->|Response| Gateway
    
    Gateway -->|Error Handling| ErrorMiddleware["Error Middleware<br/>Consistent Responses"]
    ErrorMiddleware -->|Response| Client
```

---

## 10. COMPLETE SERVICE INTEGRATION FLOW

```mermaid
graph TB
    subgraph "Client Layer"
        Web["Web Browser"]
        Mobile["Mobile App"]
    end
    
    subgraph "API Layer"
        Gateway["API Gateway<br/>Port 6000"]
    end
    
    subgraph "Service Layer"
        Auth["Auth Service<br/>6001"]
        Notif["Notification<br/>6003"]
        Chat["Chat<br/>6004/14/24"]
        WS["WS Gateway<br/>6005/15/25"]
        Courses["Courses<br/>8085/86"]
        Payment["Payment<br/>8090"]
        Recommend["Recommend<br/>8095"]
    end
    
    subgraph "Data Layer"
        DB["PostgreSQL"]
        Redis["Redis"]
        Kafka["Kafka"]
    end
    
    subgraph "External Services"
        FCM["Firebase FCM"]
        Paymob["Paymob"]
        Resend["Resend"]
        Cloudinary["Cloudinary"]
        Gemini["Gemini AI"]
        Arcjet["Arcjet"]
    end
    
    Web -->|HTTPS| Gateway
    Mobile -->|HTTPS| Gateway
    Mobile -->|WebSocket| WS
    
    Gateway -->|Route| Auth
    Gateway -->|Route| Notif
    Gateway -->|Route| Chat
    Gateway -->|Route| Courses
    Gateway -->|Route| Payment
    Gateway -->|Route| Recommend
    
    Auth -->|Query| DB
    Notif -->|Query| DB
    Chat -->|Query| DB
    Courses -->|Query| DB
    Payment -->|Query| DB
    Recommend -->|Query| DB
    
    Auth -->|Cache| Redis
    Chat -->|Cache| Redis
    Courses -->|Cache| Redis
    Payment -->|Cache| Redis
    Recommend -->|Cache| Redis
    WS -->|Pub/Sub| Redis
    
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
