# Postman Testing Guide - Auth Service

This guide explains how to test the Notifications, Parent Link, and Onboarding endpoints using the provided Postman collection.

## Setup

### 1. Import Collection
1. Open Postman
2. Click **Import** button
3. Select `postman_collection.json`
4. The collection will be imported with all endpoints organized in folders

### 2. Configure Environment Variables
The collection uses these variables (already configured):
- `base_url`: `http://localhost:6001` (default)
- `user_id`: Automatically set after registration/login
- `parent_id`: Set after registering a parent user
- `request_id`: Set after sending a parent link request

### 3. Enable Cookie Handling
Postman automatically handles cookies when:
- You use the same Postman instance for all requests
- Cookies are set by the server (httpOnly cookies are supported)
- The `base_url` matches the cookie domain

**Note**: Make sure your server is running on `http://localhost:6001`

## Testing Flow

### Step 1: Authentication

#### 1.1 Register a Child User (Student)
- **Endpoint**: `POST /api/v1/auth/register`
- **Body**:
```json
{
    "name": "John Doe",
    "username": "johndoe",
    "email": "john@example.com",
    "password": "password123"
}
```
- **Expected Response**: 201 Created
- **Notes**: 
  - Tokens are set in cookies automatically
  - OTP is returned in non-production environments
  - User ID is saved to `user_id` variable

#### 1.2 Register a Parent User
- **Endpoint**: `POST /api/v1/auth/register` (Parent User request)
- **Body**:
```json
{
    "name": "Parent User",
    "username": "parentuser",
    "email": "parent@example.com",
    "password": "password123"
}
```
- **Expected Response**: 201 Created
- **Notes**: Parent ID is saved to `parent_id` variable

#### 1.3 Login (if needed)
- **Endpoint**: `POST /api/v1/auth/login`
- **Body**:
```json
{
    "emailOrUsername": "john@example.com",
    "password": "password123"
}
```
- **Expected Response**: 200 OK
- **Notes**: Tokens are set in cookies automatically

#### 1.4 Refresh Token
- **Endpoint**: `POST /api/v1/auth/refresh`
- **Body**: None (refresh token is read from cookies)
- **Expected Response**: 200 OK
- **Response Includes**:
  - User information
  - New access token (set in cookies)
  - New refresh token (set in cookies, old one is revoked)
- **Notes**: 
  - No authentication required (uses refresh token)
  - Automatically rotates refresh token for security
  - Use this when access token expires (typically after 15 minutes)
  - Refresh token is read from cookies automatically

#### 1.5 Logout
- **Endpoint**: `POST /api/v1/auth/logout`
- **Body**: None
- **Expected Response**: 200 OK
- **Notes**: 
  - Revokes refresh token
  - Clears authentication cookies

#### 1.6 Resend Verification OTP
- **Endpoint**: `POST /api/v1/auth/resend-verification-otp`
- **Body**:
```json
{
    "email": "user@example.com"
}
```
- **Expected Response**: 200 OK
- **Response Includes**:
  - Success message
  - OTP (in non-production environments for testing)
- **Notes**: 
  - Use this if user didn't receive OTP during registration or OTP expired
  - Rate limited: 1 minute cooldown between requests, max 5 requests per hour
  - If email is already verified, returns 400 error
  - If email doesn't exist, returns generic message (prevents email enumeration)
  - OTP is exposed in non-production environments for testing

#### 1.7 Verify Email OTP
- **Endpoint**: `POST /api/v1/auth/verify-email-otp`
- **Body**:
```json
{
    "email": "user@example.com",
    "otp": "123456"
}
```
- **Expected Response**: 200 OK
- **Response Includes**:
  - Success message
  - User information with verified status
- **Notes**: 
  - Verifies user's email address
  - Clears verification cooldowns on success
  - Rate limited to prevent brute force attacks

### Step 2: Onboarding

#### 2.1 Complete Onboarding for Child (Student)
- **Endpoint**: `POST /api/v1/onboarding`
- **Authentication**: Required (cookies automatically sent)
- **Body Options**:

**Minimal**:
```json
{
    "role": "STUDENT",
    "dateOfBirth": "2010-01-15",
    "gender": "MALE",
    "country": "USA"
}
```

**Full** (with preferences, interests, and parent linking):
```json
{
    "role": "STUDENT",
    "dateOfBirth": "2010-01-15",
    "gender": "MALE",
    "country": "USA",
    "profileImg": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
    "preferences": {
        "language": "en",
        "themePreference": "dark",
        "notifications": true
    },
    "interests": ["Programming", "Mathematics", "Science"],
    "parentIds": ["{{parent_id}}"]
}
```

- **Expected Response**: 200 OK
- **Response Includes**:
  - User data with all updated fields
  - Preferences
  - Interests
  - Parent link requests (if parentIds provided)

#### 2.2 Complete Onboarding for Parent
- **Endpoint**: `POST /api/v1/onboarding`
- **Authentication**: Required (login as parent user first)
- **Body**:
```json
{
    "role": "PARENT",
    "dateOfBirth": "1985-05-20",
    "gender": "FEMALE",
    "country": "USA",
    "profileImg": "data:image/png;base64,...",
    "preferences": {
        "language": "en",
        "themePreference": "light",
        "notifications": true
    },
    "interests": ["Education", "Parenting"]
}
```

- **Expected Response**: 200 OK
- **Notes**: ParentIds are not needed for parents

### Step 3: Parent Link Requests

#### 3.1 Search for Parents
- **Endpoint**: `GET /api/v1/parent-link/search?query=parent&page=1&limit=10`
- **Authentication**: Required
- **Query Parameters**:
  - `query`: Search term (username, email, or name)
  - `page`: Page number (default: 1)
  - `limit`: Results per page (default: 10, max: 50)
- **Expected Response**: 200 OK
- **Response Includes**:
  - Array of parent users matching the search
  - Pagination information

#### 3.2 Send Parent Link Request
- **Endpoint**: `POST /api/v1/parent-link/request`
- **Authentication**: Required (must be logged in as child)
- **Body**:
```json
{
    "parentId": "{{parent_id}}"
}
```
- **Expected Response**: 201 Created
- **Response Includes**:
  - Request ID
  - Parent information
  - Request status (PENDING)
  - Created timestamp
- **Notes**: 
  - Triggers a notification to the parent
  - Request ID is saved to `request_id` variable

#### 3.3 Get Pending Requests (Child View)
- **Endpoint**: `GET /api/v1/parent-link/requests`
- **Authentication**: Required (logged in as child)
- **Expected Response**: 200 OK
- **Response Includes**:
  - Array of pending requests sent by the child
  - Parent information for each request
  - Request status and timestamps

#### 3.4 Get Pending Requests (Parent View)
- **Endpoint**: `GET /api/v1/parent-link/requests`
- **Authentication**: Required (logged in as parent)
- **Expected Response**: 200 OK
- **Response Includes**:
  - Array of pending requests received by the parent
  - Child information for each request
  - Request status and timestamps

#### 3.5 Respond to Request (Accept)
- **Endpoint**: `POST /api/v1/parent-link/respond`
- **Authentication**: Required (must be logged in as parent)
- **Body**:
```json
{
    "requestId": "{{request_id}}",
    "action": "accept"
}
```
- **Expected Response**: 200 OK
- **Response Includes**:
  - Updated request status (ACCEPTED)
  - Responded timestamp
- **Notes**: 
  - Creates a parent-child link
  - Sends notification to child
  - Only parents can respond

#### 3.6 Respond to Request (Decline)
- **Endpoint**: `POST /api/v1/parent-link/respond`
- **Authentication**: Required (must be logged in as parent)
- **Body**:
```json
{
    "requestId": "{{request_id}}",
    "action": "decline"
}
```
- **Expected Response**: 200 OK
- **Response Includes**:
  - Updated request status (DECLINED)
  - Responded timestamp
- **Notes**: 
  - Sends notification to child
  - Does not create a link

#### 3.7 Get Linked Accounts
- **Endpoint**: `GET /api/v1/parent-link/linked`
- **Authentication**: Required
- **Expected Response**: 200 OK
- **Response Includes**:
  - For children: Array of linked parents
  - For parents: Array of linked children
  - Link creation timestamps

### Step 4: Notifications

#### 4.1 Register FCM Token
- **Endpoint**: `POST /api/v1/notifications/register-token`
- **Authentication**: Required (Authorization header)
- **Body**:
```json
{
    "token": "FCM_DEVICE_TOKEN",
    "platform": "ios",
    "deviceId": "optional-device-id"
}
```
- **Expected Response**: 200 OK
- **Notes**: 
  - Mobile clients should register FCM token after login
  - Token is used to send push notifications to the device
  - Platform should be "ios" or "android"

#### 4.2 Get Notifications (History)
- **Endpoint**: `GET /api/v1/notifications`
- **Authentication**: Required (Authorization header)
- **Query Parameters**:
  - `page` (optional, default: 1)
  - `limit` (optional, default: 10, max: 50)
  - `unreadOnly` (optional, default: false)
- **Expected Response**: 200 OK with paginated notification list
- **Notes**: 
  - Returns notification history from database
  - Real-time notifications are delivered via FCM push notifications

## Complete Test Scenarios

### Scenario 1: Full Parent-Child Linking Flow

1. **Register Parent User**
   - Register a parent user
   - Complete onboarding for parent (set role to PARENT)

2. **Register Child User**
   - Register a child user
   - Complete onboarding for child (set role to STUDENT, include parentIds)

3. **Parent Opens Notification Stream**
   - Connect to `/api/v1/notifications/stream` as parent
   - Keep connection open to receive real-time notifications

4. **Child Sends Link Request**
   - Login as child
   - Search for parents (optional)
   - Send parent link request

5. **Parent Receives Notification**
   - Parent should receive notification in SSE stream
   - Notification type: `parent_link_request_received`

6. **Parent Views Pending Requests**
   - Get pending requests as parent
   - Should see the request from child

7. **Parent Responds to Request**
   - Accept or decline the request
   - Child receives notification via SSE stream

8. **Verify Link**
   - Get linked accounts as child (should see parent)
   - Get linked accounts as parent (should see child)

### Scenario 2: Onboarding with Parent Linking

1. **Register Parent**
   - Register and complete onboarding for parent

2. **Register Child**
   - Register child user

3. **Complete Onboarding with Parent IDs**
   - Complete onboarding including `parentIds` array
   - Parent link requests are automatically sent
   - Notifications are skipped during onboarding (to avoid spam)

4. **Parent Views Requests**
   - Parent gets pending requests
   - Should see requests from onboarding

5. **Parent Accepts Requests**
   - Parent responds to requests
   - Links are created

## Expected Response Formats

### Parent Link Request Response
```json
{
    "message": "Parent link request sent successfully",
    "request": {
        "id": "request-id",
        "parent": {
            "id": "parent-id",
            "username": "parentuser",
            "name": "Parent User",
            "profileImg": null
        },
        "status": "PENDING",
        "createdAt": "2024-01-15T10:30:00.000Z"
    }
}
```

### Pending Requests Response (Parent View)
```json
{
    "data": [
        {
            "id": "request-id",
            "status": "PENDING",
            "createdAt": "2024-01-15T10:30:00.000Z",
            "child": {
                "id": "child-id",
                "username": "johndoe",
                "name": "John Doe",
                "email": "john@example.com",
                "profileImg": null
            }
        }
    ]
}
```

### Pending Requests Response (Child View)
```json
{
    "data": [
        {
            "id": "request-id",
            "status": "PENDING",
            "createdAt": "2024-01-15T10:30:00.000Z",
            "parent": {
                "id": "parent-id",
                "username": "parentuser",
                "name": "Parent User",
                "email": "parent@example.com",
                "profileImg": null
            }
        }
    ]
}
```

### Notification Format (SSE)
```
data: {"type":"parent_link_request_received","requestId":"...","child":{"id":"...","username":"...","name":"..."},"status":"PENDING","createdAt":"..."}

data: {"type":"parent_link_request_accepted","requestId":"...","parent":{"id":"...","username":"...","name":"..."},"status":"ACCEPTED","respondedAt":"..."}
```

### Onboarding Response
```json
{
    "message": "Onboarding completed successfully",
    "user": {
        "id": "user-id",
        "name": "John Doe",
        "username": "johndoe",
        "email": "john@example.com",
        "dateOfBirth": "2010-01-15T00:00:00.000Z",
        "gender": "MALE",
        "country": "USA",
        "role": "STUDENT",
        "profileImg": "https://...",
        "onboardingCompleted": true,
        "preferences": {
            "id": "pref-id",
            "userId": "user-id",
            "language": "en",
            "themePreference": "dark",
            "notifications": true
        },
        "interests": [
            {"id": "interest-id", "name": "Programming"},
            {"id": "interest-id", "name": "Mathematics"}
        ],
        "parentLinkRequests": [
            {
                "id": "request-id",
                "parentId": "parent-id",
                "status": "PENDING",
                "createdAt": "2024-01-15T10:30:00.000Z"
            }
        ]
    }
}
```

## Troubleshooting

### Cookies Not Working
- Ensure you're using the same Postman instance for all requests
- Check that `base_url` matches your server URL
- Verify cookies are being set in the response headers
- In Postman, check **Cookies** in the response section

### Authentication Errors
- Make sure you've registered and logged in first
- Verify cookies are being sent (check request headers)
- If you get "Invalid or expired token", use the refresh token endpoint to get a new access token
- Try logging in again if refresh token is also expired
- Access tokens expire after 15 minutes - refresh them regularly

### Notification Stream Not Receiving Messages
- Ensure Redis is running and configured
- Check that the user is subscribed to the correct channel
- Verify notifications are being published when events occur
- Check server logs for errors

### Parent Link Request Fails
- Verify parent user exists and has PARENT role
- Check that child user has completed onboarding
- Ensure request doesn't already exist (duplicate requests are prevented)
- Check that parent ID is valid

## Role Enum Values

- `STUDENT`
- `TEACHER`
- `PARENT`
- `INSTRUCTOR`
- `ASSISTANT`
- `HR`
- `RECRUITER`

## Gender Enum Values

- `MALE`
- `FEMALE`
- `OTHER`
- `PREFER_NOT_TO_SAY`

## Request Status Values

- `PENDING`
- `ACCEPTED`
- `DECLINED`
- `CANCELLED`

## Notes

- All authenticated endpoints require cookies (set automatically after login)
- SSE notifications require an open connection
- Parent link requests can only be sent by children to parents
- Only parents can respond to parent link requests
- Onboarding can only be completed once per user
- Profile images should be base64 encoded data URLs
- Date of birth should be in ISO format (YYYY-MM-DD)

