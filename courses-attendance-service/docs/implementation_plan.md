# Implementation Plan File: Courses & Attendance Microservice

## Summary
Create one implementation plan document at `courses-attendance-service/docs/implementation_plan.md` for a new Go/Fiber microservice that manages courses, lessons, attendance (secure rotating QR), parent approvals, progress, calendar updates, and notifications.

This plan locks the following decisions:
1. Plan format: single Markdown plan file.
2. Messaging backend: Kafka.
3. Device binding: add new internal Auth API.
4. Notifications: event-driven only (Kafka consumer in notification-service).
5. Emulator detection: mobile attestation required (Play Integrity/App Attest).

## Scope
1. Build new `courses-attendance-service` in Go with Clean Architecture + DDD-ish module boundaries.
2. Add API Gateway routing for new endpoints.
3. Add Docker Compose service entries for local multi-instance deployment.
4. Add Auth Service internal API contract for token+session+device+attestation verification.
5. Add Notification Service Kafka consumer for courses/attendance events.
6. Include migrations, Redis usage, Kafka producers, workers, and full test strategy.

## Out Of Scope
1. UI/mobile implementation.
2. Billing/payment gateway flow beyond `is_paid` and `price` fields.
3. Full anti-fraud device intelligence platform (only required attestation + existing auth session/device data).

## Public APIs, Interfaces, and Type Changes
1. New external API (through gateway) from courses service under `/api/v1`:
   - `POST /courses`
   - `GET /courses/:id`
   - `PATCH /courses/:id`
   - `POST /courses/:id/enroll`
   - `POST /courses/:id/assistants`
   - `POST /lessons`
   - `POST /lessons/:id/start`
   - `POST /lessons/:id/end`
   - `POST /lessons/:id/cancel`
   - `POST /lessons/:id/reschedule`
   - `POST /attendance/scan`
   - `GET /attendance/lesson/:id`
   - `GET /attendance/student/:id`
   - `POST /absence/request`
   - `POST /absence/respond`
   - `GET /calendar/me`
2. New Auth Service internal endpoint:
   - `POST /api/v1/internal/attendance/verify-context`
   - Request includes: `access_token`, `device_id`, `device_fingerprint`, `attestation_token`, `ip`, `user_agent`.
   - Response includes: `valid`, `user_id`, `role`, `session_jti`, `device_verified`, `emulator_detected`, `multi_device_violation`, `shared_device_violation`, `reasons`.
3. Existing internal Auth endpoint reused:
   - `GET /api/v1/parent-link/verify-link?parentId=...&childId=...`
4. Notification Service interface addition:
   - Kafka consumer group `notification-courses-v1` consuming new courses topics.
5. API Gateway config additions:
   - `COURSES_SERVICE_URLS` in `api-gateway/src/config/index.ts`.
   - Route proxy entries for `/api/v1/courses`, `/api/v1/lessons`, `/api/v1/attendance`, `/api/v1/absence`, `/api/v1/calendar`.

## Architecture and Project Structure
Use this target structure:
`courses-attendance-service/`
1. `cmd/server/main.go`
2. `internal/bootstrap` (wiring, DI)
3. `internal/config`
4. `internal/interfaces/http` (Fiber handlers, DTOs, validators, middleware)
5. `internal/application` (use-cases)
6. `internal/domain/course`
7. `internal/domain/lesson`
8. `internal/domain/attendance`
9. `internal/domain/absence`
10. `internal/domain/progress`
11. `internal/domain/calendar`
12. `internal/infrastructure/persistence/postgres` (GORM repos, transactions)
13. `internal/infrastructure/cache/redis`
14. `internal/infrastructure/messaging/kafka`
15. `internal/infrastructure/authclient`
16. `internal/infrastructure/notificationevents`
17. `internal/infrastructure/clock` (UTC-only time provider)
18. `internal/worker` (scheduled/background jobs)
19. `migrations` (SQL-first, no AutoMigrate in production)
20. `tests/unit` and `tests/integration`

## Data Model and Migrations (PostgreSQL, UTC)
Tables and enums:
1. `courses`
2. `course_assistants`
3. `subjects`
4. `lessons`
5. `enrollments`
6. `attendance_sessions`
7. `attendance_qr_tokens`
8. `attendance_records`
9. `absence_requests`
10. `progress_snapshots`
11. `audit_logs` (immutable append-only)

Key constraints:
1. UUID primary keys everywhere.
2. `enrollments` unique `(course_id, user_id)`.
3. `attendance_records` unique `(lesson_id, student_id)`.
4. `attendance_qr_tokens` unique `(lesson_id, nonce)`.
5. `course_assistants` unique `(course_id, assistant_id)`.
6. All timestamps are `timestamptz` stored in UTC.

Enums:
1. `delivery_type`: `ONLINE | OFFLINE`
2. `course_status`: `ACTIVE | PAUSED | ARCHIVED`
3. `lesson_status`: `SCHEDULED | LIVE | COMPLETED | CANCELED`
4. `attendance_status`: `PRESENT | LATE | ABSENT | EXCUSED`
5. `absence_reason_type`: `PARENT_EXCUSE | MEDICAL | EMERGENCY`
6. `absence_status`: `PENDING | APPROVED | REJECTED`

## Redis Design
Keys:
1. `attendance:lesson:{lesson_id}:active_qr` (current token payload, TTL = rotation seconds)
2. `attendance:lesson:{lesson_id}:nonce:{nonce}` (single-use QR nonce, TTL until token expiry)
3. `attendance:lock:scan:{lesson_id}:{student_id}` (short lock, TTL 5s)
4. `attendance:ratelimit:scan:{user_id}` (token bucket counters)
5. `attendance:presence:lesson:{lesson_id}:user:{user_id}` (optional realtime)
6. `jobs:attendance:close:{lesson_id}` (close trigger marker)
7. `jobs:lesson:upcoming:{lesson_id}` (upcoming reminder marker)

## Attendance, QR, and Security Flow
1. Lesson start endpoint validates RBAC (`TEACHER` owner or assigned `ASSISTANT`), updates lesson to `LIVE`, creates attendance session, starts QR rotator worker.
2. QR rotates every 30 seconds. Payload includes `lesson_id`, `issued_at`, `expires_at`, `nonce`, `signature`.
3. Signature uses HMAC-SHA256 with secret `QR_SIGNING_SECRET`, canonical payload string.
4. Scan endpoint flow:
   - Verify JWT signature.
   - Call Auth internal `verify-context` endpoint.
   - Verify Redis QR nonce exists and is unused.
   - Verify signature and expiry using server UTC time.
   - Validate enrollment and lesson state.
   - For offline lessons, compute Haversine distance and enforce `distance <= radius_m`.
   - Acquire Redis scan lock.
   - Upsert attendance record idempotently in DB transaction.
   - Mark nonce consumed and emit event.
5. Status rules:
   - `PRESENT` when `scan_time <= starts_at + attendance_window_minutes`.
   - `LATE` when after window and before `ends_at`.
   - `ABSENT` assigned automatically at close for no valid scan.
   - `EXCUSED` only after approved absence request.
6. Non-negotiables enforced:
   - no client time trust
   - no static QR
   - no attendance update without verification
   - no location-only trust (must pass auth/device checks too)

## Parent Approval and Absence Flow
1. Student may submit `POST /absence/request`.
2. System auto-creates pending absence tasks for ABSENT students with linked parent.
3. Parent responds via `POST /absence/respond`.
4. Parent-child relation verified via auth-service `verify-link`.
5. Approved request updates attendance to `EXCUSED`, emits event, and notifies teacher/student.
6. Rejected request keeps `ABSENT`.

## Progress and Calendar Rules
1. Progress recomputed on lesson completion and attendance changes.
2. Formula:
   - `completion_ratio = completed_lessons / total_lessons`
   - `attendance_ratio = weighted_attendance_points / total_lessons`
   - `progress = ((1 - attendance_weight) * completion_ratio + attendance_weight * attendance_ratio) * 100`
3. Default `attendance_weight = 0.30`.
4. Attendance points: `PRESENT=1.0`, `LATE=0.7`, `EXCUSED=0.8`, `ABSENT=0.0`.
5. Calendar feed built from enrolled lessons and emits on lesson cancel/reschedule.

## Kafka Event Contracts
Topics:
1. `courses.lesson.started.v1`
2. `courses.lesson.ended.v1`
3. `courses.lesson.canceled.v1`
4. `courses.lesson.rescheduled.v1`
5. `courses.attendance.recorded.v1`
6. `courses.attendance.finalized.v1`
7. `courses.absence.requested.v1`
8. `courses.absence.reviewed.v1`
9. `courses.progress.updated.v1`
10. `courses.notification.requested.v1`

Event envelope:
1. `event_id` UUID
2. `event_type`
3. `occurred_at` UTC ISO8601
4. `aggregate_id`
5. `actor_user_id`
6. `payload`
7. `trace_id`

Delivery:
1. Producer retries with backoff.
2. Idempotent consumer handling by `event_id`.
3. At-least-once semantics with replay-safe handlers.

## Implementation Phases
1. Phase 1: scaffold service, config, health endpoint, Fiber middleware, DI, UTC clock abstraction.
2. Phase 2: create migrations and GORM entities/repositories with transaction manager.
3. Phase 3: implement course/subject/lesson/enrollment/assistant use-cases and HTTP routes.
4. Phase 4: implement attendance session lifecycle, QR rotation worker, secure scan validation.
5. Phase 5: implement absence request/respond flow and parent-link verification integration.
6. Phase 6: implement progress calculation, calendar feed, lesson reminder/cancel/reschedule workflows.
7. Phase 7: add Kafka producers and notification-service consumer integration.
8. Phase 8: integrate API gateway routing + docker-compose instances and env wiring.
9. Phase 9: add observability, audit logging, rate limiting, and hardening.
10. Phase 10: complete automated tests and run end-to-end verification.

## Test Cases and Scenarios
Unit tests:
1. QR signature generation/validation and expiry windows.
2. Haversine distance correctness and geofence edge.
3. Attendance status mapping by server time.
4. RBAC policy checks for teacher/assistant/student/parent.
5. Progress formula with weighted attendance.
6. Idempotency logic under duplicate scan requests.

Integration tests:
1. Start lesson -> QR rotation -> valid scan -> attendance persisted -> event emitted.
2. Expired QR scan rejected.
3. Replay nonce scan rejected.
4. Device mismatch and emulator attestation failure rejected.
5. Offline geofence violation rejected.
6. Concurrent duplicate scans only create one attendance record.
7. Lesson end auto-marks absentees.
8. Parent approval updates ABSENT to EXCUSED.
9. Reschedule/cancel emits events and updates calendar endpoint.
10. Kafka consumer replay does not duplicate notifications.

Security tests:
1. Tampered QR payload rejected.
2. JWT with invalid signature rejected.
3. Missing internal auth secret rejected in service-to-service calls.
4. Rate limit enforced on scan endpoint.
5. Audit log write on all attendance/absence state transitions.

## Rollout and Monitoring
1. Deploy behind gateway with canary instance first.
2. Enable feature flags:
   - `ATTENDANCE_QR_ENABLED`
   - `ATTENDANCE_ATTESTATION_REQUIRED`
   - `ABSENCE_PARENT_APPROVAL_ENABLED`
3. Metrics:
   - scan success/failure counts by reason
   - QR rotation lag
   - attendance finalize duration
   - auth verification latency
   - Kafka publish failures/retries
4. Logs:
   - structured JSON with `trace_id`, `lesson_id`, `student_id`, `device_id`.
5. Alerts:
   - spike in fraud rejections
   - high Redis lock contention
   - worker lag past lesson end.

## Assumptions and Defaults
1. Go version is latest stable at implementation time; repo currently uses Go 1.23.x for existing Go services.
2. Service base path is `/api/v1` through gateway.
3. `x-internal-service-secret` remains the internal service auth mechanism.
4. Notification-service is extended to consume Kafka topics (events-only model, no direct fallback).
5. Auth-service team will implement `POST /api/v1/internal/attendance/verify-context`.
6. Mobile clients provide `x-device-id`, `x-device-fingerprint`, GPS headers, and attestation token in attendance scan requests.
7. UTC is enforced for DB writes, event timestamps, and scheduling logic.
8. SQL migrations are the source of truth; production will not rely on GORM AutoMigrate.
