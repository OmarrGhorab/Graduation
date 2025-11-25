# Sessions Endpoints Testing Guide

## Prerequisites

1. **Authenticated User** - You need to be logged in (have valid access token)
2. **Multiple Sessions** - Create multiple sessions by logging in from different devices/browsers
3. **Server Running** - Ensure your auth service is running on port 6001

## Available Endpoints

### 1. Get Activity
- **Endpoint:** `GET /api/v1/auth/activity`
- **Auth:** Required
- **Description:** Get user's last activity and current device info

### 2. Get All Sessions
- **Endpoint:** `GET /api/v1/auth/sessions`
- **Auth:** Required
- **Description:** List all user sessions (active and inactive)

### 3. Revoke Specific Session
- **Endpoint:** `DELETE /api/v1/auth/sessions/:sessionId`
- **Auth:** Required
- **Description:** Revoke a specific session by ID

### 4. Revoke All Sessions
- **Endpoint:** `DELETE /api/v1/auth/sessions/all`
- **Auth:** Required
- **Description:** Revoke all sessions except the current one

---

## Step-by-Step Testing Guide

### Step 1: Login to Create Sessions

**Create at least 2-3 sessions by logging in multiple times:**

```bash
# Login 1 (Browser/Postman)
POST {{BASE_URL}}/api/v1/auth/login
Content-Type: application/json

{
  "emailOrUsername": "your-email@example.com",
  "password": "your-password"
}

# Save the cookies (access_token and refresh_token)
```

**Login from different devices/browsers to create multiple sessions:**
- Browser 1 (Chrome)
- Browser 2 (Firefox) 
- Mobile app
- Postman (different user agent)

### Step 2: Get All Sessions

```bash
GET {{BASE_URL}}/api/v1/auth/sessions
Authorization: Bearer YOUR_ACCESS_TOKEN
# OR include cookies: access_token and refresh_token
```

**Expected Response:**
```json
{
  "sessions": [
    {
      "id": "session-id-1",
      "deviceName": "Chrome on Windows",
      "platform": "WEB",
      "ipAddress": "192.168.1.1",
      "location": null,
      "isActive": true,
      "isCurrent": true,  // This marks your current session
      "isRevoked": false,
      "isExpired": false,
      "lastActivityAt": "2025-01-15T10:30:00.000Z",
      "createdAt": "2025-01-15T08:00:00.000Z",
      "expiresAt": "2025-01-15T10:45:00.000Z",
      "revokedAt": null
    },
    {
      "id": "session-id-2",
      "deviceName": "Firefox on Windows",
      "platform": "WEB",
      "ipAddress": "192.168.1.1",
      "location": null,
      "isActive": true,
      "isCurrent": false,  // Other session
      "isRevoked": false,
      "isExpired": false,
      "lastActivityAt": "2025-01-15T09:00:00.000Z",
      "createdAt": "2025-01-15T08:30:00.000Z",
      "expiresAt": "2025-01-15T09:15:00.000Z",
      "revokedAt": null
    }
  ],
  "totalSessions": 2,
  "activeSessions": 2
}
```

**Note:** Copy one of the `session.id` values for testing revoke endpoint.

### Step 3: Get Activity

```bash
GET {{BASE_URL}}/api/v1/auth/activity
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Expected Response:**
```json
{
  "lastActivityAt": "2025-01-15T10:30:00.000Z",
  "currentDevice": {
    "deviceName": "Chrome on Windows",
    "platform": "WEB",
    "ipAddress": "192.168.1.1",
    "location": null
  },
  "totalActiveSessions": 2
}
```

### Step 4: Revoke a Specific Session (Other Session)

**Revoke a session that's NOT your current session:**

```bash
DELETE {{BASE_URL}}/api/v1/auth/sessions/session-id-2
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Expected Response:**
```json
{
  "message": "Session revoked successfully",
  "revoked": true,
  "loggedOut": false  // You stay logged in
}
```

**Verify:**
- Check sessions list again - the revoked session should show `isRevoked: true` and `isActive: false`
- You should still be able to make authenticated requests

### Step 5: Revoke Current Session

**Revoke your own current session:**

```bash
DELETE {{BASE_URL}}/api/v1/auth/sessions/session-id-1
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Expected Response:**
```json
{
  "message": "Session revoked successfully. You have been logged out.",
  "revoked": true,
  "loggedOut": true  // You've been logged out
}
```

**Verify:**
- Cookies should be cleared (check response headers)
- Try making another authenticated request - should get 401 Unauthorized
- Need to login again to continue

### Step 6: Revoke All Sessions

**First, login again and create multiple sessions, then:**

```bash
DELETE {{BASE_URL}}/api/v1/auth/sessions/all
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Expected Response:**
```json
{
  "message": "Revoked 2 session(s) successfully",
  "revokedCount": 2
}
```

**Verify:**
- Your current session remains active (you stay logged in)
- All other sessions are revoked
- Check sessions list - only current session should be active

---

## Testing with Postman

### Setup:

1. **Create Environment Variables:**
   - `BASE_URL`: `http://localhost:6001`
   - `ACCESS_TOKEN`: (will be set after login)
   - `SESSION_ID`: (will be set from sessions list)

2. **Login Request:**
   - Method: `POST`
   - URL: `{{BASE_URL}}/api/v1/auth/login`
   - Body (JSON):
     ```json
     {
       "emailOrUsername": "test@example.com",
       "password": "password123"
     }
     ```
   - **Tests Tab:** Add script to save cookies:
     ```javascript
     pm.environment.set("ACCESS_TOKEN", pm.cookies.get("access_token"));
     ```

3. **Get Sessions Request:**
   - Method: `GET`
   - URL: `{{BASE_URL}}/api/v1/auth/sessions`
   - Authorization: Bearer Token `{{ACCESS_TOKEN}}`
   - **Tests Tab:** Save a session ID:
     ```javascript
     const sessions = pm.response.json().sessions;
     if (sessions.length > 0) {
       pm.environment.set("SESSION_ID", sessions[0].id);
     }
     ```

4. **Revoke Session Request:**
   - Method: `DELETE`
   - URL: `{{BASE_URL}}/api/v1/auth/sessions/{{SESSION_ID}}`
   - Authorization: Bearer Token `{{ACCESS_TOKEN}}`

---

## Testing with cURL

### Get Sessions:
```bash
curl -X GET "http://localhost:6001/api/v1/auth/sessions" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Cookie: access_token=YOUR_ACCESS_TOKEN; refresh_token=YOUR_REFRESH_TOKEN"
```

### Revoke Specific Session:
```bash
curl -X DELETE "http://localhost:6001/api/v1/auth/sessions/SESSION_ID_HERE" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Cookie: access_token=YOUR_ACCESS_TOKEN; refresh_token=YOUR_REFRESH_TOKEN"
```

### Revoke All Sessions:
```bash
curl -X DELETE "http://localhost:6001/api/v1/auth/sessions/all" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Cookie: access_token=YOUR_ACCESS_TOKEN; refresh_token=YOUR_REFRESH_TOKEN"
```

---

## Test Scenarios

### ✅ Scenario 1: View All Sessions
1. Login from 2-3 different browsers/devices
2. Call `GET /sessions`
3. Verify all sessions appear with correct device info
4. Verify current session is marked with `isCurrent: true`

### ✅ Scenario 2: Revoke Other Session
1. Have multiple active sessions
2. Revoke a session that's NOT your current one
3. Verify you stay logged in
4. Verify the revoked session shows `isRevoked: true` in sessions list

### ✅ Scenario 3: Revoke Current Session
1. Get your current session ID from sessions list
2. Revoke your current session
3. Verify cookies are cleared
4. Verify you get logged out (`loggedOut: true`)
5. Try making another request - should get 401

### ✅ Scenario 4: Revoke All Sessions
1. Have multiple active sessions
2. Call `DELETE /sessions/all`
3. Verify only current session remains active
4. Verify you stay logged in
5. Verify other sessions are revoked

### ✅ Scenario 5: Activity Tracking
1. Make several authenticated requests
2. Call `GET /activity`
3. Verify `lastActivityAt` updates
4. Verify `totalActiveSessions` matches actual count

---

## Common Issues & Solutions

### Issue: "Cannot GET /api/v1/auth/sessions"
**Solution:** 
- Restart your server
- Verify route is registered in `auth.route.ts`
- Check server logs for errors

### Issue: "Authentication required"
**Solution:**
- Make sure you're logged in
- Include `Authorization: Bearer TOKEN` header
- Or include cookies with `access_token`

### Issue: "Session not found"
**Solution:**
- Verify the session ID is correct
- Make sure the session belongs to your user
- Check if session was already revoked

### Issue: No sessions showing up
**Solution:**
- Sessions are only created on login
- Make sure you've logged in after implementing sessions feature
- Check database - sessions should be in `Session` table

---

## Database Verification

You can also verify sessions directly in the database:

```sql
-- View all sessions for a user
SELECT 
  id,
  "sessionToken",
  "isActive",
  "isRevoked",
  "lastActivityAt",
  "createdAt",
  "expiresAt"
FROM "Session"
WHERE "userId" = 'USER_ID_HERE'
ORDER BY "lastActivityAt" DESC;
```

---

## Quick Test Checklist

- [ ] Login successfully
- [ ] Get sessions list (should show at least 1 session)
- [ ] Get activity info
- [ ] Login from another device/browser (create 2nd session)
- [ ] Get sessions list again (should show 2 sessions)
- [ ] Revoke other session (should stay logged in)
- [ ] Revoke current session (should get logged out)
- [ ] Login again
- [ ] Revoke all sessions (should stay logged in, others revoked)

