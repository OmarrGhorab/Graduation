# Requirements Document

## Introduction

This document specifies the requirements for implementing comprehensive unit tests for the auth service. The auth service is a Node.js/Express application built with TypeScript that handles authentication, authorization, session management, 2FA, OAuth, and related functionality. Currently, the service has no unit tests. This specification defines the testing requirements to achieve good code coverage while maintaining test quality, focusing on critical business logic in utils, services, and middleware.

## Glossary

- **Test_Suite**: The complete collection of unit tests for the auth service
- **Jest**: The testing framework to be configured in the project
- **Mock**: A test double that simulates external dependencies (Prisma, Redis, email service, etc.)
- **Utils**: Utility modules containing business logic for tokens, sessions, OTP, 2FA, email verification, etc.
- **Services**: Service modules containing higher-level business logic (authSession, location, parentLink)
- **Middleware**: Express middleware functions for authentication, rate limiting, error handling, etc.
- **Coverage**: The percentage of code lines/branches executed during test runs
- **Test_Isolation**: Tests that run independently without side effects or shared state
- **Edge_Case**: Boundary conditions and unusual inputs that must be tested

## Requirements

### Requirement 1: Test Framework Configuration

**User Story:** As a developer, I want Jest properly configured for unit testing, so that I can run tests efficiently with proper TypeScript and ES module support.

#### Acceptance Criteria

1. THE Test_Suite SHALL use Jest as the testing framework
2. WHEN tests are executed, THE Test_Suite SHALL support TypeScript without compilation errors
3. WHEN tests are executed, THE Test_Suite SHALL support ES modules syntax
4. THE Test_Suite SHALL include a jest.config.cjs configuration file with appropriate settings
5. THE Test_Suite SHALL configure coverage reporting with threshold targets

### Requirement 2: Mock Infrastructure

**User Story:** As a developer, I want external dependencies properly mocked, so that tests run fast and isolated without real database or service calls.

#### Acceptance Criteria

1. THE Test_Suite SHALL mock Prisma client for all database operations
2. THE Test_Suite SHALL mock Redis client for all cache operations
3. THE Test_Suite SHALL mock email service (Resend) for all email operations
4. THE Test_Suite SHALL mock Cloudinary for all image upload operations
5. THE Test_Suite SHALL mock external HTTP requests for location services
6. WHEN a test runs, THE Test_Suite SHALL ensure no real external service calls are made
7. WHEN tests complete, THE Test_Suite SHALL clean up all mocks to prevent test pollution

### Requirement 3: Utils Testing - Token Management

**User Story:** As a developer, I want comprehensive tests for token utilities, so that JWT generation, verification, and revocation work correctly.

#### Acceptance Criteria

1. WHEN generating an access token, THE Test_Suite SHALL verify the token contains correct payload structure
2. WHEN generating a refresh token, THE Test_Suite SHALL verify the token is stored in Redis with correct TTL
3. WHEN verifying a valid access token, THE Test_Suite SHALL return the decoded payload
4. WHEN verifying an invalid access token, THE Test_Suite SHALL throw an appropriate error
5. WHEN verifying an expired access token, THE Test_Suite SHALL throw a TokenExpiredError
6. WHEN verifying a refresh token, THE Test_Suite SHALL check Redis for token existence
7. WHEN verifying a revoked refresh token, THE Test_Suite SHALL throw an error
8. WHEN rotating a refresh token, THE Test_Suite SHALL revoke the old token and create a new one
9. WHEN revoking all user refresh tokens, THE Test_Suite SHALL delete all tokens from Redis
10. THE Test_Suite SHALL test edge cases including malformed tokens and missing secrets

### Requirement 4: Utils Testing - Session Management

**User Story:** As a developer, I want comprehensive tests for session utilities, so that session creation, updates, and revocation work correctly.

#### Acceptance Criteria

1. WHEN creating a session, THE Test_Suite SHALL verify the session is stored in the database with correct fields
2. WHEN updating session activity, THE Test_Suite SHALL verify the lastActivityAt timestamp is updated
3. WHEN revoking a session, THE Test_Suite SHALL verify the session is deleted and refresh token is revoked
4. WHEN revoking all user sessions, THE Test_Suite SHALL verify all sessions are deleted except current if specified
5. WHEN getting session details, THE Test_Suite SHALL verify device info and status are correctly parsed
6. WHEN parsing user agent strings, THE Test_Suite SHALL correctly identify browser, OS, and platform
7. WHEN cleaning up expired sessions, THE Test_Suite SHALL delete only expired sessions
8. THE Test_Suite SHALL test edge cases including missing device info and invalid session IDs

### Requirement 5: Utils Testing - OTP Management

**User Story:** As a developer, I want comprehensive tests for OTP utilities, so that OTP generation, verification, and cooldown logic work correctly.

#### Acceptance Criteria

1. WHEN generating an OTP, THE Test_Suite SHALL verify the OTP is numeric and has correct length
2. WHEN storing an OTP, THE Test_Suite SHALL verify it is stored in Redis with correct TTL
3. WHEN verifying a correct OTP, THE Test_Suite SHALL return true and delete the OTP
4. WHEN verifying an incorrect OTP, THE Test_Suite SHALL increment attempt counter
5. WHEN OTP attempts exceed limit, THE Test_Suite SHALL set cooldown period
6. WHEN verifying OTP during cooldown, THE Test_Suite SHALL return false
7. WHEN verifying OTP without consuming, THE Test_Suite SHALL not delete the OTP on success
8. THE Test_Suite SHALL test edge cases including expired OTPs and missing OTPs

### Requirement 6: Utils Testing - Two-Factor Authentication

**User Story:** As a developer, I want comprehensive tests for 2FA utilities, so that TOTP generation, verification, and backup codes work correctly.

#### Acceptance Criteria

1. WHEN generating a 2FA secret, THE Test_Suite SHALL verify the secret is base32 encoded
2. WHEN generating a QR code, THE Test_Suite SHALL verify the QR code data URL is valid
3. WHEN verifying a valid TOTP token, THE Test_Suite SHALL return true
4. WHEN verifying an invalid TOTP token, THE Test_Suite SHALL return false
5. WHEN encrypting a secret, THE Test_Suite SHALL verify the encrypted value differs from plaintext
6. WHEN decrypting a secret, THE Test_Suite SHALL verify it matches the original plaintext
7. WHEN generating backup codes, THE Test_Suite SHALL verify correct count and format
8. WHEN verifying a valid backup code, THE Test_Suite SHALL remove it from the list
9. WHEN verifying an invalid backup code, THE Test_Suite SHALL keep the list unchanged
10. THE Test_Suite SHALL test edge cases including missing encryption keys and malformed codes

### Requirement 7: Utils Testing - Email Verification

**User Story:** As a developer, I want comprehensive tests for email verification utilities, so that cooldown and rate limiting work correctly.

#### Acceptance Criteria

1. WHEN checking email verification cooldown, THE Test_Suite SHALL return remaining seconds
2. WHEN setting email verification cooldown, THE Test_Suite SHALL apply progressive cooldown based on attempts
3. WHEN checking if email verification is allowed, THE Test_Suite SHALL consider both cooldown and attempts
4. WHEN clearing email verification cooldown, THE Test_Suite SHALL reset both cooldown and attempts
5. WHEN checking resend OTP cooldown, THE Test_Suite SHALL return remaining seconds
6. WHEN resend OTP attempts exceed limit, THE Test_Suite SHALL set cooldown
7. THE Test_Suite SHALL test edge cases including expired cooldowns and boundary attempt counts

### Requirement 8: Utils Testing - Password Reset

**User Story:** As a developer, I want comprehensive tests for password reset utilities, so that password reset token generation and verification work correctly.

#### Acceptance Criteria

1. WHEN generating a password reset token, THE Test_Suite SHALL verify the token is stored in Redis
2. WHEN verifying a valid password reset token, THE Test_Suite SHALL return the user ID
3. WHEN verifying an expired password reset token, THE Test_Suite SHALL throw an error
4. WHEN verifying an invalid password reset token, THE Test_Suite SHALL throw an error
5. WHEN consuming a password reset token, THE Test_Suite SHALL delete it from Redis
6. THE Test_Suite SHALL test edge cases including missing tokens and malformed tokens

### Requirement 9: Services Testing - Auth Session Service

**User Story:** As a developer, I want comprehensive tests for auth session service, so that session creation with device tracking works correctly.

#### Acceptance Criteria

1. WHEN calculating session expiry, THE Test_Suite SHALL verify correct expiry dates based on environment variables
2. WHEN generating tokens, THE Test_Suite SHALL verify both access and refresh tokens are created
3. WHEN finding or creating a device, THE Test_Suite SHALL reuse existing devices with same fingerprint
4. WHEN creating a new device, THE Test_Suite SHALL store correct device information
5. WHEN creating a device and session, THE Test_Suite SHALL coordinate device creation, token generation, and session creation
6. WHEN creating a temporary 2FA session, THE Test_Suite SHALL create session without refresh token
7. THE Test_Suite SHALL test edge cases including missing device info and duplicate devices

### Requirement 10: Services Testing - Location Service

**User Story:** As a developer, I want comprehensive tests for location service, so that location tracking and updates work correctly.

#### Acceptance Criteria

1. WHEN updating session location, THE Test_Suite SHALL update the location field in the database
2. WHEN getting location from IP, THE Test_Suite SHALL handle private IP addresses
3. WHEN getting location from IP, THE Test_Suite SHALL handle API failures gracefully
4. THE Test_Suite SHALL test edge cases including invalid IPs and API timeouts

### Requirement 11: Services Testing - Parent Link Service

**User Story:** As a developer, I want comprehensive tests for parent link service, so that parent-child account linking works correctly.

#### Acceptance Criteria

1. WHEN creating a parent link, THE Test_Suite SHALL verify the link is stored with correct expiry
2. WHEN verifying a parent link, THE Test_Suite SHALL check expiry and validity
3. WHEN accepting a parent link, THE Test_Suite SHALL create the relationship and delete the link
4. WHEN rejecting a parent link, THE Test_Suite SHALL delete the link
5. THE Test_Suite SHALL test edge cases including expired links and invalid link codes

### Requirement 12: Middleware Testing - Authentication Middleware

**User Story:** As a developer, I want comprehensive tests for authentication middleware, so that request authentication and authorization work correctly.

#### Acceptance Criteria

1. WHEN a valid access token is provided, THE Middleware SHALL attach user info to the request
2. WHEN no access token is provided, THE Middleware SHALL throw UnauthorizedError
3. WHEN an invalid access token is provided, THE Middleware SHALL throw UnauthorizedError
4. WHEN an expired access token is provided, THE Middleware SHALL throw UnauthorizedError
5. WHEN a refresh token is provided instead of access token, THE Middleware SHALL throw UnauthorizedError
6. WHEN the session is revoked, THE Middleware SHALL throw UnauthorizedError
7. WHEN the user account is deleted, THE Middleware SHALL throw UnauthorizedError
8. WHEN the user account is deactivated, THE Middleware SHALL throw UnauthorizedError
9. WHEN session activity is updated, THE Middleware SHALL update lastActivityAt asynchronously
10. THE Test_Suite SHALL test edge cases including missing sessions and database errors

### Requirement 13: Middleware Testing - Rate Limiting Middleware

**User Story:** As a developer, I want comprehensive tests for rate limiting middleware, so that API rate limits are enforced correctly.

#### Acceptance Criteria

1. WHEN requests are within rate limit, THE Middleware SHALL allow the requests
2. WHEN requests exceed rate limit, THE Middleware SHALL return 429 status code
3. WHEN rate limit window expires, THE Middleware SHALL reset the counter
4. THE Test_Suite SHALL test edge cases including concurrent requests and boundary conditions

### Requirement 14: Middleware Testing - Error Handler Middleware

**User Story:** As a developer, I want comprehensive tests for error handler middleware, so that errors are formatted and logged correctly.

#### Acceptance Criteria

1. WHEN a known error type is thrown, THE Middleware SHALL return appropriate status code and message
2. WHEN an unknown error is thrown, THE Middleware SHALL return 500 status code
3. WHEN an error occurs, THE Middleware SHALL log the error details
4. THE Test_Suite SHALL test edge cases including errors without messages and nested errors

### Requirement 15: Test Organization and Maintainability

**User Story:** As a developer, I want well-organized and maintainable tests, so that tests are easy to understand and modify.

#### Acceptance Criteria

1. THE Test_Suite SHALL organize tests in a directory structure mirroring the source code
2. THE Test_Suite SHALL use descriptive test names following the pattern "should [expected behavior] when [condition]"
3. THE Test_Suite SHALL group related tests using describe blocks
4. THE Test_Suite SHALL use beforeEach and afterEach for shared setup and teardown
5. THE Test_Suite SHALL avoid code duplication by extracting common test utilities
6. THE Test_Suite SHALL include comments explaining complex test scenarios
7. WHEN tests fail, THE Test_Suite SHALL provide clear error messages indicating what went wrong

### Requirement 16: Test Coverage and Quality

**User Story:** As a developer, I want good test coverage without sacrificing quality, so that critical bugs are caught while tests remain maintainable.

#### Acceptance Criteria

1. THE Test_Suite SHALL achieve at least 80% line coverage for utils modules
2. THE Test_Suite SHALL achieve at least 75% line coverage for services modules
3. THE Test_Suite SHALL achieve at least 70% line coverage for middleware modules
4. THE Test_Suite SHALL test both success and error paths for all critical functions
5. THE Test_Suite SHALL test edge cases and boundary conditions
6. THE Test_Suite SHALL test validation logic thoroughly
7. THE Test_Suite SHALL prioritize testing business logic over trivial code
8. WHEN coverage reports are generated, THE Test_Suite SHALL exclude test files and configuration files

### Requirement 17: Test Execution and Performance

**User Story:** As a developer, I want fast and reliable test execution, so that I can run tests frequently during development.

#### Acceptance Criteria

1. WHEN running the full test suite, THE Test_Suite SHALL complete in under 30 seconds
2. WHEN running individual test files, THE Test_Suite SHALL complete in under 5 seconds
3. THE Test_Suite SHALL run tests in parallel when possible
4. THE Test_Suite SHALL ensure tests are isolated and can run in any order
5. THE Test_Suite SHALL not leave any hanging processes or connections after completion
6. WHEN tests fail, THE Test_Suite SHALL fail fast and report the first failure clearly

### Requirement 18: Continuous Integration Support

**User Story:** As a developer, I want tests that work in CI environments, so that automated testing can catch issues before deployment.

#### Acceptance Criteria

1. THE Test_Suite SHALL run successfully in CI environments without manual intervention
2. THE Test_Suite SHALL not require external services or network access
3. THE Test_Suite SHALL use environment variables for configuration
4. THE Test_Suite SHALL generate coverage reports in CI-friendly formats
5. WHEN tests fail in CI, THE Test_Suite SHALL exit with non-zero status code
