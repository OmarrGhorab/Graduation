# Design Document: Auth Service Unit Tests

## Overview

This design document outlines the comprehensive unit testing strategy for the auth service. The auth service is a Node.js/Express application built with TypeScript that handles authentication, authorization, session management, 2FA, OAuth, and related functionality. The testing approach focuses on achieving good code coverage while maintaining test quality through proper mocking, test isolation, and clear test organization.

The test suite will use Jest as the testing framework and will cover:
- **Utils** (15 files): Core business logic for tokens, sessions, OTP, 2FA, email verification, password reset, etc.
- **Services** (3 files): Higher-level business logic for auth sessions, location tracking, and parent linking
- **Middleware** (7 files): Request processing for authentication, rate limiting, and error handling

The design prioritizes testing critical business logic over trivial code, with a focus on both success and error paths, edge cases, and validation logic.

## Architecture

### Test Structure

The test suite will follow a directory structure that mirrors the source code:

```
auth-service/
├── src/
│   ├── utils/
│   ├── services/
│   └── middleware/
└── tests/
    ├── unit/
    │   ├── utils/
    │   │   ├── tokens.test.ts
    │   │   ├── sessions.test.ts
    │   │   ├── otp.test.ts
    │   │   ├── twoFactor.test.ts
    │   │   ├── emailVerification.test.ts
    │   │   ├── passwordReset.test.ts
    │   │   ├── cookies.test.ts
    │   │   ├── device.test.ts
    │   │   └── errors.test.ts
    │   ├── services/
    │   │   ├── authSession.service.test.ts
    │   │   ├── location.service.test.ts
    │   │   └── parentLink.service.test.ts
    │   └── middleware/
    │       ├── auth.middleware.test.ts
    │       ├── rateLimiter.middleware.test.ts
    │       └── errorHandler.test.ts
    ├── mocks/
    │   ├── prisma.mock.ts
    │   ├── redis.mock.ts
    │   ├── email.mock.ts
    │   └── cloudinary.mock.ts
    └── setup/
        └── jest.setup.ts
```

### Mock Strategy

All external dependencies will be mocked to ensure test isolation and speed:

1. **Prisma Client**: Mock all database operations using `jest.mock()` with a factory function that returns mock implementations
2. **Redis Client**: Mock all cache operations with in-memory storage for test isolation
3. **Email Service (Resend)**: Mock email sending to prevent actual emails during tests
4. **Cloudinary**: Mock image upload operations
5. **External HTTP Requests**: Mock fetch calls for location services

### Test Patterns

Each test file will follow these patterns:

1. **Arrange-Act-Assert (AAA)**: Clear separation of test setup, execution, and verification
2. **Descriptive Test Names**: Use "should [expected behavior] when [condition]" format
3. **Grouped Tests**: Use `describe` blocks to group related tests
4. **Shared Setup**: Use `beforeEach` and `afterEach` for common setup and cleanup
5. **Mock Isolation**: Reset all mocks between tests to prevent pollution

## Components and Interfaces

### Jest Configuration

The test suite requires a `jest.config.cjs` file with the following configuration:

```javascript
/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests', '<rootDir>/src'],
  testMatch: ['**/__tests__/**/*.ts', '**/?(*.)+(spec|test).ts'],
  setupFilesAfterEnv: ['<rootDir>/tests/setup/jest.setup.ts'],
  transform: {
    '^.+\\.ts$': 'ts-jest',
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/**/*.test.ts',
    '!src/**/*.spec.ts',
    '!src/main.ts',
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'json', 'html', 'lcov'],
  coverageThresholds: {
    'src/utils/**/*.ts': {
      lines: 80,
      functions: 80,
      branches: 75,
      statements: 80,
    },
    'src/services/**/*.ts': {
      lines: 75,
      functions: 75,
      branches: 70,
      statements: 75,
    },
    'src/middleware/**/*.ts': {
      lines: 70,
      functions: 70,
      branches: 65,
      statements: 70,
    },
  },
  moduleFileExtensions: ['ts', 'js', 'json'],
  verbose: true,
  testTimeout: 10000,
  clearMocks: true,
  resetMocks: true,
  restoreMocks: true,
};
```

### Mock Implementations

#### Prisma Mock

The Prisma mock will provide a complete mock of the Prisma client with all models:

```typescript
// tests/mocks/prisma.mock.ts
export const mockPrisma = {
  user: {
    findUnique: jest.fn(),
    findFirst: jest.fn(),
    findMany: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    delete: jest.fn(),
    updateMany: jest.fn(),
    deleteMany: jest.fn(),
  },
  session: {
    findUnique: jest.fn(),
    findFirst: jest.fn(),
    findMany: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    delete: jest.fn(),
    updateMany: jest.fn(),
    deleteMany: jest.fn(),
  },
  userDevice: {
    findUnique: jest.fn(),
    findFirst: jest.fn(),
    findMany: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    delete: jest.fn(),
  },
  // ... other models
};
```

#### Redis Mock

The Redis mock will simulate Redis operations with in-memory storage:

```typescript
// tests/mocks/redis.mock.ts
export class RedisMock {
  private store: Map<string, { value: string; expiry?: number }> = new Map();
  private sets: Map<string, Set<string>> = new Map();

  async get(key: string): Promise<string | null> {
    const entry = this.store.get(key);
    if (!entry) return null;
    if (entry.expiry && Date.now() > entry.expiry) {
      this.store.delete(key);
      return null;
    }
    return entry.value;
  }

  async set(key: string, value: string, mode?: string, duration?: number): Promise<string> {
    const expiry = mode === 'EX' && duration ? Date.now() + duration * 1000 : undefined;
    this.store.set(key, { value, expiry });
    return 'OK';
  }

  async del(...keys: string[]): Promise<number> {
    let count = 0;
    for (const key of keys) {
      if (this.store.delete(key)) count++;
    }
    return count;
  }

  async ttl(key: string): Promise<number> {
    const entry = this.store.get(key);
    if (!entry) return -2;
    if (!entry.expiry) return -1;
    const remaining = Math.ceil((entry.expiry - Date.now()) / 1000);
    return remaining > 0 ? remaining : -2;
  }

  async incr(key: string): Promise<number> {
    const current = await this.get(key);
    const newValue = (parseInt(current || '0', 10) + 1).toString();
    await this.set(key, newValue);
    return parseInt(newValue, 10);
  }

  async expire(key: string, seconds: number): Promise<number> {
    const entry = this.store.get(key);
    if (!entry) return 0;
    entry.expiry = Date.now() + seconds * 1000;
    return 1;
  }

  async sadd(key: string, ...members: string[]): Promise<number> {
    if (!this.sets.has(key)) {
      this.sets.set(key, new Set());
    }
    const set = this.sets.get(key)!;
    let added = 0;
    for (const member of members) {
      if (!set.has(member)) {
        set.add(member);
        added++;
      }
    }
    return added;
  }

  async smembers(key: string): Promise<string[]> {
    const set = this.sets.get(key);
    return set ? Array.from(set) : [];
  }

  async srem(key: string, ...members: string[]): Promise<number> {
    const set = this.sets.get(key);
    if (!set) return 0;
    let removed = 0;
    for (const member of members) {
      if (set.delete(member)) removed++;
    }
    return removed;
  }

  pipeline() {
    const commands: Array<() => Promise<any>> = [];
    const results: Array<[Error | null, any]> = [];

    const pipelineProxy = new Proxy(this, {
      get: (target, prop) => {
        if (prop === 'exec') {
          return async () => {
            for (const cmd of commands) {
              try {
                const result = await cmd();
                results.push([null, result]);
              } catch (error) {
                results.push([error as Error, null]);
              }
            }
            return results;
          };
        }
        if (typeof (target as any)[prop] === 'function') {
          return (...args: any[]) => {
            commands.push(() => (target as any)[prop](...args));
            return pipelineProxy;
          };
        }
        return (target as any)[prop];
      },
    });

    return pipelineProxy;
  }

  clear() {
    this.store.clear();
    this.sets.clear();
  }
}
```

### Test Utilities

Common test utilities will be extracted to reduce duplication:

```typescript
// tests/utils/testHelpers.ts

export function createMockUser(overrides = {}) {
  return {
    id: 'user-123',
    email: 'test@example.com',
    role: 'USER',
    isActive: true,
    deletedAt: null,
    ...overrides,
  };
}

export function createMockSession(overrides = {}) {
  return {
    id: 'session-123',
    userId: 'user-123',
    sessionToken: 'jti-123',
    refreshToken: 'refresh-jti-123',
    isActive: true,
    isRevoked: false,
    expiresAt: new Date(Date.now() + 900000), // 15 minutes
    lastActivityAt: new Date(),
    ...overrides,
  };
}

export function createMockRequest(overrides = {}) {
  return {
    headers: {},
    cookies: {},
    query: {},
    body: {},
    user: undefined,
    ...overrides,
  };
}

export function createMockResponse() {
  const res: any = {
    status: jest.fn().mockReturnThis(),
    json: jest.fn().mockReturnThis(),
    send: jest.fn().mockReturnThis(),
    cookie: jest.fn().mockReturnThis(),
    clearCookie: jest.fn().mockReturnThis(),
  };
  return res;
}

export function createMockNext() {
  return jest.fn();
}

export function advanceTime(ms: number) {
  jest.advanceTimersByTime(ms);
}
```

## Data Models

### Test Data Structures

The test suite will use consistent test data structures that match the application's data models:

#### User Test Data
```typescript
interface TestUser {
  id: string;
  email: string;
  password: string;
  role: 'USER' | 'ADMIN';
  isActive: boolean;
  deletedAt: Date | null;
  emailVerified: boolean;
  twoFactorEnabled: boolean;
  twoFactorSecret: string | null;
}
```

#### Session Test Data
```typescript
interface TestSession {
  id: string;
  userId: string;
  deviceId: string | null;
  sessionToken: string; // JWT jti
  refreshToken: string | null; // Refresh token jti
  ipAddress: string | null;
  userAgent: string | null;
  location: string | null;
  expiresAt: Date;
  refreshExpiresAt: Date | null;
  isActive: boolean;
  isRevoked: boolean;
  lastActivityAt: Date;
}
```

#### Device Test Data
```typescript
interface TestDevice {
  id: string;
  userId: string;
  deviceFingerprint: string;
  deviceName: string;
  platform: string;
  ipAddress: string | null;
  userAgent: string | null;
  isTrusted: boolean;
  lastLoginAt: Date | null;
}
```

### Mock Data Factories

Test data factories will generate consistent test data:

```typescript
// tests/factories/userFactory.ts
export function buildUser(overrides: Partial<TestUser> = {}): TestUser {
  return {
    id: `user-${Math.random().toString(36).substr(2, 9)}`,
    email: `test-${Math.random().toString(36).substr(2, 9)}@example.com`,
    password: 'hashedPassword123',
    role: 'USER',
    isActive: true,
    deletedAt: null,
    emailVerified: true,
    twoFactorEnabled: false,
    twoFactorSecret: null,
    ...overrides,
  };
}

// tests/factories/sessionFactory.ts
export function buildSession(overrides: Partial<TestSession> = {}): TestSession {
  return {
    id: `session-${Math.random().toString(36).substr(2, 9)}`,
    userId: 'user-123',
    deviceId: 'device-123',
    sessionToken: `jti-${Math.random().toString(36).substr(2, 9)}`,
    refreshToken: `refresh-jti-${Math.random().toString(36).substr(2, 9)}`,
    ipAddress: '192.168.1.1',
    userAgent: 'Mozilla/5.0',
    location: 'San Francisco, CA, USA',
    expiresAt: new Date(Date.now() + 900000),
    refreshExpiresAt: new Date(Date.now() + 2592000000),
    isActive: true,
    isRevoked: false,
    lastActivityAt: new Date(),
    ...overrides,
  };
}
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property Reflection

After analyzing all acceptance criteria, I identified several areas where properties can be consolidated:

1. **Token operations** (3.1-3.9): Multiple properties about token generation and verification can be grouped by operation type
2. **Session operations** (4.1-4.7): Session CRUD operations share common patterns
3. **OTP operations** (5.1-5.7): OTP lifecycle properties follow similar patterns
4. **2FA operations** (6.1-6.9): Encryption round-trip and backup code management can be consolidated
5. **Mock isolation** (2.6-2.7): Test isolation properties apply universally

The following properties represent the unique, non-redundant validation requirements:

### Token Management Properties

**Property 1: Access token generation produces valid JWT structure**
*For any* user with id and role, generating an access token should produce a JWT that when decoded contains the user's id as `sub`, a unique `jti`, the user's role, and type "access".
**Validates: Requirements 3.1**

**Property 2: Refresh token storage includes Redis persistence**
*For any* user id, generating a refresh token should store the token's jti in Redis with the correct TTL and add the jti to the user's refresh token set.
**Validates: Requirements 3.2**

**Property 3: Valid access token verification returns payload**
*For any* valid access token, verification should successfully decode and return the payload containing sub, jti, role, and type fields.
**Validates: Requirements 3.3**

**Property 4: Invalid token verification throws appropriate errors**
*For any* malformed, tampered, or invalid token, verification should throw an error indicating the token is invalid.
**Validates: Requirements 3.4**

**Property 5: Refresh token verification checks Redis existence**
*For any* refresh token, verification should check Redis for the token's jti and reject tokens not found in Redis.
**Validates: Requirements 3.6, 3.7**

**Property 6: Token rotation revokes old and creates new**
*For any* valid refresh token jti and user id, rotation should delete the old jti from Redis and create a new refresh token with a new jti.
**Validates: Requirements 3.8**

**Property 7: Bulk token revocation deletes all user tokens**
*For any* user id, revoking all refresh tokens should delete all jtis from the user's refresh token set and delete all corresponding token keys from Redis.
**Validates: Requirements 3.9**

### Session Management Properties

**Property 8: Session creation stores complete session data**
*For any* valid session parameters (userId, deviceId, tokens, network info, expiry), creating a session should store a database record with all provided fields and set isActive to true and isRevoked to false.
**Validates: Requirements 4.1**

**Property 9: Session activity update modifies timestamp**
*For any* active, non-revoked, non-expired session token, updating activity should set lastActivityAt to the current time.
**Validates: Requirements 4.2**

**Property 10: Session revocation deletes session and refresh token**
*For any* valid session id and user id, revoking the session should delete the session from the database and revoke the associated refresh token in Redis.
**Validates: Requirements 4.3**

**Property 11: Bulk session revocation respects current session exclusion**
*For any* user id and current session token, revoking all sessions with includeCurrent=false should delete all sessions except the current one, while includeCurrent=true should delete all sessions including current.
**Validates: Requirements 4.4**

**Property 12: Session details parsing extracts device information**
*For any* session with user agent string, getting session details should parse and return browser, OS, platform, and device name information.
**Validates: Requirements 4.5, 4.6**

**Property 13: Expired session cleanup deletes only expired sessions**
*For any* user id, cleaning up expired sessions should delete only sessions where expiresAt is less than the current time, leaving active sessions unchanged.
**Validates: Requirements 4.7**

### OTP Management Properties

**Property 14: OTP generation produces numeric code with correct length**
*For any* requested OTP length (default 6), generating an OTP should produce a string containing only digits 0-9 with exactly the specified length.
**Validates: Requirements 5.1**

**Property 15: OTP storage includes Redis persistence with TTL**
*For any* target identifier and OTP code, storing the OTP should set the value in Redis with the configured TTL and reset the attempt counter.
**Validates: Requirements 5.2**

**Property 16: Correct OTP verification consumes the OTP**
*For any* target identifier and matching OTP code, verification should return true and delete both the OTP value and attempt counter from Redis.
**Validates: Requirements 5.3**

**Property 17: Incorrect OTP verification increments attempts**
*For any* target identifier and non-matching OTP code, verification should increment the attempt counter in Redis and return false.
**Validates: Requirements 5.4**

**Property 18: OTP attempt limit triggers cooldown**
*For any* target identifier, when OTP verification attempts reach the configured limit, the system should set a cooldown key in Redis preventing further attempts.
**Validates: Requirements 5.5, 5.6**

**Property 19: Non-consuming OTP verification preserves OTP**
*For any* target identifier and matching OTP code, verifying without consuming should return true but leave the OTP value in Redis.
**Validates: Requirements 5.7**

### Two-Factor Authentication Properties

**Property 20: 2FA secret generation produces base32 encoded value**
*For any* user email and service name, generating a 2FA secret should produce a base32 encoded string and an otpauth URL.
**Validates: Requirements 6.1**

**Property 21: QR code generation produces valid data URL**
*For any* otpauth URL, generating a QR code should produce a data URL string starting with "data:image/png;base64,".
**Validates: Requirements 6.2**

**Property 22: Valid TOTP token verification succeeds**
*For any* valid secret and correctly generated TOTP token within the time window, verification should return true.
**Validates: Requirements 6.3**

**Property 23: Invalid TOTP token verification fails**
*For any* valid secret and incorrect TOTP token, verification should return false.
**Validates: Requirements 6.4**

**Property 24: Secret encryption round-trip preserves value**
*For any* plaintext secret, encrypting then decrypting should return the original secret value, and the encrypted value should differ from the plaintext.
**Validates: Requirements 6.5, 6.6**

**Property 25: Backup code generation produces correct count and format**
*For any* requested count, generating backup codes should produce exactly that many codes, each being 8 hexadecimal characters.
**Validates: Requirements 6.7**

**Property 26: Valid backup code verification removes code from list**
*For any* list of encrypted backup codes and a valid code, verification should return valid=true and return a list with that code removed.
**Validates: Requirements 6.8**

**Property 27: Invalid backup code verification preserves list**
*For any* list of encrypted backup codes and an invalid code, verification should return valid=false and return the original list unchanged.
**Validates: Requirements 6.9**

### Email Verification Properties

**Property 28: Email verification cooldown returns remaining time**
*For any* email address, checking cooldown should return the remaining seconds if a cooldown is active, or 0 if no cooldown exists.
**Validates: Requirements 7.1**

**Property 29: Email verification cooldown applies progressive duration**
*For any* email address, setting cooldown after exceeding attempt limit should apply standard cooldown on first violation and longer cooldown on subsequent violations.
**Validates: Requirements 7.2**

**Property 30: Email verification allowed check considers multiple factors**
*For any* email address, checking if verification is allowed should return false if cooldown is active OR attempts exceed limit, and true otherwise.
**Validates: Requirements 7.3**

**Property 31: Email verification cooldown clear resets all state**
*For any* email address, clearing cooldown should delete both the cooldown key and the attempts key from Redis.
**Validates: Requirements 7.4**

**Property 32: Resend OTP cooldown enforces rate limiting**
*For any* email address, when resend attempts exceed the limit, the system should set a cooldown preventing further resend requests.
**Validates: Requirements 7.5, 7.6**

### Password Reset Properties

**Property 33: Password reset token generation stores in Redis**
*For any* user id, generating a password reset token should create a unique token and store the mapping from token to user id in Redis with the configured TTL.
**Validates: Requirements 8.1**

**Property 34: Valid password reset token verification returns user ID**
*For any* valid, non-expired password reset token, verification should return the associated user id.
**Validates: Requirements 8.2**

**Property 35: Invalid password reset token verification throws error**
*For any* non-existent, malformed, or expired password reset token, verification should throw an error.
**Validates: Requirements 8.4**

**Property 36: Password reset token consumption deletes from Redis**
*For any* valid password reset token, consuming the token should delete it from Redis, making it unusable for subsequent attempts.
**Validates: Requirements 8.5**

### Auth Session Service Properties

**Property 37: Session expiry calculation uses environment variables**
*For any* call to calculate session expiry, the function should return expiresAt and refreshExpiresAt dates based on ACCESS_TOKEN_TTL_SEC and REFRESH_TOKEN_TTL_SEC environment variables.
**Validates: Requirements 9.1**

**Property 38: Token generation creates both access and refresh tokens**
*For any* user id and role, generating tokens should return both an access token and refresh token, each with their respective jtis.
**Validates: Requirements 9.2**

**Property 39: Device lookup reuses existing devices by fingerprint**
*For any* user id and device fingerprint, finding or creating a device should return the existing device if one exists with that fingerprint, rather than creating a duplicate.
**Validates: Requirements 9.3**

**Property 40: New device creation stores complete device information**
*For any* user id, device fingerprint, and device name, creating a new device should store a database record with all provided fields.
**Validates: Requirements 9.4**

**Property 41: Complete session creation coordinates all operations**
*For any* request, user id, and role, creating a device and session should successfully complete device creation, token generation, and session creation in sequence.
**Validates: Requirements 9.5**

**Property 42: Temporary 2FA session excludes refresh token**
*For any* request, user id, role, and device id, creating a temporary 2FA session should create a session with a null refreshToken and null refreshExpiresAt.
**Validates: Requirements 9.6**

### Location Service Properties

**Property 43: Session location update modifies database field**
*For any* session token and location data, updating session location should set the location field in the database to the provided value.
**Validates: Requirements 10.1**

**Property 44: Location API failure handling returns null gracefully**
*For any* IP address, when the location API fails or times out, getting location should return null without throwing an error.
**Validates: Requirements 10.3**

### Parent Link Service Properties

**Property 45: Parent link creation stores with expiry**
*For any* parent user id, child user id, and link code, creating a parent link should store the link in the database with an expiry timestamp.
**Validates: Requirements 11.1**

**Property 46: Parent link verification checks expiry and validity**
*For any* link code, verifying a parent link should check that the link exists and has not expired before returning the link data.
**Validates: Requirements 11.2**

**Property 47: Parent link acceptance creates relationship and deletes link**
*For any* valid link code, accepting a parent link should create the parent-child relationship in the database and delete the link record.
**Validates: Requirements 11.3**

**Property 48: Parent link rejection deletes link**
*For any* valid link code, rejecting a parent link should delete the link record from the database.
**Validates: Requirements 11.4**

### Authentication Middleware Properties

**Property 49: Valid token authentication attaches user info**
*For any* valid access token with an active session and active user, the authentication middleware should attach user id, role, and jti to the request object.
**Validates: Requirements 12.1**

**Property 50: Invalid token authentication throws UnauthorizedError**
*For any* invalid, malformed, or expired access token, the authentication middleware should throw an UnauthorizedError.
**Validates: Requirements 12.3**

**Property 51: Revoked session authentication throws UnauthorizedError**
*For any* valid access token with a revoked or expired session, the authentication middleware should throw an UnauthorizedError.
**Validates: Requirements 12.6**

**Property 52: Authenticated request updates session activity**
*For any* successful authentication, the middleware should asynchronously update the session's lastActivityAt timestamp.
**Validates: Requirements 12.9**

### Rate Limiting Middleware Properties

**Property 53: Requests within limit are allowed**
*For any* endpoint with rate limiting, when request count is below the limit, the middleware should call next() and allow the request.
**Validates: Requirements 13.1**

**Property 54: Requests exceeding limit return 429**
*For any* endpoint with rate limiting, when request count exceeds the limit, the middleware should return a 429 status code.
**Validates: Requirements 13.2**

**Property 55: Rate limit window expiry resets counter**
*For any* endpoint with rate limiting, after the time window expires, the request counter should reset to zero.
**Validates: Requirements 13.3**

### Error Handler Middleware Properties

**Property 56: Known error types return appropriate status codes**
*For any* known error type (UnauthorizedError, ValidationError, etc.), the error handler should return the error's designated status code and message.
**Validates: Requirements 14.1**

**Property 57: Unknown errors return 500 status code**
*For any* error that is not a known error type, the error handler should return a 500 status code with a generic error message.
**Validates: Requirements 14.2**

**Property 58: Error occurrence triggers logging**
*For any* error passed to the error handler, the middleware should log the error details before sending the response.
**Validates: Requirements 14.3**

### Test Infrastructure Properties

**Property 59: Test execution makes no real external calls**
*For any* test in the test suite, execution should not make real calls to Prisma, Redis, email services, Cloudinary, or external HTTP endpoints.
**Validates: Requirements 2.6**

**Property 60: Test completion cleans up mocks**
*For any* test in the test suite, after completion, all mocks should be reset to prevent state pollution affecting subsequent tests.
**Validates: Requirements 2.7**

**Property 61: Test failure provides clear error messages**
*For any* test that fails, the error message should clearly indicate what was expected, what was received, and which assertion failed.
**Validates: Requirements 15.7**

**Property 62: Tests run in any order with consistent results**
*For any* permutation of test execution order, the results should be identical, demonstrating complete test isolation.
**Validates: Requirements 17.4**

**Property 63: Test completion leaves no hanging resources**
*For any* test suite execution, after completion, there should be no hanging processes, open connections, or timers remaining.
**Validates: Requirements 17.5**

**Property 64: Tests require no external services**
*For any* test in the test suite, execution should not require network access or external services to be running.
**Validates: Requirements 18.2**

## Error Handling

### Mock Error Scenarios

The test suite must handle errors in mock implementations:

1. **Database Errors**: Prisma mocks should be able to simulate database connection failures, constraint violations, and query timeouts
2. **Cache Errors**: Redis mocks should be able to simulate connection failures and operation timeouts
3. **Network Errors**: HTTP mocks should be able to simulate network failures, timeouts, and invalid responses
4. **Validation Errors**: Mocks should be able to trigger validation errors for testing error paths

### Test Error Handling

Tests should handle errors appropriately:

1. **Expected Errors**: Use `expect().toThrow()` or `expect().rejects.toThrow()` for testing error paths
2. **Unexpected Errors**: Tests should fail with clear messages when unexpected errors occur
3. **Async Errors**: Use proper async/await patterns to catch errors in asynchronous code
4. **Mock Errors**: Reset mocks between tests to prevent error state pollution

### Error Recovery

The test suite should ensure proper cleanup even when tests fail:

1. **afterEach Hooks**: Always run cleanup code in afterEach hooks, even if tests fail
2. **Try-Finally Blocks**: Use try-finally for critical cleanup operations
3. **Mock Reset**: Ensure mocks are reset even if tests throw errors
4. **Resource Cleanup**: Close any open resources (timers, connections) in cleanup hooks

## Testing Strategy

### Dual Testing Approach

The test suite will use both unit tests and property-based tests:

**Unit Tests**:
- Test specific examples and edge cases
- Test error conditions and validation logic
- Test integration points between components
- Focus on concrete scenarios with known inputs and outputs

**Property-Based Tests**:
- Test universal properties across many generated inputs
- Verify correctness properties hold for all valid inputs
- Use libraries like `fast-check` for property-based testing
- Run minimum 100 iterations per property test

Both approaches are complementary and necessary for comprehensive coverage. Unit tests catch specific bugs and edge cases, while property tests verify general correctness across the input space.

### Test Organization

Tests will be organized by module:

1. **Utils Tests**: One test file per utility module (tokens.test.ts, sessions.test.ts, etc.)
2. **Services Tests**: One test file per service module
3. **Middleware Tests**: One test file per middleware module
4. **Shared Mocks**: Centralized mock implementations in tests/mocks/
5. **Test Utilities**: Shared test helpers in tests/utils/

### Test Naming Convention

All tests will follow the pattern: `should [expected behavior] when [condition]`

Examples:
- `should return decoded payload when verifying valid access token`
- `should throw UnauthorizedError when access token is expired`
- `should increment attempt counter when OTP verification fails`

### Coverage Targets

The test suite aims for the following coverage targets:

- **Utils**: 80% line coverage (high priority, core business logic)
- **Services**: 75% line coverage (important business logic)
- **Middleware**: 70% line coverage (request processing logic)

Coverage should focus on meaningful tests, not just hitting coverage numbers. Trivial code (getters, simple formatters) may have lower coverage if testing provides little value.

### Property-Based Testing Configuration

Each property test will:

1. **Run 100+ iterations**: Ensure sufficient input coverage through randomization
2. **Reference design property**: Include comment with property number and text
3. **Tag format**: `// Feature: auth-service-unit-tests, Property N: [property text]`
4. **Use appropriate generators**: Generate realistic test data matching domain constraints
5. **Shrink on failure**: Use library's shrinking to find minimal failing case

Example property test structure:

```typescript
// Feature: auth-service-unit-tests, Property 24: Secret encryption round-trip preserves value
test('should preserve secret value after encrypt-decrypt round trip', () => {
  fc.assert(
    fc.property(
      fc.string({ minLength: 10, maxLength: 100 }), // Generate random secrets
      (secret) => {
        const encrypted = encryptSecret(secret);
        const decrypted = decryptSecret(encrypted);
        
        expect(encrypted).not.toBe(secret); // Encrypted differs from plaintext
        expect(decrypted).toBe(secret); // Round trip preserves value
      }
    ),
    { numRuns: 100 }
  );
});
```

### Test Execution

Tests will be executed using Jest commands:

- `npm test`: Run all tests once (for CI)
- `npm run test:watch`: Run tests in watch mode (for development)
- `npm run test:coverage`: Run tests with coverage report

### Continuous Integration

Tests will run in CI with:

1. **Automated execution**: Tests run on every commit/PR
2. **Coverage reporting**: Coverage reports uploaded to CI artifacts
3. **Fast feedback**: Tests complete in under 30 seconds
4. **No external dependencies**: All dependencies mocked
5. **Deterministic results**: Tests produce same results on every run
