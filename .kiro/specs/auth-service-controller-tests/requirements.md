# Requirements Document

## Introduction

This document specifies the requirements for implementing comprehensive controller/component tests for the auth-service. The auth-service is a Node.js/Express application with 15 controllers handling authentication, authorization, session management, 2FA, OAuth, and related functionality. While unit tests exist for utils, services, and middleware, there are no component-level tests that verify HTTP endpoint behavior, request/response handling, and middleware integration. This specification defines the testing requirements for controller/component tests following the pattern established in api-gateway's app.test.ts.

## Glossary

- **Test_Suite**: The complete collection of controller/component tests for the auth service
- **Vitest**: The testing framework already configured in the project
- **Supertest**: HTTP testing library for making requests to Express applications
- **Mock**: A test double that simulates external dependencies (Prisma, Redis, email service, etc.)
- **Controller**: Express route handlers that process HTTP requests and return responses
- **Component_Test**: Integration test that verifies HTTP endpoint behavior including middleware
- **Request_Validation**: Testing that endpoints properly validate input data
- **Response_Format**: Testing that endpoints return correctly structured responses
- **Middleware_Integration**: Testing that authentication, rate limiting, and error handling work correctly
- **HTTP_Status_Code**: Standard HTTP response codes (200, 201, 400, 401, 403, 404, 500, etc.)

## Requirements

### Requirement 1: Test Framework Configuration

**User Story:** As a developer, I want Vitest configured with supertest for HTTP endpoint testing, so that I can test controllers as integrated components.

#### Acceptance Criteria

1. THE Test_Suite SHALL use Vitest as the testing framework
2. THE Test_Suite SHALL use supertest for HTTP request testing
3. WHEN tests are executed, THE Test_Suite SHALL support TypeScript without compilation errors
4. WHEN tests are executed, THE Test_Suite SHALL support ES modules syntax
5. THE Test_Suite SHALL configure coverage reporting with 70% threshold for controllers

### Requirement 2: Test Application Setup

**User Story:** As a developer, I want a test application instance with mocked dependencies, so that tests run isolated without real external services.

#### Acceptance Criteria

1. THE Test_Suite SHALL create an Express application instance for testing
2. THE Test_Suite SHALL mock Prisma client for all database operations
3. THE Test_Suite SHALL mock Redis client for all cache operations
4. THE Test_Suite SHALL mock email service (Resend) for all email operations
5. THE Test_Suite SHALL mock Cloudinary for all image upload operations
6. THE Test_Suite SHALL mock external HTTP requests for location and OAuth services
7. WHEN tests run, THE Test_Suite SHALL ensure no real external service calls are made
8. WHEN each test completes, THE Test_Suite SHALL reset all mocks to prevent test pollution

### Requirement 3: Authentication Controller Tests

**User Story:** As a developer, I want comprehensive tests for authentication endpoints, so that user registration, login, and logout work correctly.

#### Acceptance Criteria

1. WHEN registering with valid data, THE Test_Suite SHALL verify 201 status and user object returned
2. WHEN registering with invalid email, THE Test_Suite SHALL verify 400 status and error message
3. WHEN registering with missing fields, THE Test_Suite SHALL verify 400 status and validation error
4. WHEN registering with duplicate email, THE Test_Suite SHALL verify 409 status and error message
5. WHEN logging in with valid credentials, THE Test_Suite SHALL verify 200 status and tokens returned
6. WHEN logging in with invalid credentials, THE Test_Suite SHALL verify 401 status and error message
7. WHEN logging in with unverified email, THE Test_Suite SHALL verify appropriate status and message
8. WHEN logging out with valid token, THE Test_Suite SHALL verify 200 status and success message
9. WHEN logging out without token, THE Test_Suite SHALL verify 401 status and error message
10. WHEN refreshing token with valid refresh token, THE Test_Suite SHALL verify 200 status and new tokens

### Requirement 4: Password Management Controller Tests

**User Story:** As a developer, I want comprehensive tests for password management endpoints, so that password reset and recovery work correctly.

#### Acceptance Criteria

1. WHEN requesting password reset with valid email, THE Test_Suite SHALL verify 200 status and OTP sent
2. WHEN requesting password reset with invalid email, THE Test_Suite SHALL verify 404 status and error message
3. WHEN verifying reset OTP with valid code, THE Test_Suite SHALL verify 200 status and success message
4. WHEN verifying reset OTP with invalid code, THE Test_Suite SHALL verify 400 status and error message
5. WHEN resetting password with valid token, THE Test_Suite SHALL verify 200 status and success message
6. WHEN resetting password with expired token, THE Test_Suite SHALL verify 401 status and error message
7. WHEN resetting password with weak password, THE Test_Suite SHALL verify 400 status and validation error

### Requirement 5: Email Verification Controller Tests

**User Story:** As a developer, I want comprehensive tests for email verification endpoints, so that email verification and OTP resend work correctly.

#### Acceptance Criteria

1. WHEN verifying email with valid OTP, THE Test_Suite SHALL verify 200 status and success message
2. WHEN verifying email with invalid OTP, THE Test_Suite SHALL verify 400 status and error message
3. WHEN verifying email with expired OTP, THE Test_Suite SHALL verify 400 status and error message
4. WHEN resending verification OTP with valid email, THE Test_Suite SHALL verify 200 status and OTP sent
5. WHEN resending verification OTP during cooldown, THE Test_Suite SHALL verify 429 status and cooldown message
6. WHEN resending verification OTP for verified email, THE Test_Suite SHALL verify 400 status and error message

### Requirement 6: Device Verification Controller Tests

**User Story:** As a developer, I want comprehensive tests for device verification endpoints, so that device trust and verification work correctly.

#### Acceptance Criteria

1. WHEN verifying device with valid OTP, THE Test_Suite SHALL verify 200 status and device trusted
2. WHEN verifying device with invalid OTP, THE Test_Suite SHALL verify 400 status and error message
3. WHEN resending device verification OTP, THE Test_Suite SHALL verify 200 status and OTP sent
4. WHEN verifying device from new location, THE Test_Suite SHALL verify location tracking works

### Requirement 7: OAuth Controller Tests

**User Story:** As a developer, I want comprehensive tests for OAuth endpoints, so that Google authentication works correctly.

#### Acceptance Criteria

1. WHEN authenticating with valid Google ID token, THE Test_Suite SHALL verify 200 status and tokens returned
2. WHEN authenticating with invalid Google ID token, THE Test_Suite SHALL verify 401 status and error message
3. WHEN authenticating with expired Google ID token, THE Test_Suite SHALL verify 401 status and error message
4. WHEN authenticating with Google for new user, THE Test_Suite SHALL verify user creation and tokens returned
5. WHEN authenticating with Google for existing user, THE Test_Suite SHALL verify login and tokens returned

### Requirement 8: Two-Factor Authentication Controller Tests

**User Story:** As a developer, I want comprehensive tests for 2FA endpoints, so that two-factor authentication setup and verification work correctly.

#### Acceptance Criteria

1. WHEN enabling 2FA, THE Test_Suite SHALL verify 200 status and QR code returned
2. WHEN verifying 2FA setup with valid TOTP, THE Test_Suite SHALL verify 200 status and backup codes returned
3. WHEN verifying 2FA setup with invalid TOTP, THE Test_Suite SHALL verify 400 status and error message
4. WHEN disabling 2FA, THE Test_Suite SHALL verify 200 status and success message
5. WHEN verifying 2FA login with valid TOTP, THE Test_Suite SHALL verify 200 status and full tokens returned
6. WHEN verifying 2FA login with invalid TOTP, THE Test_Suite SHALL verify 400 status and error message
7. WHEN verifying 2FA login with backup code, THE Test_Suite SHALL verify 200 status and code consumed
8. WHEN regenerating backup codes, THE Test_Suite SHALL verify 200 status and new codes returned
9. WHEN getting 2FA status, THE Test_Suite SHALL verify 200 status and enabled state returned

### Requirement 9: Account Management Controller Tests

**User Story:** As a developer, I want comprehensive tests for account management endpoints, so that account deactivation, deletion, and reactivation work correctly.

#### Acceptance Criteria

1. WHEN deactivating account with valid token, THE Test_Suite SHALL verify 200 status and account deactivated
2. WHEN deactivating already deactivated account, THE Test_Suite SHALL verify 400 status and error message
3. WHEN deleting account with valid password, THE Test_Suite SHALL verify 200 status and account deleted
4. WHEN deleting account with invalid password, THE Test_Suite SHALL verify 401 status and error message
5. WHEN deleting OAuth account without password, THE Test_Suite SHALL verify 200 status and account deleted
6. WHEN confirming reactivation with valid token, THE Test_Suite SHALL verify 200 status and account reactivated
7. WHEN confirming reactivation for active account, THE Test_Suite SHALL verify 400 status and error message
8. WHEN deleting profile image, THE Test_Suite SHALL verify 200 status and image removed

### Requirement 10: Session Management Controller Tests

**User Story:** As a developer, I want comprehensive tests for session management endpoints, so that session listing, revocation, and cleanup work correctly.

#### Acceptance Criteria

1. WHEN getting sessions with valid token, THE Test_Suite SHALL verify 200 status and session list returned
2. WHEN getting session by ID with valid token, THE Test_Suite SHALL verify 200 status and session details returned
3. WHEN getting session by ID for different user, THE Test_Suite SHALL verify 403 status and error message
4. WHEN revoking session by ID, THE Test_Suite SHALL verify 200 status and session revoked
5. WHEN revoking all sessions, THE Test_Suite SHALL verify 200 status and all sessions revoked except current
6. WHEN cleaning up expired sessions, THE Test_Suite SHALL verify 200 status and expired sessions removed

### Requirement 11: Activity Tracking Controller Tests

**User Story:** As a developer, I want comprehensive tests for activity tracking endpoints, so that user activity logging and retrieval work correctly.

#### Acceptance Criteria

1. WHEN getting activity with valid token, THE Test_Suite SHALL verify 200 status and activity list returned
2. WHEN getting activity without token, THE Test_Suite SHALL verify 401 status and error message
3. WHEN getting activity with pagination, THE Test_Suite SHALL verify correct page returned

### Requirement 12: Profile Management Controller Tests

**User Story:** As a developer, I want comprehensive tests for profile management endpoints, so that profile updates and retrieval work correctly.

#### Acceptance Criteria

1. WHEN getting profile with valid token, THE Test_Suite SHALL verify 200 status and profile data returned
2. WHEN updating profile with valid data, THE Test_Suite SHALL verify 200 status and profile updated
3. WHEN updating profile with invalid data, THE Test_Suite SHALL verify 400 status and validation error
4. WHEN uploading profile image, THE Test_Suite SHALL verify 200 status and image URL returned
5. WHEN uploading invalid image format, THE Test_Suite SHALL verify 400 status and error message

### Requirement 13: Onboarding Controller Tests

**User Story:** As a developer, I want comprehensive tests for onboarding endpoints, so that user onboarding flow works correctly.

#### Acceptance Criteria

1. WHEN completing onboarding with valid data, THE Test_Suite SHALL verify 200 status and onboarding completed
2. WHEN completing onboarding with missing fields, THE Test_Suite SHALL verify 400 status and validation error
3. WHEN completing onboarding for already onboarded user, THE Test_Suite SHALL verify 400 status and error message
4. WHEN getting onboarding status, THE Test_Suite SHALL verify 200 status and status returned

### Requirement 14: Parent Link Controller Tests

**User Story:** As a developer, I want comprehensive tests for parent link endpoints, so that parent-child account linking works correctly.

#### Acceptance Criteria

1. WHEN creating parent link with valid data, THE Test_Suite SHALL verify 201 status and link created
2. WHEN verifying parent link with valid code, THE Test_Suite SHALL verify 200 status and link verified
3. WHEN accepting parent link with valid code, THE Test_Suite SHALL verify 200 status and relationship created
4. WHEN rejecting parent link with valid code, THE Test_Suite SHALL verify 200 status and link deleted
5. WHEN verifying expired parent link, THE Test_Suite SHALL verify 400 status and error message

### Requirement 15: Location Tracking Controller Tests

**User Story:** As a developer, I want comprehensive tests for location tracking endpoints, so that location updates and retrieval work correctly.

#### Acceptance Criteria

1. WHEN updating location with valid coordinates, THE Test_Suite SHALL verify 200 status and location updated
2. WHEN updating location with invalid coordinates, THE Test_Suite SHALL verify 400 status and validation error
3. WHEN getting location history, THE Test_Suite SHALL verify 200 status and location list returned

### Requirement 16: Internal Service Controller Tests

**User Story:** As a developer, I want comprehensive tests for internal service endpoints, so that service-to-service communication works correctly.

#### Acceptance Criteria

1. WHEN validating token with valid token, THE Test_Suite SHALL verify 200 status and user data returned
2. WHEN validating token with invalid token, THE Test_Suite SHALL verify 401 status and error message
3. WHEN getting user by ID with valid ID, THE Test_Suite SHALL verify 200 status and user data returned
4. WHEN getting user by ID with invalid ID, THE Test_Suite SHALL verify 404 status and error message

### Requirement 17: Authentication Middleware Integration Tests

**User Story:** As a developer, I want tests that verify authentication middleware integration, so that protected endpoints enforce authentication correctly.

#### Acceptance Criteria

1. WHEN accessing protected endpoint with valid token, THE Test_Suite SHALL verify request succeeds
2. WHEN accessing protected endpoint without token, THE Test_Suite SHALL verify 401 status returned
3. WHEN accessing protected endpoint with invalid token, THE Test_Suite SHALL verify 401 status returned
4. WHEN accessing protected endpoint with expired token, THE Test_Suite SHALL verify 401 status returned
5. WHEN accessing protected endpoint with revoked session, THE Test_Suite SHALL verify 401 status returned

### Requirement 18: Rate Limiting Middleware Integration Tests

**User Story:** As a developer, I want tests that verify rate limiting middleware integration, so that rate limits are enforced correctly.

#### Acceptance Criteria

1. WHEN making requests within rate limit, THE Test_Suite SHALL verify requests succeed
2. WHEN exceeding login rate limit, THE Test_Suite SHALL verify 429 status returned
3. WHEN exceeding registration rate limit, THE Test_Suite SHALL verify 429 status returned
4. WHEN exceeding OTP verification rate limit, THE Test_Suite SHALL verify 429 status returned
5. WHEN rate limit window expires, THE Test_Suite SHALL verify requests succeed again

### Requirement 19: Error Handling Integration Tests

**User Story:** As a developer, I want tests that verify error handling middleware integration, so that errors are formatted consistently.

#### Acceptance Criteria

1. WHEN a validation error occurs, THE Test_Suite SHALL verify 400 status and error format
2. WHEN an authentication error occurs, THE Test_Suite SHALL verify 401 status and error format
3. WHEN an authorization error occurs, THE Test_Suite SHALL verify 403 status and error format
4. WHEN a not found error occurs, THE Test_Suite SHALL verify 404 status and error format
5. WHEN an internal error occurs, THE Test_Suite SHALL verify 500 status and error format
6. WHEN any error occurs, THE Test_Suite SHALL verify response includes error, statusCode, and timestamp

### Requirement 20: Request Validation Tests

**User Story:** As a developer, I want tests that verify request validation, so that invalid input is rejected with clear error messages.

#### Acceptance Criteria

1. WHEN sending request with missing required fields, THE Test_Suite SHALL verify 400 status and field errors
2. WHEN sending request with invalid email format, THE Test_Suite SHALL verify 400 status and validation error
3. WHEN sending request with invalid data types, THE Test_Suite SHALL verify 400 status and type error
4. WHEN sending request with out-of-range values, THE Test_Suite SHALL verify 400 status and range error
5. WHEN sending request with malformed JSON, THE Test_Suite SHALL verify 400 status and parse error

### Requirement 21: Response Format Tests

**User Story:** As a developer, I want tests that verify response formats, so that API responses are consistent and well-structured.

#### Acceptance Criteria

1. WHEN a successful request completes, THE Test_Suite SHALL verify response includes expected data fields
2. WHEN an error occurs, THE Test_Suite SHALL verify response includes error, statusCode, and timestamp
3. WHEN returning lists, THE Test_Suite SHALL verify response includes array and metadata
4. WHEN returning tokens, THE Test_Suite SHALL verify response includes accessToken and refreshToken
5. WHEN returning user data, THE Test_Suite SHALL verify sensitive fields are excluded

### Requirement 22: Test Organization and Maintainability

**User Story:** As a developer, I want well-organized and maintainable tests, so that tests are easy to understand and modify.

#### Acceptance Criteria

1. THE Test_Suite SHALL organize tests by controller in separate test files
2. THE Test_Suite SHALL use descriptive test names following the pattern "should [expected behavior] when [condition]"
3. THE Test_Suite SHALL group related tests using describe blocks by endpoint
4. THE Test_Suite SHALL use beforeEach and afterEach for shared setup and teardown
5. THE Test_Suite SHALL extract common test utilities and mock factories
6. THE Test_Suite SHALL include comments explaining complex test scenarios
7. WHEN tests fail, THE Test_Suite SHALL provide clear error messages indicating what went wrong

### Requirement 23: Test Coverage and Quality

**User Story:** As a developer, I want good test coverage for controllers, so that critical endpoint bugs are caught while tests remain maintainable.

#### Acceptance Criteria

1. THE Test_Suite SHALL achieve at least 70% line coverage for all controllers
2. THE Test_Suite SHALL test both success and error paths for all endpoints
3. THE Test_Suite SHALL test authentication and authorization for protected endpoints
4. THE Test_Suite SHALL test request validation for all input parameters
5. THE Test_Suite SHALL test response formats for all endpoints
6. THE Test_Suite SHALL test middleware integration for all routes
7. WHEN coverage reports are generated, THE Test_Suite SHALL exclude test files and configuration files

### Requirement 24: Test Execution and Performance

**User Story:** As a developer, I want fast and reliable test execution, so that I can run tests frequently during development.

#### Acceptance Criteria

1. WHEN running the full controller test suite, THE Test_Suite SHALL complete in under 60 seconds
2. WHEN running individual controller test files, THE Test_Suite SHALL complete in under 10 seconds
3. THE Test_Suite SHALL run tests in parallel when possible
4. THE Test_Suite SHALL ensure tests are isolated and can run in any order
5. THE Test_Suite SHALL not leave any hanging processes or connections after completion
6. WHEN tests fail, THE Test_Suite SHALL fail fast and report the first failure clearly

### Requirement 25: Continuous Integration Support

**User Story:** As a developer, I want tests that work in CI environments, so that automated testing can catch issues before deployment.

#### Acceptance Criteria

1. THE Test_Suite SHALL run successfully in CI environments without manual intervention
2. THE Test_Suite SHALL not require external services or network access
3. THE Test_Suite SHALL use environment variables for configuration
4. THE Test_Suite SHALL generate coverage reports in CI-friendly formats
5. WHEN tests fail in CI, THE Test_Suite SHALL exit with non-zero status code
