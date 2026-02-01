# Implementation Plan: Auth Service Unit Tests

## Overview

This implementation plan breaks down the creation of comprehensive unit tests for the auth service into discrete, manageable tasks. The approach prioritizes setting up the test infrastructure first, then implementing tests for utils (highest priority), followed by services and middleware. Each task builds on previous work, with checkpoints to ensure quality and catch issues early.

## Tasks

- [ ] 1. Set up test infrastructure and configuration
  - Remove Vitest configuration and dependencies
  - Install Jest and related dependencies: `npm uninstall vitest @vitest/coverage-v8 && npm install --save-dev jest @types/jest ts-jest @jest/globals`
  - Create jest.config.cjs with TypeScript, ES modules, and coverage configuration
  - Create tests/setup/jest.setup.ts for global test setup
  - Configure coverage thresholds (utils: 80%, services: 75%, middleware: 70%)
  - Update test scripts in package.json to use Jest
  - Ensure fast-check is installed for property-based testing
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 16.8_

- [ ] 2. Create mock infrastructure
  - [ ] 2.1 Create Prisma mock (tests/mocks/prisma.mock.ts)
    - Mock all Prisma client models (user, session, userDevice, parentChildLink, unlinkRequest, oAuthAccount)
    - Provide mock implementations for CRUD operations (findUnique, findFirst, findMany, create, update, delete, updateMany, deleteMany)
    - Export mockPrisma object and setup/teardown functions
    - _Requirements: 2.1_

  - [ ] 2.2 Create Redis mock (tests/mocks/redis.mock.ts)
    - Implement RedisMock class with in-memory storage
    - Mock all Redis operations (get, set, del, ttl, incr, expire, sadd, smembers, srem)
    - Implement pipeline support for batched operations
    - Export RedisMock class and factory function
    - _Requirements: 2.2_

  - [ ] 2.3 Create email service mock (tests/mocks/email.mock.ts)
    - Mock Resend email service
    - Track sent emails for verification
    - Export mockEmailService and helper functions
    - _Requirements: 2.3_

  - [ ] 2.4 Create Cloudinary mock (tests/mocks/cloudinary.mock.ts)
    - Mock Cloudinary upload operations
    - Return mock upload responses
    - Export mockCloudinary object
    - _Requirements: 2.4_

  - [ ] 2.5 Create HTTP fetch mock (tests/mocks/fetch.mock.ts)
    - Mock global fetch for location API calls
    - Support configurable responses for different URLs
    - Export mockFetch and helper functions
    - _Requirements: 2.5_

- [ ] 3. Create test utilities and factories
  - Create tests/utils/testHelpers.ts with common test utilities
  - Create tests/factories/userFactory.ts for generating test users
  - Create tests/factories/sessionFactory.ts for generating test sessions
  - Create tests/factories/deviceFactory.ts for generating test devices
  - Add helper functions: createMockRequest, createMockResponse, createMockNext, advanceTime
  - _Requirements: 15.5_

- [ ] 4. Implement utils tests - Token Management (tests/unit/utils/tokens.test.ts)
  - [ ] 4.1 Test access token generation
    - Write unit test: should generate valid JWT with correct payload structure
    - Write property test for Property 1: Access token generation produces valid JWT structure
    - _Requirements: 3.1, Property 1_

  - [ ] 4.2 Test refresh token generation and storage
    - Write unit test: should generate refresh token and store in Redis with TTL
    - Write property test for Property 2: Refresh token storage includes Redis persistence
    - _Requirements: 3.2, Property 2_

  - [ ] 4.3 Test access token verification
    - Write unit test: should return decoded payload for valid access token
    - Write property test for Property 3: Valid access token verification returns payload
    - _Requirements: 3.3, Property 3_

  - [ ] 4.4 Test invalid token verification
    - Write unit tests: should throw error for invalid, malformed, and tampered tokens
    - Write property test for Property 4: Invalid token verification throws appropriate errors
    - _Requirements: 3.4, 3.5, Property 4_

  - [ ] 4.5 Test refresh token verification
    - Write unit test: should check Redis for token existence
    - Write unit test: should throw error for revoked refresh token
    - Write property test for Property 5: Refresh token verification checks Redis existence
    - _Requirements: 3.6, 3.7, Property 5_

  - [ ] 4.6 Test token rotation
    - Write unit test: should revoke old token and create new one
    - Write property test for Property 6: Token rotation revokes old and creates new
    - _Requirements: 3.8, Property 6_

  - [ ] 4.7 Test bulk token revocation
    - Write unit test: should delete all user refresh tokens from Redis
    - Write property test for Property 7: Bulk token revocation deletes all user tokens
    - _Requirements: 3.9, Property 7_

  - [ ] 4.8 Test edge cases
    - Write unit tests for malformed tokens, missing secrets, expired tokens
    - _Requirements: 3.10_

- [ ] 5. Implement utils tests - Session Management (tests/unit/utils/sessions.test.ts)
  - [ ] 5.1 Test session creation
    - Write unit test: should store session with correct fields
    - Write property test for Property 8: Session creation stores complete session data
    - _Requirements: 4.1, Property 8_

  - [ ] 5.2 Test session activity update
    - Write unit test: should update lastActivityAt timestamp
    - Write property test for Property 9: Session activity update modifies timestamp
    - _Requirements: 4.2, Property 9_

  - [ ] 5.3 Test session revocation
    - Write unit test: should delete session and revoke refresh token
    - Write property test for Property 10: Session revocation deletes session and refresh token
    - _Requirements: 4.3, Property 10_

  - [ ] 5.4 Test bulk session revocation
    - Write unit test: should delete all sessions except current when specified
    - Write property test for Property 11: Bulk session revocation respects current session exclusion
    - _Requirements: 4.4, Property 11_

  - [ ] 5.5 Test session details parsing
    - Write unit test: should parse device info and status correctly
    - Write property test for Property 12: Session details parsing extracts device information
    - _Requirements: 4.5, 4.6, Property 12_

  - [ ] 5.6 Test expired session cleanup
    - Write unit test: should delete only expired sessions
    - Write property test for Property 13: Expired session cleanup deletes only expired sessions
    - _Requirements: 4.7, Property 13_

  - [ ] 5.7 Test edge cases
    - Write unit tests for missing device info, invalid session IDs
    - _Requirements: 4.8_

- [ ] 6. Implement utils tests - OTP Management (tests/unit/utils/otp.test.ts)
  - [ ] 6.1 Test OTP generation
    - Write unit test: should generate numeric OTP with correct length
    - Write property test for Property 14: OTP generation produces numeric code with correct length
    - _Requirements: 5.1, Property 14_

  - [ ] 6.2 Test OTP storage
    - Write unit test: should store OTP in Redis with TTL
    - Write property test for Property 15: OTP storage includes Redis persistence with TTL
    - _Requirements: 5.2, Property 15_

  - [ ] 6.3 Test correct OTP verification
    - Write unit test: should return true and delete OTP
    - Write property test for Property 16: Correct OTP verification consumes the OTP
    - _Requirements: 5.3, Property 16_

  - [ ] 6.4 Test incorrect OTP verification
    - Write unit test: should increment attempt counter
    - Write property test for Property 17: Incorrect OTP verification increments attempts
    - _Requirements: 5.4, Property 17_

  - [ ] 6.5 Test OTP attempt limit and cooldown
    - Write unit test: should set cooldown when attempts exceed limit
    - Write property test for Property 18: OTP attempt limit triggers cooldown
    - _Requirements: 5.5, 5.6, Property 18_

  - [ ] 6.6 Test non-consuming OTP verification
    - Write unit test: should not delete OTP on success
    - Write property test for Property 19: Non-consuming OTP verification preserves OTP
    - _Requirements: 5.7, Property 19_

  - [ ] 6.7 Test edge cases
    - Write unit tests for expired OTPs, missing OTPs
    - _Requirements: 5.8_

- [ ] 7. Implement utils tests - Two-Factor Authentication (tests/unit/utils/twoFactor.test.ts)
  - [ ] 7.1 Test 2FA secret generation
    - Write unit test: should generate base32 encoded secret
    - Write property test for Property 20: 2FA secret generation produces base32 encoded value
    - _Requirements: 6.1, Property 20_

  - [ ] 7.2 Test QR code generation
    - Write unit test: should generate valid QR code data URL
    - Write property test for Property 21: QR code generation produces valid data URL
    - _Requirements: 6.2, Property 21_

  - [ ] 7.3 Test TOTP token verification
    - Write unit test: should return true for valid TOTP token
    - Write unit test: should return false for invalid TOTP token
    - Write property test for Property 22: Valid TOTP token verification succeeds
    - Write property test for Property 23: Invalid TOTP token verification fails
    - _Requirements: 6.3, 6.4, Property 22, Property 23_

  - [ ] 7.4 Test secret encryption and decryption
    - Write unit test: should encrypt and decrypt secret correctly
    - Write property test for Property 24: Secret encryption round-trip preserves value
    - _Requirements: 6.5, 6.6, Property 24_

  - [ ] 7.5 Test backup code generation
    - Write unit test: should generate correct count and format
    - Write property test for Property 25: Backup code generation produces correct count and format
    - _Requirements: 6.7, Property 25_

  - [ ] 7.6 Test backup code verification
    - Write unit test: should remove valid code from list
    - Write unit test: should preserve list for invalid code
    - Write property test for Property 26: Valid backup code verification removes code from list
    - Write property test for Property 27: Invalid backup code verification preserves list
    - _Requirements: 6.8, 6.9, Property 26, Property 27_

  - [ ] 7.7 Test edge cases
    - Write unit tests for missing encryption keys, malformed codes
    - _Requirements: 6.10_

- [ ] 8. Implement utils tests - Email Verification (tests/unit/utils/emailVerification.test.ts)
  - [ ] 8.1 Test email verification cooldown check
    - Write unit test: should return remaining seconds
    - Write property test for Property 28: Email verification cooldown returns remaining time
    - _Requirements: 7.1, Property 28_

  - [ ] 8.2 Test email verification cooldown setting
    - Write unit test: should apply progressive cooldown
    - Write property test for Property 29: Email verification cooldown applies progressive duration
    - _Requirements: 7.2, Property 29_

  - [ ] 8.3 Test email verification allowed check
    - Write unit test: should consider cooldown and attempts
    - Write property test for Property 30: Email verification allowed check considers multiple factors
    - _Requirements: 7.3, Property 30_

  - [ ] 8.4 Test email verification cooldown clear
    - Write unit test: should reset cooldown and attempts
    - Write property test for Property 31: Email verification cooldown clear resets all state
    - _Requirements: 7.4, Property 31_

  - [ ] 8.5 Test resend OTP cooldown
    - Write unit test: should enforce rate limiting
    - Write property test for Property 32: Resend OTP cooldown enforces rate limiting
    - _Requirements: 7.5, 7.6, Property 32_

  - [ ] 8.6 Test edge cases
    - Write unit tests for expired cooldowns, boundary attempt counts
    - _Requirements: 7.7_

- [ ] 9. Implement utils tests - Password Reset (tests/unit/utils/passwordReset.test.ts)
  - [ ] 9.1 Test password reset token generation
    - Write unit test: should store token in Redis
    - Write property test for Property 33: Password reset token generation stores in Redis
    - _Requirements: 8.1, Property 33_

  - [ ] 9.2 Test password reset token verification
    - Write unit test: should return user ID for valid token
    - Write property test for Property 34: Valid password reset token verification returns user ID
    - _Requirements: 8.2, Property 34_

  - [ ] 9.3 Test invalid password reset token verification
    - Write unit test: should throw error for expired/invalid token
    - Write property test for Property 35: Invalid password reset token verification throws error
    - _Requirements: 8.3, 8.4, Property 35_

  - [ ] 9.4 Test password reset token consumption
    - Write unit test: should delete token from Redis
    - Write property test for Property 36: Password reset token consumption deletes from Redis
    - _Requirements: 8.5, Property 36_

  - [ ] 9.5 Test edge cases
    - Write unit tests for missing tokens, malformed tokens
    - _Requirements: 8.6_

- [ ] 10. Implement utils tests - Additional Utils
  - [ ] 10.1 Test cookies utility (tests/unit/utils/cookies.test.ts)
    - Write unit tests for cookie setting, clearing, and parsing
    - Test secure cookie options and httpOnly flags
    - _Requirements: 15.1, 15.2, 15.3_

  - [ ] 10.2 Test device utility (tests/unit/utils/device.test.ts)
    - Write unit tests for device fingerprint generation
    - Test user agent parsing
    - Test device name extraction
    - _Requirements: 15.1, 15.2, 15.3_

  - [ ] 10.3 Test errors utility (tests/unit/utils/errors.test.ts)
    - Write unit tests for custom error classes
    - Test error message formatting
    - Test error status codes
    - _Requirements: 15.1, 15.2, 15.3_

- [ ] 11. Implement services tests - Auth Session Service (tests/unit/services/authSession.service.test.ts)
  - [ ] 11.1 Test session expiry calculation
    - Write unit test: should calculate correct expiry dates
    - Write property test for Property 37: Session expiry calculation uses environment variables
    - _Requirements: 9.1, Property 37_

  - [ ] 11.2 Test token generation
    - Write unit test: should create both access and refresh tokens
    - Write property test for Property 38: Token generation creates both access and refresh tokens
    - _Requirements: 9.2, Property 38_

  - [ ] 11.3 Test device lookup
    - Write unit test: should reuse existing devices by fingerprint
    - Write property test for Property 39: Device lookup reuses existing devices by fingerprint
    - _Requirements: 9.3, Property 39_

  - [ ] 11.4 Test new device creation
    - Write unit test: should store complete device information
    - Write property test for Property 40: New device creation stores complete device information
    - _Requirements: 9.4, Property 40_

  - [ ] 11.5 Test complete session creation
    - Write unit test: should coordinate all operations
    - Write property test for Property 41: Complete session creation coordinates all operations
    - _Requirements: 9.5, Property 41_

  - [ ] 11.6 Test temporary 2FA session
    - Write unit test: should create session without refresh token
    - Write property test for Property 42: Temporary 2FA session excludes refresh token
    - _Requirements: 9.6, Property 42_

  - [ ] 11.7 Test edge cases
    - Write unit tests for missing device info, duplicate devices
    - _Requirements: 9.7_

- [ ] 12. Implement services tests - Location Service (tests/unit/services/location.service.test.ts)
  - [ ] 12.1 Test session location update
    - Write unit test: should update location field
    - Write property test for Property 43: Session location update modifies database field
    - _Requirements: 10.1, Property 43_

  - [ ] 12.2 Test location from IP
    - Write unit test: should handle private IP addresses
    - Write unit test: should handle API failures gracefully
    - Write property test for Property 44: Location API failure handling returns null gracefully
    - _Requirements: 10.2, 10.3, Property 44_

  - [ ] 12.3 Test edge cases
    - Write unit tests for invalid IPs, API timeouts
    - _Requirements: 10.4_

- [ ] 13. Implement services tests - Parent Link Service (tests/unit/services/parentLink.service.test.ts)
  - [ ] 13.1 Test parent link creation
    - Write unit test: should store link with expiry
    - Write property test for Property 45: Parent link creation stores with expiry
    - _Requirements: 11.1, Property 45_

  - [ ] 13.2 Test parent link verification
    - Write unit test: should check expiry and validity
    - Write property test for Property 46: Parent link verification checks expiry and validity
    - _Requirements: 11.2, Property 46_

  - [ ] 13.3 Test parent link acceptance
    - Write unit test: should create relationship and delete link
    - Write property test for Property 47: Parent link acceptance creates relationship and deletes link
    - _Requirements: 11.3, Property 47_

  - [ ] 13.4 Test parent link rejection
    - Write unit test: should delete link
    - Write property test for Property 48: Parent link rejection deletes link
    - _Requirements: 11.4, Property 48_

  - [ ] 13.5 Test edge cases
    - Write unit tests for expired links, invalid link codes
    - _Requirements: 11.5_

- [ ] 14. Implement middleware tests - Authentication Middleware (tests/unit/middleware/auth.middleware.test.ts)
  - [ ] 14.1 Test valid token authentication
    - Write unit test: should attach user info to request
    - Write property test for Property 49: Valid token authentication attaches user info
    - _Requirements: 12.1, Property 49_

  - [ ] 14.2 Test missing token authentication
    - Write unit test: should throw UnauthorizedError
    - _Requirements: 12.2_

  - [ ] 14.3 Test invalid token authentication
    - Write unit test: should throw UnauthorizedError for invalid/expired tokens
    - Write property test for Property 50: Invalid token authentication throws UnauthorizedError
    - _Requirements: 12.3, 12.4, 12.5, Property 50_

  - [ ] 14.4 Test revoked session authentication
    - Write unit test: should throw UnauthorizedError
    - Write property test for Property 51: Revoked session authentication throws UnauthorizedError
    - _Requirements: 12.6, Property 51_

  - [ ] 14.5 Test deleted/deactivated user authentication
    - Write unit test: should throw UnauthorizedError
    - _Requirements: 12.7, 12.8_

  - [ ] 14.6 Test session activity update
    - Write unit test: should update lastActivityAt asynchronously
    - Write property test for Property 52: Authenticated request updates session activity
    - _Requirements: 12.9, Property 52_

  - [ ] 14.7 Test edge cases
    - Write unit tests for missing sessions, database errors
    - _Requirements: 12.10_

- [ ] 15. Implement middleware tests - Rate Limiting Middleware (tests/unit/middleware/rateLimiter.middleware.test.ts)
  - [ ] 15.1 Test requests within limit
    - Write unit test: should allow requests
    - Write property test for Property 53: Requests within limit are allowed
    - _Requirements: 13.1, Property 53_

  - [ ] 15.2 Test requests exceeding limit
    - Write unit test: should return 429 status code
    - Write property test for Property 54: Requests exceeding limit return 429
    - _Requirements: 13.2, Property 54_

  - [ ] 15.3 Test rate limit window expiry
    - Write unit test: should reset counter
    - Write property test for Property 55: Rate limit window expiry resets counter
    - _Requirements: 13.3, Property 55_

  - [ ] 15.4 Test edge cases
    - Write unit tests for concurrent requests, boundary conditions
    - _Requirements: 13.4_

- [ ] 16. Implement middleware tests - Error Handler Middleware (tests/unit/middleware/errorHandler.test.ts)
  - [ ] 16.1 Test known error types
    - Write unit test: should return appropriate status codes
    - Write property test for Property 56: Known error types return appropriate status codes
    - _Requirements: 14.1, Property 56_

  - [ ] 16.2 Test unknown errors
    - Write unit test: should return 500 status code
    - Write property test for Property 57: Unknown errors return 500 status code
    - _Requirements: 14.2, Property 57_

  - [ ] 16.3 Test error logging
    - Write unit test: should log error details
    - Write property test for Property 58: Error occurrence triggers logging
    - _Requirements: 14.3, Property 58_

  - [ ] 16.4 Test edge cases
    - Write unit tests for errors without messages, nested errors
    - _Requirements: 14.4_

- [ ] 17. Verify test infrastructure properties
  - [ ] 17.1 Verify no real external calls
    - Write property test for Property 59: Test execution makes no real external calls
    - _Requirements: 2.6, Property 59_

  - [ ] 17.2 Verify mock cleanup
    - Write property test for Property 60: Test completion cleans up mocks
    - _Requirements: 2.7, Property 60_

  - [ ] 17.3 Verify clear error messages
    - Write property test for Property 61: Test failure provides clear error messages
    - _Requirements: 15.7, Property 61_

  - [ ] 17.4 Verify test isolation
    - Write property test for Property 62: Tests run in any order with consistent results
    - _Requirements: 17.4, Property 62_

  - [ ] 17.5 Verify no hanging resources
    - Write property test for Property 63: Test completion leaves no hanging resources
    - _Requirements: 17.5, Property 63_

  - [ ] 17.6 Verify no external services required
    - Write property test for Property 64: Tests require no external services
    - _Requirements: 18.2, Property 64_

- [ ] 18. Run full test suite and verify coverage
  - Run `npm test` to execute all tests
  - Run `npm run test:coverage` to generate coverage report
  - Verify coverage thresholds are met (utils: 80%, services: 75%, middleware: 70%)
  - Review coverage report and identify any gaps
  - Add additional tests if needed to meet coverage targets
  - _Requirements: 16.1, 16.2, 16.3, 17.1, 17.2, 17.3_

- [ ] 19. Optimize test performance
  - Review test execution time
  - Ensure full test suite completes in under 30 seconds
  - Ensure individual test files complete in under 5 seconds
  - Optimize slow tests if needed
  - _Requirements: 17.1, 17.2_

- [ ] 20. Final verification and documentation
  - Verify all tests pass consistently
  - Verify tests work in CI environment
  - Document any special test setup requirements
  - Create README for test suite if needed
  - _Requirements: 18.1, 18.3, 18.4, 18.5_
