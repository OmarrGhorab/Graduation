# Postman Testing Files

This directory contains all the files needed to test the Auth Service API endpoints for Notifications, Parent Link, and Onboarding.

## Files Overview

### 1. `postman_collection.json`
**Complete Postman collection** with all endpoints organized in folders:
- **Authentication**: Register, Login endpoints
- **Notifications**: SSE stream and polling endpoints
- **Parent Link**: Search, send request, get pending, respond, get linked accounts
- **Onboarding**: Complete onboarding with various scenarios

**How to use**:
1. Open Postman
2. Click **Import**
3. Select `postman_collection.json`
4. Start testing!

### 2. `postman_environment.json`
**Postman environment template** with pre-configured variables:
- `base_url`: API base URL (default: http://localhost:6001)
- `user_id`: Automatically set after registration/login
- `parent_id`: Set after registering a parent user
- `request_id`: Set after sending a parent link request
- And more...

**How to use**:
1. Open Postman
2. Click **Import**
3. Select `postman_environment.json`
4. Select the environment from the dropdown (top right)
5. Variables will be automatically updated as you make requests

### 3. `postman_examples.json`
**Quick reference** with all example request bodies in JSON format. Use this to:
- Copy request bodies quickly
- See all available options
- Reference enum values (roles, genders, statuses)
- View all endpoint URLs

### 4. `POSTMAN_TESTING_GUIDE.md`
**Comprehensive testing guide** with:
- Step-by-step setup instructions
- Complete test scenarios
- Expected response formats
- Troubleshooting tips
- All enum values and constants

## Quick Start

### 1. Import Files
```bash
# Import into Postman:
# 1. postman_collection.json (Collection)
# 2. postman_environment.json (Environment)
```

### 2. Set Environment
- Select "Auth Service - Local" environment in Postman (top right)
- Verify `base_url` is set to `http://localhost:6001`

### 3. Start Testing

#### Basic Flow:
1. **Register Parent User** → Get parent ID
2. **Register Child User** → Get child ID
3. **Complete Onboarding (Parent)** → Set role to PARENT
4. **Complete Onboarding (Child)** → Set role to STUDENT, include parentIds
5. **Send Parent Link Request** → (if not done in onboarding)
6. **Get Pending Requests (Parent)** → View requests
7. **Respond to Request (Parent)** → Accept/Decline
8. **Get Linked Accounts** → Verify link created

#### Notification Testing:
1. **Open Notification Stream (Parent)** → Keep connection open
2. **Send Parent Link Request (Child)** → Parent receives notification
3. **Respond to Request (Parent)** → Child receives notification

## Endpoints Summary

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/resend-verification-otp` - Resend verification OTP
- `POST /api/v1/auth/verify-email-otp` - Verify email with OTP

### Onboarding
- `POST /api/v1/onboarding` - Complete onboarding

### Parent Link
- `GET /api/v1/parent-link/search` - Search for parents
- `POST /api/v1/parent-link/request` - Send parent link request
- `GET /api/v1/parent-link/requests` - Get pending requests
- `POST /api/v1/parent-link/respond` - Respond to request (accept/decline)
- `GET /api/v1/parent-link/linked` - Get linked accounts

### Notifications
- `GET /api/v1/notifications/stream` - SSE stream for real-time notifications
- `GET /api/v1/notifications` - Polling endpoint (fallback)

## Important Notes

### Authentication
- All endpoints (except auth) require authentication
- Authentication is handled via **cookies** (httpOnly)
- Cookies are automatically set after login/register
- Postman handles cookies automatically (no manual token management needed)

### Testing Order
1. Always register/login first to get authentication cookies
2. Complete onboarding before testing parent link requests
3. Register parent user before child user (to get parent ID)
4. Keep notification stream open while testing to see real-time updates

### Variables
- Variables are automatically set by test scripts in the collection
- Check environment variables in Postman to see current values
- Use `{{variable_name}}` syntax in request bodies to use variables

### Cookies
- Cookies are automatically handled by Postman
- Make sure you're using the same Postman instance for all requests
- Check cookies in the response to verify they're being set
- Cookies are httpOnly, so they're only accessible via the browser/Postman

## Common Issues

### Cookies Not Working
- Ensure you're using the same Postman instance
- Check that base_url matches your server
- Verify cookies are being set in response headers
- Try logging in again to refresh tokens

### Authentication Errors
- Make sure you've registered and logged in first
- Verify cookies are being sent (check request headers)
- Try logging in again to refresh tokens

### Notification Stream Not Working
- Ensure Redis is running and configured
- Check server logs for errors
- Verify user is authenticated
- Check that notifications are being published

### Parent Link Request Fails
- Verify parent user exists and has PARENT role
- Check that child user has completed onboarding
- Ensure request doesn't already exist
- Verify parent ID is valid

## Support

For detailed testing instructions, see `POSTMAN_TESTING_GUIDE.md`.

For example request bodies, see `postman_examples.json`.

For endpoint documentation, check the API code in `auth-service/src/controllers/`.

