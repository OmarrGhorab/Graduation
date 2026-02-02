# Chat Service Documentation

## How "Read Message" Logic Works

The "Read" status is tracked **per user, per conversation** utilizing the `conversation_members` table. It is **NOT** tracked on the individual messages themselves.

### 1. Sending a Message (Incrementing Unread)
When a message is sent:
1. The message is saved to the `messages` table.
2. The system calls `MemberRepository.IncrementUnreadCount`.
3. This finds all **active members** of the conversation (everyone who hasn't left).
4. It increments the `unread_count` by 1 for **everyone except the sender**.

### 2. Reading a Message (Resetting Unread)
When a user opens a chat or scrolls to the bottom, the frontend calls the **Mark as Read** endpoint (`POST /api/v1/conversations/:id/read`).
1. The backend verifies the user is a member.
2. It calls `MemberRepository.ResetUnreadCount`.
3. This updates the `conversation_members` table for that specific user:
   - Sets `unread_count` to `0`.
   - Updates `last_read_at` to the current timestamp (`NOW()`).
   - Optionally updates `last_read_message_id`.

### Direct vs Group Chat
The logic is **identical** for both:
- **Direct Chat**: User A sends -> User B's count +1. User B reads -> User B's count = 0.
- **Group Chat**: User A sends -> User B, C, D counts +1. User B reads -> Only User B's count = 0. C and D remain unread until they view it.

---

## API Endpoints

### Conversations
- **GET /api/v1/conversations**
  - List all conversations for user.
  - Supports filtering by `role`, `type`, and search `q`.
- **GET /api/v1/conversations/:id**
  - Get single conversation details.
### Group Creation
- **Endpoint**: `POST /api/v1/conversations`
- **Permissions**: Only Global **TEACHER** or **INSTRUCTOR** can create groups.
- **Request Body**:
  ```json
  {
    "name": "Group Name",
    "description": "Optional description",
    "member_ids": ["uuid-1", "uuid-2"]
  }
  ```
- **POST /api/v1/conversations/direct**
  - Create/Get direct chat. `(body: recipient_id)`
- **POST /api/v1/conversations/:id/read**
  - Mark conversation as read.

### Members
- **GET /api/v1/conversations/:id/members**
  - List all members. (Enriched with names/avatars).
- **POST /api/v1/conversations/:id/members**
  - Add member to group.
- **DELETE /api/v1/conversations/:id/members/:memberId**
  ### 8. Remove Member (Leave / Kick)
Removes a member from a conversation. This single endpoint handles both "Leaving" and "Kicking" depending on who is calling it.

- **URL**: `DELETE /api/v1/conversations/:id/members/:memberId`

#### Case A: Leave Group
If a user calls this endpoint with **their own `memberId`**:
- **Action**: The user leaves the group.
- **Permission**: Any member can leave a group.

#### Case B: Kick User
If a user calls this endpoint with **someone else's `memberId`**:
- **Action**: The target user is removed (kicked) from the group.
- **Permission**:
  - **Group Owner**: Can kick anyone.
  - **Group Admin**: Can kick members.
  - **Note**: Global roles (Teacher, Assistant, etc.) **cannot** kick users unless they have been explicitly assigned the `ADMIN` or `OWNER` role within that specific group.
