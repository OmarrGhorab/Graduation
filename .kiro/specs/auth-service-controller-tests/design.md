# Design Document: Auth Service Controller Tests

## Overview

This design document outlines the architecture and implementation approach for comprehensive controller/component tests for the auth-service. The auth-service is a Node.js/Express application with 15 controllers handling authentication, authorization, session management, 2FA, OAuth, and related functionality.

### Testing Philosophy

Controller tests sit between unit tests and end-to-end tests. They verify:
- HTTP endpoint behavior (request/response handling)
- Middleware integration (authentication, rate limiting, error handling)
- Request validation and error responses
- Response format consistency
- Business logic integration at the API layer

Unlike unit tests that test individual functions in isolation, controller tests verify the complete request-response cycle through the Express application, including middleware execution and error handling.

### Key Differences from Unit Tests

The existing unit tests (`.kiro/specs/auth-service-unit-tests/`) focus on:
- Utils: Token generation, OTP management, 2FA utilities, session utilities
- Services: Auth session service, location service, parent link service
- Middleware: Authentication, rate limiting, error handling

This spec focuses on:
- Controllers: HTTP endpoint handlers that use the tested utils/services
- Integration: How controllers, middleware, and error handling work together
- API Contracts: Request/response formats, status codes, validation

### Testing Approach

We will use:
- **Vitest**: Already configured in the project
- **Supertest**: HTTP testing library for making requests to Express apps
- **Mock Strategy**: Mock external dependencies (Prisma, Redis, email, etc.) but test real middleware execution

## Architecture

### Test Structure

```
auth-service/tests/
├── controllers/
│   ├── auth.controller.test.ts          # Auth endpoints (register, login, logout, refresh)
│   ├── password.controller.test.ts      # Password management endpoints
│   ├── email-verification.controller.test.ts  # Email verification endpoints
│   ├── device.controller.test.ts        # Device verification endpoints
│   ├── oauth.controller.test.ts         # OAuth endpoints
│   ├── twoFactor.controller.test.ts     # 2FA endpoints
│   ├── account.controller.test.ts       # Account management endpoints
│   ├── sessions.controller.test.ts      # Session management endpoints
│   ├── activity.controller.test.ts      # Activity tracking endpoints
│   ├── profile.controller.test.ts       # Profile management endpoints
│   ├── onboarding.controller.test.ts    # Onboarding endpoints
│   ├── parent-link.controller.test.ts   # Parent link endpoints
│   ├── location.controller.test.ts      # Location tracking endpoints
│   └── internal.controller.test.ts      # Internal service endpoints
├── helpers/
│   ├── testApp.ts                       # Test Express app factory
│   ├── mocks.ts                         # Mock factories for Prisma, Redis, etc.
│   └── fixtures.ts                      # Test data fixtures
└── setup/
    └── vitest.setup.ts                  # Test setup and global mocks
```

### Test Application Factory

Each test file will create an Express application instance with mocked dependencies:

```typescript
// helpers/testApp.ts
import express, { Express } from 'express';
import { mockPrisma, mockRedis, mockResend } from './mocks';

export function createTestApp(): Express {
  const app = express();
  
  // Apply middleware
  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));
  
  // Apply routes
  app.use('/api/v1/auth', authRouter);
  // ... other routes
  
  // Apply error handler
  app.use(errorHandler);
  
  return app;
}
```

### Mock Strategy

We will mock external dependencies at the module level:

1. **Prisma Client**: Mock database operations
2. **Redis Client**: Mock cache operations
3. **Resend (Email)**: Mock email sending
4. **Cloudinary**: Mock image uploads
5. **Google OAuth**: Mock token verification
6. **Location Service**: Mock IP geolocation

Middleware will execute normally to test integration.

## Components and Interfaces

### Test Helpers

#### Test App Factory

```typescript
interface TestAppOptions {
  skipAuth?: boolean;
  skipRateLimit?: boolean;
  customMocks?: {
    prisma?: any;
    redis?: any;
    resend?: any;
  };
}

function createTestApp(options?: TestAppOptions): Express;
```

#### Mock Factories

```typescript
// Prisma mock factory
interface MockPrismaOptions {
  user?: Partial<User>;
  session?: Partial<Session>;
  // ... other models
}

function mockPrisma(options?: MockPrismaOptions): PrismaClient;

// Redis mock factory
interface MockRedisOptions {
  data?: Record<string, string>;
}

function mockRedis(options?: MockRedisOptions): Redis;

// Email mock factory
function mockResend(): Resend;
```

#### Test Fixtures

```typescript
// User fixtures
function createUserFixture(overrides?: Partial<User>): User;
function createVerifiedUserFixture(): User;
function createUnverifiedUserFixture(): User;
function createUser2FAFixture(): User;

// Token fixtures
function createValidAccessToken(userId: string): string;
function createExpiredAccessToken(userId: string): string;
function createInvalidAccessToken(): string;

// Session fixtures
function createSessionFixture(overrides?: Partial<Session>): Session;
```

### Controller Test Patterns

Each controller test file will follow this pattern:

```typescript
import request from 'supertest';
import { Express } from 'express';
import { createTestApp } from '../helpers/testApp';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import { createUserFixture, createValidAccessToken } from '../helpers/fixtures';

describe('Controller Name', () => {
  let app: Express;
  let prisma: any;
  let redis: any;

  beforeEach(() => {
    // Create fresh mocks
    prisma = mockPrisma();
    redis = mockRedis();
    
    // Create test app
    app = createTestApp({ customMocks: { prisma, redis } });
  });

  afterEach(() => {
    // Clear mocks
    vi.clearAllMocks();
  });

  describe('POST /api/v1/endpoint', () => {
    it('should return 200 with valid input', async () => {
      // Arrange
      const user = createUserFixture();
      prisma.user.findUnique.mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/endpoint')
        .send({ data: 'test' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('result');
    });

    it('should return 400 with invalid input', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/endpoint')
        .send({ invalid: 'data' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 401 without authentication', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/endpoint')
        .send({ data: 'test' });

      // Assert
      expect(response.status).toBe(401);
    });
  });
});
```

## Data Models

### Test Data Structures

#### User Test Data

```typescript
interface UserTestData {
  id: string;
  email: string;
  password: string;  // Plain text for testing
  hashedPassword: string;  // Bcrypt hash
  name: string;
  username: string;
  verified: boolean;
  onboardingCompleted: boolean;
  role: 'USER' | 'ADMIN';
  twoFactorEnabled: boolean;
  twoFactorSecret?: string;
  isActive: boolean;
  deletedAt?: Date;
}
```

#### Session Test Data

```typescript
interface SessionTestData {
  id: string;
  userId: string;
  sessionToken: string;
  refreshToken: string;
  ipAddress: string;
  userAgent: string;
  location?: string;
  isActive: boolean;
  isRevoked: boolean;
  expiresAt: Date;
  refreshExpiresAt: Date;
  lastActivityAt: Date;
}
```

#### Request Test Data

```typescript
interface AuthRequestData {
  email: string;
  password: string;
}

interface RegisterRequestData {
  email: string;
  password: string;
  name: string;
  username: string;
}

interface OTPRequestData {
  email: string;
  otp: string;
}
```

### Mock Response Structures

#### Success Response

```typescript
interface SuccessResponse<T = any> {
  message: string;
  data?: T;
  [key: string]: any;
}
```

#### Error Response

```typescript
interface ErrorResponse {
  error: string;
  statusCode: number;
  timestamp: string;
  details?: any;
}
```

## Testing Strategy

### Test Organization

Tests are organized by controller, with each controller having its own test file. Within each file, tests are grouped by endpoint using `describe` blocks.

### Test Coverage Goals

- **Controllers**: 70% line coverage minimum
- **Success Paths**: Test all successful request scenarios
- **Error Paths**: Test all error conditions (400, 401, 403, 404, 500)
- **Validation**: Test all input validation rules
- **Authentication**: Test protected endpoints with/without auth
- **Rate Limiting**: Test rate limit enforcement

### Test Execution

- Tests run in parallel using Vitest
- Each test file is independent
- Mocks are reset between tests
- No shared state between tests

### Mock Configuration

#### Prisma Mocks

```typescript
// Mock successful database operations
prisma.user.findUnique.mockResolvedValue(user);
prisma.user.create.mockResolvedValue(user);
prisma.user.update.mockResolvedValue(user);

// Mock database errors
prisma.user.findUnique.mockRejectedValue(new Error('Database error'));

// Mock not found
prisma.user.findUnique.mockResolvedValue(null);
```

#### Redis Mocks

```typescript
// Mock successful cache operations
redis.get.mockResolvedValue('value');
redis.set.mockResolvedValue('OK');
redis.del.mockResolvedValue(1);

// Mock cache miss
redis.get.mockResolvedValue(null);
```

#### Email Mocks

```typescript
// Mock successful email send
resend.emails.send.mockResolvedValue({ id: 'email-id' });

// Mock email error
resend.emails.send.mockRejectedValue(new Error('Email error'));
```

### Testing Patterns by Controller

#### Authentication Controller

Test patterns:
- Registration with valid/invalid data
- Login with correct/incorrect credentials
- Login with unverified email
- Login with 2FA enabled
- Logout with valid/invalid token
- Token refresh with valid/expired token

#### Password Controller

Test patterns:
- Forgot password with valid/invalid email
- Verify reset OTP with valid/invalid code
- Reset password with valid/expired token
- Password strength validation

#### Email Verification Controller

Test patterns:
- Verify email with valid/invalid OTP
- Resend OTP with cooldown enforcement
- Verify already verified email

#### Device Controller

Test patterns:
- Verify device with valid/invalid OTP
- Resend device OTP
- Trust device after verification

#### OAuth Controller

Test patterns:
- Google auth with valid/invalid token
- New user creation via OAuth
- Existing user login via OAuth

#### Two-Factor Controller

Test patterns:
- Enable 2FA and generate QR code
- Verify 2FA setup with TOTP
- Disable 2FA with password
- Verify 2FA login with TOTP/backup code
- Regenerate backup codes

#### Account Controller

Test patterns:
- Deactivate account
- Delete account with password
- Reactivate account
- Delete profile image

#### Sessions Controller

Test patterns:
- List all sessions
- Get session details
- Revoke specific session
- Revoke all sessions
- Cleanup expired sessions

#### Activity Controller

Test patterns:
- Get activity log
- Pagination

#### Profile Controller

Test patterns:
- Get profile
- Update profile
- Upload profile image

#### Onboarding Controller

Test patterns:
- Complete onboarding
- Get onboarding status

#### Parent Link Controller

Test patterns:
- Create parent link
- Verify parent link
- Accept parent link
- Reject parent link

#### Location Controller

Test patterns:
- Update location
- Get location history

#### Internal Controller

Test patterns:
- Validate token
- Get user by ID

### Error Handling Tests

All controllers should test:
- 400 Bad Request: Invalid input, validation errors
- 401 Unauthorized: Missing/invalid/expired token
- 403 Forbidden: Insufficient permissions
- 404 Not Found: Resource not found
- 500 Internal Server Error: Unexpected errors

Error response format:
```json
{
  "error": "Error message",
  "statusCode": 400,
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

### Middleware Integration Tests

Test that middleware executes correctly:

#### Authentication Middleware

- Valid token → Request proceeds
- Invalid token → 401 error
- Expired token → 401 error
- Missing token → 401 error
- Revoked session → 401 error

#### Rate Limiting Middleware

- Requests within limit → Proceed
- Requests exceeding limit → 429 error
- Different endpoints have different limits

#### Error Handler Middleware

- Errors are caught and formatted
- Status codes are correct
- Response format is consistent

## Error Handling

### Error Types

The auth-service uses custom error classes:

```typescript
class BadRequestError extends Error {
  statusCode = 400;
}

class UnauthorizedError extends Error {
  statusCode = 401;
}

class ForbiddenError extends Error {
  statusCode = 403;
}

class NotFoundError extends Error {
  statusCode = 404;
}

class InternalServerError extends Error {
  statusCode = 500;
}
```

### Error Testing Strategy

For each endpoint, test:
1. Success case (200/201)
2. Validation errors (400)
3. Authentication errors (401)
4. Authorization errors (403)
5. Not found errors (404)
6. Internal errors (500)

### Error Response Validation

All error responses should include:
- `error`: Error message string
- `statusCode`: HTTP status code
- `timestamp`: ISO timestamp string

Example test:
```typescript
it('should return 400 with validation error', async () => {
  const response = await request(app)
    .post('/api/v1/auth/register')
    .send({ email: 'invalid' });

  expect(response.status).toBe(400);
  expect(response.body).toMatchObject({
    error: expect.any(String),
    statusCode: 400,
    timestamp: expect.any(String),
  });
});
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property Reflection

After analyzing all acceptance criteria, I identified several areas of redundancy:

1. **Error Response Format**: Properties 19.6 and 21.2 both test that error responses include error, statusCode, and timestamp. These are identical and can be combined.

2. **Input Validation**: Multiple properties test similar validation patterns (invalid email, missing fields, invalid types, out-of-range values). These can be consolidated into broader validation properties.

3. **Authentication**: Multiple properties test authentication with valid/invalid tokens. These follow the same pattern and can be consolidated.

4. **Rate Limiting**: Properties 18.2, 18.3, and 18.4 all test rate limit enforcement for different endpoints. These can be combined into a single property about rate limiting behavior.

After reflection, here are the consolidated correctness properties:

### Authentication and Authorization Properties

Property 1: Valid authentication succeeds
*For any* protected endpoint and valid access token, the request should succeed and return appropriate data
**Validates: Requirements 17.1**

Property 2: Missing authentication fails
*For any* protected endpoint without an access token, the request should return 401 status with error response
**Validates: Requirements 17.2**

Property 3: Invalid authentication fails
*For any* protected endpoint and invalid access token, the request should return 401 status with error response
**Validates: Requirements 17.3**

### Request Validation Properties

Property 4: Valid registration succeeds
*For any* valid registration data (email, password, name, username), the registration endpoint should return 201 status and user object
**Validates: Requirements 3.1**

Property 5: Invalid email format rejected
*For any* invalid email format string, endpoints requiring email should return 400 status with validation error
**Validates: Requirements 3.2, 20.2**

Property 6: Missing required fields rejected
*For any* request with missing required fields, the endpoint should return 400 status with field validation errors
**Validates: Requirements 3.3, 20.1**

Property 7: Invalid data types rejected
*For any* request with invalid data types for fields, the endpoint should return 400 status with type validation error
**Validates: Requirements 20.3**

Property 8: Out-of-range values rejected
*For any* request with out-of-range values, the endpoint should return 400 status with range validation error
**Validates: Requirements 20.4**

### Authentication Flow Properties

Property 9: Valid login succeeds
*For any* valid credentials (email and password), the login endpoint should return 200 status with accessToken and refreshToken
**Validates: Requirements 3.5**

Property 10: Invalid credentials rejected
*For any* invalid credentials, the login endpoint should return 401 status with error message
**Validates: Requirements 3.6**

Property 11: Valid logout succeeds
*For any* valid access token, the logout endpoint should return 200 status and revoke the session
**Validates: Requirements 3.8**

Property 12: Valid token refresh succeeds
*For any* valid refresh token, the refresh endpoint should return 200 status with new accessToken and refreshToken
**Validates: Requirements 3.10**

### Password Management Properties

Property 13: Valid password reset request succeeds
*For any* valid email address of existing user, the forgot password endpoint should return 200 status and send OTP
**Validates: Requirements 4.1**

Property 14: Non-existent email returns not found
*For any* non-existent email address, the forgot password endpoint should return 404 status with error message
**Validates: Requirements 4.2**

Property 15: Valid OTP verification succeeds
*For any* valid OTP code, the verify reset OTP endpoint should return 200 status with success message
**Validates: Requirements 4.3**

Property 16: Invalid OTP rejected
*For any* invalid OTP code, the verify reset OTP endpoint should return 400 status with error message
**Validates: Requirements 4.4**

Property 17: Valid password reset succeeds
*For any* valid reset token and new password, the reset password endpoint should return 200 status and update password
**Validates: Requirements 4.5**

Property 18: Weak password rejected
*For any* weak password (not meeting strength requirements), the reset password endpoint should return 400 status with validation error
**Validates: Requirements 4.7**

### Email Verification Properties

Property 19: Valid email verification succeeds
*For any* valid email verification OTP, the verify email endpoint should return 200 status and mark email as verified
**Validates: Requirements 5.1**

Property 20: Invalid email OTP rejected
*For any* invalid email verification OTP, the verify email endpoint should return 400 status with error message
**Validates: Requirements 5.2**

Property 21: Valid OTP resend succeeds
*For any* valid unverified email address, the resend verification OTP endpoint should return 200 status and send new OTP
**Validates: Requirements 5.4**

### Device Verification Properties

Property 22: Valid device verification succeeds
*For any* valid device verification OTP, the verify device endpoint should return 200 status and mark device as trusted
**Validates: Requirements 6.1**

Property 23: Invalid device OTP rejected
*For any* invalid device verification OTP, the verify device endpoint should return 400 status with error message
**Validates: Requirements 6.2**

### OAuth Properties

Property 24: Valid Google token authentication succeeds
*For any* valid Google ID token, the Google auth endpoint should return 200 status with accessToken and refreshToken
**Validates: Requirements 7.1**

Property 25: Invalid Google token rejected
*For any* invalid Google ID token, the Google auth endpoint should return 401 status with error message
**Validates: Requirements 7.2**

### Two-Factor Authentication Properties

Property 26: Valid TOTP setup succeeds
*For any* valid TOTP token during 2FA setup, the verify setup endpoint should return 200 status with backup codes
**Validates: Requirements 8.2**

Property 27: Invalid TOTP setup rejected
*For any* invalid TOTP token during 2FA setup, the verify setup endpoint should return 400 status with error message
**Validates: Requirements 8.3**

Property 28: Valid 2FA login succeeds
*For any* valid TOTP token during 2FA login, the verify login endpoint should return 200 status with full access tokens
**Validates: Requirements 8.5**

Property 29: Invalid 2FA login rejected
*For any* invalid TOTP token during 2FA login, the verify login endpoint should return 400 status with error message
**Validates: Requirements 8.6**

### Account Management Properties

Property 30: Invalid password for deletion rejected
*For any* invalid password during account deletion, the delete account endpoint should return 401 status with error message
**Validates: Requirements 9.4**

### Profile Management Properties

Property 31: Valid profile update succeeds
*For any* valid profile data, the update profile endpoint should return 200 status and updated profile
**Validates: Requirements 12.2**

Property 32: Invalid profile data rejected
*For any* invalid profile data, the update profile endpoint should return 400 status with validation error
**Validates: Requirements 12.3**

Property 33: Invalid image format rejected
*For any* invalid image format, the upload profile image endpoint should return 400 status with error message
**Validates: Requirements 12.5**

### Onboarding Properties

Property 34: Valid onboarding completion succeeds
*For any* valid onboarding data, the complete onboarding endpoint should return 200 status and mark onboarding as complete
**Validates: Requirements 13.1**

Property 35: Missing onboarding fields rejected
*For any* onboarding request with missing required fields, the endpoint should return 400 status with validation error
**Validates: Requirements 13.2**

### Parent Link Properties

Property 36: Valid parent link creation succeeds
*For any* valid parent link data, the create link endpoint should return 201 status and link object
**Validates: Requirements 14.1**

Property 37: Valid parent link verification succeeds
*For any* valid parent link code, the verify link endpoint should return 200 status and link details
**Validates: Requirements 14.2**

### Location Tracking Properties

Property 38: Valid location update succeeds
*For any* valid GPS coordinates, the update location endpoint should return 200 status and updated location
**Validates: Requirements 15.1**

Property 39: Invalid coordinates rejected
*For any* invalid GPS coordinates, the update location endpoint should return 400 status with validation error
**Validates: Requirements 15.2**

### Internal Service Properties

Property 40: Valid token validation succeeds
*For any* valid access token, the internal validate token endpoint should return 200 status with user data
**Validates: Requirements 16.1**

Property 41: Invalid token validation fails
*For any* invalid access token, the internal validate token endpoint should return 401 status with error message
**Validates: Requirements 16.2**

Property 42: Valid user retrieval succeeds
*For any* valid user ID, the internal get user endpoint should return 200 status with user data
**Validates: Requirements 16.3**

Property 43: Invalid user ID returns not found
*For any* invalid or non-existent user ID, the internal get user endpoint should return 404 status with error message
**Validates: Requirements 16.4**

### Error Response Format Properties

Property 44: Error responses have consistent format
*For any* error response from any endpoint, the response should include error (string), statusCode (number), and timestamp (ISO string)
**Validates: Requirements 19.6, 21.2**

### Success Response Format Properties

Property 45: Successful responses include expected fields
*For any* successful request to any endpoint, the response should include the expected data fields for that endpoint
**Validates: Requirements 21.1**

Property 46: List responses include array and metadata
*For any* endpoint returning a list, the response should include an array of items and metadata (count, pagination, etc.)
**Validates: Requirements 21.3**

Property 47: Token responses include both tokens
*For any* authentication endpoint that issues tokens, the response should include both accessToken and refreshToken
**Validates: Requirements 21.4**

Property 48: User data excludes sensitive fields
*For any* endpoint returning user data, the response should exclude sensitive fields (password, twoFactorSecret, etc.)
**Validates: Requirements 21.5**

### Example-Based Tests

The following scenarios are best tested with specific examples rather than properties:

- Duplicate email registration (Requirement 3.4)
- Login with unverified email (Requirement 3.7)
- Logout without token (Requirement 3.9)
- Resend OTP during cooldown (Requirement 5.5)
- Resend OTP for verified email (Requirement 5.6)
- Resend device OTP (Requirement 6.3)
- Device verification from new location (Requirement 6.4)
- Google auth for new user (Requirement 7.4)
- Google auth for existing user (Requirement 7.5)
- Enable 2FA (Requirement 8.1)
- Disable 2FA (Requirement 8.4)
- 2FA login with backup code (Requirement 8.7)
- Regenerate backup codes (Requirement 8.8)
- Get 2FA status (Requirement 8.9)
- Deactivate account (Requirement 9.1)
- Delete account with valid password (Requirement 9.3)
- Delete OAuth account (Requirement 9.5)
- Confirm reactivation (Requirement 9.6)
- Delete profile image (Requirement 9.8)
- Get sessions (Requirement 10.1)
- Get session by ID (Requirement 10.2)
- Get session for different user (Requirement 10.3)
- Revoke session by ID (Requirement 10.4)
- Revoke all sessions (Requirement 10.5)
- Cleanup expired sessions (Requirement 10.6)
- Get activity (Requirement 11.1)
- Get activity without token (Requirement 11.2)
- Get activity with pagination (Requirement 11.3)
- Get profile (Requirement 12.1)
- Upload profile image (Requirement 12.4)
- Get onboarding status (Requirement 13.4)
- Accept parent link (Requirement 14.3)
- Reject parent link (Requirement 14.4)
- Get location history (Requirement 15.3)
- Protected endpoint with valid token (Requirement 17.1)
- Protected endpoint without token (Requirement 17.2)
- Protected endpoint with invalid token (Requirement 17.3)
- Requests within rate limit (Requirement 18.1)
- Exceed login rate limit (Requirement 18.2)
- Exceed registration rate limit (Requirement 18.3)
- Exceed OTP rate limit (Requirement 18.4)
- Rate limit window expires (Requirement 18.5)
- Validation error format (Requirement 19.1)
- Authentication error format (Requirement 19.2)
- Authorization error format (Requirement 19.3)
- Not found error format (Requirement 19.4)
- Internal error format (Requirement 19.5)
- Malformed JSON (Requirement 20.5)

### Edge Cases

The following edge cases should be handled by property test generators:

- Expired reset token (Requirement 4.6)
- Expired email OTP (Requirement 5.3)
- Expired Google token (Requirement 7.3)
- Already deactivated account (Requirement 9.2)
- Already active account reactivation (Requirement 9.7)
- Already onboarded user (Requirement 13.3)
- Expired parent link (Requirement 14.5)
- Expired access token (Requirement 17.4)
- Revoked session (Requirement 17.5)

