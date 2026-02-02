# Chat Service API Reference

This document provides detailed information about endpoints, request bodies, and expected responses for the Chat Service.

---

## 1. Conversations Management

### 1.1 Update Group Profile Image
Used to change the image of a group chat. Requires Owner or Admin (Assigned Assistant) role. Global roles without a group role cannot perform this.
- **Endpoint**: `PATCH /api/v1/conversations/:id/image`
- **Request Body**:
  ```json
  {
    "image_url": "https://cloudinary.com/path/to/image.png"
  }
  ```
- **Success Response (200 OK)**:
  ```json
  {
    "message": "Group image updated successfully"
  }
  ```

### 1.2 Get Pinned Messages
Retrieves all messages pinned within a specific conversation.
- **Endpoint**: `GET /api/v1/conversations/:id/messages/pinned`
- **Success Response (200 OK)**:
  ```json
  {
    "pinned_messages": [
      {
        "id": "uuid-pinned-id",
        "message_id": "uuid-message-id",
        "conversation_id": "uuid-conversation-id",
        "pinned_by": "uuid-user-id",
        "pinned_at": "2026-02-02T10:00:00Z",
        "message": {
          "id": "uuid-message-id",
          "content": "This is a pinned message",
          "sender_name": "Omar Ghorab",
          "sender_role": "INSTRUCTOR",
          "type": "text",
          "created_at": "2026-02-02T09:00:00Z"
        }
      }
    ]
  }
  ```

### 1.3 Get Media, Links, and Docs
Retrieves the history of images, voice messages, and files shared in a chat.
- **Endpoint**: `GET /api/v1/conversations/:id/messages/media`
- **Query Params**: `limit` (default 20), `offset` (default 0)
- **Success Response (200 OK)**:
  ```json
  {
    "messages": [
      {
        "id": "uuid-message-id",
        "type": "image",
        "media_urls": ["https://cloudinary.com/img1.png"],
        "sender_name": "Omar Ghorab",
        "created_at": "2026-02-02T11:00:00Z"
      },
      {
        "id": "uuid-voice-id",
        "type": "voice",
        "media_urls": ["https://cloudinary.com/audio.mp3"],
        "sender_name": "User X",
        "created_at": "2026-02-02T11:05:00Z"
      }
    ]
  }
  ```

### 1.4 Delete Entire Group
Permanently deletes the conversation, all messages, all members, and all pinned messages from the database.
- **Endpoint**: `DELETE /api/v1/conversations/:id`
- **Permission**: Requires **OWNER** or **ADMIN** role in the group.
- **Success Response (200 OK)**:
  ```json
  {
    "message": "Conversation deleted successfully"
  }
  ```

---

## 2. Member Management

### 2.1 Add Member
Adds a new user to a group conversation.
- **Endpoint**: `POST /api/v1/conversations/:id/members`
- **Request Body**:
  ```json
  {
    "user_id": "uuid-of-new-user",
    "member_role": "MEMBER" 
  }
  ```
- **Success Response (201 Created)**:
  ```json
  {
    "message": "Member added successfully"
  }
  ```

### 2.2 Assign Roles (Assistant / Admin)
Used to promote or demote a user within a group. "Assistant" in the UI corresponds to the `ADMIN` role.
- **Endpoint**: `PATCH /api/v1/conversations/:id/members/:memberId/role`
- **Request Body**:
  ```json
  {
    "member_role": "ADMIN"
  }
  ```
  *Options: `OWNER`, `ADMIN`, `MEMBER`*
- **Success Response (200 OK)**:
  ```json
  {
    "message": "Member role updated successfully"
  }
  ```

### 2.3 Leave / Kick Member
Removes a member from the group.
- **Endpoint**: `DELETE /api/v1/conversations/:id/members/:memberId`
- **Action**:
  - If calling on **yourself**: You leave the group.
  - If calling on **someone else**: You kick them (requires Owner or Admin role within the group).
- **Success Response (200 OK)**:
  ```json
  {
    "message": "Member removed successfully"
  }
  ```

---

## 3. Reference: Member Roles
| Role Name | Description |
| :--- | :--- |
| **OWNER** | Creator of the group. Full permissions. |
| **ADMIN** | Assigned Assistant. Can kick members, invite users, and change info. |
| **MEMBER** | Regular user. Can chat and leave. |
