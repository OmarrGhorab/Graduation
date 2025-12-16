# Design Document: User Profile Endpoint

## Overview

This feature adds a GET `/myprofile` endpoint to the authentication service that returns the authenticated user's complete profile information. The endpoint leverages existing authentication middleware and follows established patterns in the codebase for consistency and maintainability.

## Architecture

The endpoint follows the existing MVC pattern used throughout the auth service:

- **Route Layer**: Defines the endpoint and applies authentication middleware
- **Controller Layer**: Handles request processing and response formatting
- **Data Layer**: Uses Prisma ORM to query user data from the database

The endpoint integrates seamlessly with the existing authentication flow, requiring a valid access token to access user profile data.

## Components and Interfaces

### Route Definition

```typescript
// In auth-service/src/routes/auth.route.ts
router.get("/myprofile", authenticate, getMyProfile);
```

The route uses the existing `authenticate` middleware to verify the access token and attach user information to the request object.

### Controller Function

```typescript
// In auth-service/src/controllers/auth.core.controller.ts
export const getMyProfile = async (req: Request, res: Response, next: NextFunction) => {
  // Implementation details in tasks
}
```

The controller function:
1. Extracts user ID from `req.user` (populated by authenticate middleware)
2. Queries the database for complete user profile
3. Returns sanitized user data (excluding sensitive fields)
4. Handles error cases (user not found)

### Response Interface

```typescript
interface MyProfileResponse {
  user: {
    id: string;
    name: string;
    username: string;
    email: string;
    verified: boolean;
    onboardingCompleted: boolean;
    role: string;
    profileImg: string | null;
    isActive: boolean;
    lastLoginAt: Date | null;
    createdAt: Date;
    updatedAt: Date;
  }
}
```

## Data Models

The endpoint uses the existing User model from Prisma schema. Key fields returned:

- **Identity**: id, name, username, email
- **Status**: verified, onboardingCompleted, isActive
- **Authorization**: role
- **Profile**: profileImg
- **Timestamps**: lastLoginAt, createdAt, updatedAt

Fields explicitly excluded for security:
- password (sensitive)
- deletedAt (internal)
- deviceBlocked (internal)
- pendingDeviceFingerprint (internal)
- twoFactorSecret (sensitive)
- twoFactorBackupCodes (sensitive)

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Authenticated access only
*For any* request to /myprofile without a valid access token, the system should return a 401 Unauthorized error
**Validates: Requirements 1.5**

### Property 2: User data retrieval
*For any* authenticated request with a valid user ID, the system should return the complete user profile matching that user ID
**Validates: Requirements 1.1, 1.2, 1.3**

### Property 3: Sensitive data exclusion
*For any* user profile response, the returned data should not contain password, deletedAt, deviceBlocked, pendingDeviceFingerprint, twoFactorSecret, or twoFactorBackupCodes fields
**Validates: Requirements 2.1, 2.2, 2.3**

### Property 4: Non-existent user handling
*For any* authenticated request where the user ID does not exist in the database, the system should return a 404 Not Found error
**Validates: Requirements 1.4**

## Error Handling

The endpoint follows existing error handling patterns:

1. **401 Unauthorized**: Missing or invalid access token (handled by authenticate middleware)
2. **404 Not Found**: User record not found in database
3. **500 Internal Server Error**: Database or unexpected errors (handled by global error handler)

All errors are passed to the `next()` function for centralized error handling.

## Testing Strategy

### Unit Tests

Unit tests will verify:
- Successful profile retrieval with valid authentication
- 404 error when user doesn't exist
- Proper field exclusion (no sensitive data in response)

### Property-Based Tests

Property-based tests will use **fast-check** (TypeScript property testing library) to verify:
- Property 1: Unauthenticated requests always fail
- Property 2: Valid user IDs always return matching profile data
- Property 3: Response never contains sensitive fields
- Property 4: Invalid user IDs always return 404

Each property test will run a minimum of 100 iterations to ensure comprehensive coverage across random inputs.

Property tests will be tagged with comments in this format:
```typescript
// Feature: user-profile-endpoint, Property 1: Authenticated access only
```

## Integration Points

The endpoint integrates with:

1. **Authentication Middleware**: Uses existing `authenticate` middleware for token verification
2. **Prisma Client**: Uses existing Prisma instance for database queries
3. **Error Handler**: Uses existing error handling middleware
4. **Express Router**: Adds route to existing auth router

No new dependencies or infrastructure changes required.
