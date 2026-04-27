# Courses & Attendance Service API Specification

This document provides a comprehensive overview of the **Courses & Attendance Service**, its technical architecture, core logic, and detailed API endpoints.

## 🏗 Architecture Overview
- **Language**: Go 1.21+
- **Framework**: [Fiber v2](https://gofiber.io/) (Fast HTTP framework)
- **Database**: PostgreSQL (GORM)
- **Cache**: Redis (for QR tokens and real-time state)
- **Messaging**: Kafka (Event-driven communication with other services)
- **Communication**: REST API (with internal service secrets for inter-service calls)

## 🌐 Base URL
`http://courses-service:8085/api/v1` (Proxied via API Gateway)

---

## 🛠 Core Logic & Features

### 1. Dynamic QR Attendance System
- **QR Rotation**: Tokens rotate every X seconds (default 30s) to prevent sharing/screenshot abuse.
- **HMAC Signing**: All QR tokens are signed using a `QR_SIGNING_SECRET`.
- **Validation Chain**:
    - **Geofencing**: Checks if student's Lat/Lng is within the allowed radius of the lesson location.
    - **Device Fingerprinting**: Prevents multiple students from scanning with the same device.
    - **Emulator Detection**: Blocks scanners running on virtual devices.
    - **Idempotency**: Prevents double-scanning for the same lesson.

### 2. Progress Calculation
- **Weighted Progress**: Calculates student overall progress based on:
    - Attendance (e.g., 70% weight)
    - Completed lessons
    - Excused vs Unexcused absences.
- **Auto-Recomputation**: Triggered automatically when attendance is recorded or updated.

### 3. Kafka Event System (Fan-out)
- Emits events to the `notification-service` for:
    - **Lesson Events**: `started`, `ended`, `canceled`, `rescheduled`.
    - **Attendance Events**: `recorded`, `finalized`.
    - **Absence Events**: `requested`, `reviewed`.
    - **Progress Events**: `updated`.

---

## 📍 API Endpoints

### 1. Course Management
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/courses` | List all courses (Supports filtering by `subjectId` query) |
| `GET` | `/courses/my` | List courses the current user is enrolled in |
| `GET` | `/courses/my-subjects` | List subjects (categories) the current user is enrolled in |
| `POST` | `/courses` | Create a new course |
| `GET` | `/courses/:id` | Get details for a specific course |
| `PATCH` | `/courses/:id` | Update course details (Location, Radius, etc.) |
| `POST` | `/courses/:id/enroll` | Enroll a student in a course |
| `POST` | `/courses/:id/assistants` | Add an assistant with specific permissions |

**Create Course Request Body:**
```json
{
  "title": "Advanced Mathematics",
  "subjectId": "uuid",
  "deliveryType": "OFFLINE",
  "locationName": "Room 101",
  "locationLat": 30.0123,
  "locationLng": 31.2345,
  "geofenceRadiusM": 50,
  "totalLessons": 24,
  "attendanceWeight": 0.7
}
```

### 2. Lesson Lifecycle
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/lessons` | Schedule a new lesson |
| `POST` | `/lessons/:id/start` | Set status to LIVE & start QR rotation |
| `POST` | `/lessons/:id/end` | Set status to COMPLETED |
| `POST` | `/lessons/:id/cancel` | Cancel lesson (notifies students) |
| `POST` | `/lessons/:id/reschedule` | Change time (notifies students) |

### 3. Attendance & QR
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/lessons/:id/qr` | Get the current active signed QR token |
| `POST` | `/attendance/scan` | Submit a scan (Requires Lat/Lng & Device ID) |
| `GET` | `/attendance/lesson/:id` | Get all attendance records for a lesson |
| `GET` | `/attendance/student/:id` | Get student attendance history |

**Scan Request Body:**
```json
{
  "qrPayload": "signed_token_payload",
  "qrSignature": "hmac_signature",
  "deviceId": "unique_device_id",
  "latitude": 30.0123,
  "longitude": 31.2345
}
```

### 4. Absence Requests
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/absences` | Student/Parent requests an excuse |
| `POST` | `/absences/:id/respond` | Admin/Teacher approves or rejects |
| `GET` | `/absences/pending-parent` | Parents see requests for their children |

### 5. Progress & Insights
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/progress/student/:courseId/:studentId` | Get detailed progress metrics |
| `POST` | `/progress/recompute/:courseId/:studentId` | Force refresh of progress stats |

---

## 📡 Event Payloads (Kafka)
The service emits events to the `courses.notification.requested.v1` topic with structured envelopes:

```json
{
  "event_id": "uuid",
  "event_type": "courses.lesson.started.v1",
  "payload": {
    "course_id": "uuid",
    "lesson_id": "uuid",
    "title": "Advanced Algorthims"
  }
}
```
