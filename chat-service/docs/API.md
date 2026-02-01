# Chat Service API Documentation

Base URL: `http://localhost:6004/api/v1`

---

## Authentication

All endpoints (except health checks) require JWT authentication.

**Header:**
```
Authorization: Bearer <access_token>
```

The JWT token must contain:
- `sub`: User ID (UUID)
- `role`: User role (STUDENT, INSTRUCTOR, TEACHER, PARENT, ASSISTANT)

---

## Conversations

### Create Group Chat

Creates a new group conversation. Only Instructors, Teachers, and Assistants can create groups.

```
POST /conversations
```

**Request Body:**
```json
{
  "name": "Study Group",
  "description": "Optional description",
  "member_ids": ["user-id-1", "user-id-2"]
}
```

**Response:** `201 Created`
```json
{
  "id": "conv-uuid",
  "type": "GROUP",
  "name": "Study Group",
  "description": "Optional description",
  "created_by": "creator-uuid",
  "created_at": "2026-02-01T06:00:00Z",
  "members": [
    {
      "id": "member-uuid",
      "user_id": "user-uuid",
      "user_role": "TEACHER",
      "member_role": "OWNER",
      "joined_at": "2026-02-01T06:00:00Z"
    }
  ]
}
```

---

### Create Direct Chat

Creates or retrieves a direct (1-to-1) chat.

```
POST /conversations/direct
```

**Request Body:**
```json
{
  "recipient_id": "user-uuid"
}
```

**Response:** `201 Created`
```json
{
  "id": "conv-uuid",
  "type": "DIRECT",
  "created_by": "sender-uuid",
  "members": [...]
}
```

---

### Get User Conversations

Lists all conversations for the authenticated user.

```
GET /conversations?limit=20&offset=0
```

**Response:** `200 OK`
```json
{
  "conversations": [
    {
      "id": "conv-uuid",
      "type": "GROUP",
      "name": "Study Group",
      "updated_at": "2026-02-01T06:00:00Z"
    }
  ]
}
```

---

### Get Conversation

Retrieves a specific conversation with members.

```
GET /conversations/:id
```

**Response:** `200 OK`
```json
{
  "id": "conv-uuid",
  "type": "GROUP",
  "name": "Study Group",
  "members": [...]
}
```

---

### Add Member

Adds a user to a group conversation. Requires Owner or Admin role.

```
POST /conversations/:id/members
```

**Request Body:**
```json
{
  "user_id": "new-user-uuid",
  "member_role": "MEMBER"
}
```

**Response:** `201 Created`
```json
{
  "message": "Member added successfully"
}
```

---

### Remove Member

Removes a user from a conversation. Requires Owner or Admin role.

```
DELETE /conversations/:id/members/:memberId
```

**Response:** `200 OK`
```json
{
  "message": "Member removed successfully"
}
```

---

### Update Member Role

Updates a member's role. Owner only.

```
PATCH /conversations/:id/members/:memberId/role
```

**Request Body:**
```json
{
  "member_role": "ADMIN"
}
```

**Response:** `200 OK`

---

## Messages

### Send Message

Sends a message to a conversation.

```
POST /conversations/:id/messages
```

**Request Body (Text):**
```json
{
  "type": "text",
  "content": "Hello everyone!"
}
```

**Request Body (Image):**
```json
{
  "type": "image",
  "media_url": "https://res.cloudinary.com/.../image.jpg",
  "content": "Check this out"
}
```

**Request Body (Voice):**
```json
{
  "type": "voice",
  "media_url": "https://res.cloudinary.com/.../audio.mp3",
  "media_metadata": {
    "duration": 45
  }
}
```

**Request Body (Reply):**
```json
{
  "type": "text",
  "content": "I agree!",
  "reply_to_id": "original-message-uuid"
}
```

**Response:** `201 Created`
```json
{
  "id": "msg-uuid",
  "conversation_id": "conv-uuid",
  "sender_id": "user-uuid",
  "sender_role": "TEACHER",
  "type": "text",
  "content": "Hello everyone!",
  "created_at": "2026-02-01T06:00:00Z"
}
```

---

### Get Messages

Retrieves messages for a conversation with pagination.

```
GET /conversations/:id/messages?limit=50&offset=0
```

**Response:** `200 OK`
```json
{
  "messages": [
    {
      "id": "msg-uuid",
      "content": "Hello",
      "sender_id": "user-uuid",
      "created_at": "2026-02-01T06:00:00Z",
      "reply_to": null
    }
  ]
}
```

---

### Long Poll Messages

Polls for new messages. Holds connection up to 30 seconds.

```
GET /conversations/:id/poll?after=last-message-uuid
```

**Response:** `200 OK` (if new messages)
```json
{
  "messages": [...],
  "has_more": false
}
```

**Response:** `204 No Content` (if no new messages after timeout)

---

### Delete Message

Soft-deletes a message. Only sender or moderators can delete.

```
DELETE /conversations/:id/messages/:messageId
```

**Response:** `200 OK`

---

### Pin Message

Pins a message. Requires Owner, Admin, or moderator role.

```
POST /conversations/:id/messages/:messageId/pin
```

**Response:** `200 OK`

---

### Unpin Message

Unpins a message.

```
DELETE /conversations/:id/messages/:messageId/pin
```

**Response:** `200 OK`

---

### Get Pinned Messages

Returns all pinned messages in a conversation.

```
GET /conversations/:id/pinned
```

**Response:** `200 OK`
```json
{
  "pinned_messages": [
    {
      "id": "pin-uuid",
      "message_id": "msg-uuid",
      "pinned_by": "user-uuid",
      "pinned_at": "2026-02-01T06:00:00Z",
      "message": {...}
    }
  ]
}
```

---

## Typing Indicators

### Set Typing

Indicates the user is typing. Auto-expires after 3 seconds.

```
POST /typing
```

**Request Body:**
```json
{
  "conversation_id": "conv-uuid"
}
```

**Response:** `204 No Content`

---

### Get Typing Users

Returns users currently typing in a conversation.

```
GET /typing?conversation_id=conv-uuid
```

**Response:** `200 OK`
```json
{
  "typing_users": [
    {
      "user_id": "user-uuid",
      "user_role": "TEACHER"
    }
  ]
}
```

---

## Media Upload

### Get Presigned URL

Generates a Cloudinary upload URL for media.

```
POST /media/presign
```

**Request Body:**
```json
{
  "type": "image",
  "content_type": "image/jpeg",
  "file_size": 1048576
}
```

**Limits:**
- Image: max 5 MB
- Voice: max 10 MB

**Response:** `200 OK`
```json
{
  "upload_url": "https://api.cloudinary.com/v1_1/cloud/image/upload",
  "signature": "abc123...",
  "timestamp": 1706767200,
  "api_key": "your-api-key",
  "public_id": "chat/2026/02/1706767200",
  "download_url": "https://res.cloudinary.com/.../chat/2026/02/1706767200"
}
```

---

## Health Checks

### Health

```
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "ok",
  "service": "chat-service"
}
```

### Ready

```
GET /ready
```

**Response:** `200 OK`
```json
{
  "status": "ready"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `BAD_REQUEST` | 400 | Invalid request body |
| `UNAUTHORIZED` | 401 | Missing/invalid token |
| `FORBIDDEN` | 403 | Permission denied |
| `NOT_FOUND` | 404 | Resource not found |
| `PAYLOAD_TOO_LARGE` | 413 | File size exceeded |
| `INTERNAL_ERROR` | 500 | Server error |
