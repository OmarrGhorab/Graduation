# Implementation Plan: Auth Service Controller Tests

## Overview

This implementation plan breaks down the creation of comprehensive controller/component tests for the auth-service into discrete, manageable tasks. The tests will use Vitest and supertest to verify HTTP endpoint behavior, middleware integration, and API contracts for all 15 controllers.

## Tasks

- [x] 1. Set up test infrastructure and helpers
  - [x] 1.1 Install supertest dependency
    - Add supertest to package.json devDependencies
    - Run npm install
    - _Requirements: 1.2_
  
  - [x] 1.2 Create test app factory
    - Create `tests/helpers/testApp.ts`
    - Implement `createTestApp()` function that creates Express app with routes
    - Support options for skipping auth/rate limiting
    - _Requirements: 2.1, 2.8_
  
  - [x] 1.3 Create mock factories
    - Create `tests/helpers/mocks.ts`
    - Implement `mockPrisma()` factory for database mocks
    - Implement `mockRedis()` factory for cache mocks
    - Implement `mockResend()` factory for email mocks
    - Implement `mockCloudinary()` factory for image upload mocks
    - _Requirements: 2.2, 2.3, 2.4, 2.5_
  
  - [x] 1.4 Create test fixtures
    - Create `tests/helpers/fixtures.ts`
    - Implement user fixtures (verified, unverified, 2FA-enabled)
    - Implement token fixtures (valid, expired, invalid)
    - Implement session fixtures
    - _Requirements: 22.5_

- [x] 2. Implement authentication controller tests
  - [x] 2.1 Create auth controller test file
    - Create `tests/controllers/auth.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 2.2 Test registration endpoints
    - Test POST /api/v1/auth/register with valid data
    - Test registration with invalid email
    - Test registration with missing fields
    - Test registration with duplicate email
    - _Requirements: 3.1, 3.2, 3.3, 3.4_
  
  - [x] 2.3 Write property test for valid registration
    - **Property 4: Valid registration succeeds**
    - **Validates: Requirements 3.1**
  
  - [x] 2.4 Write property test for invalid email rejection
    - **Property 5: Invalid email format rejected**
    - **Validates: Requirements 3.2, 20.2**
  
  - [x] 2.5 Write property test for missing fields rejection
    - **Property 6: Missing required fields rejected**
    - **Validates: Requirements 3.3, 20.1**
  
  - [x] 2.6 Test login endpoints
    - Test POST /api/v1/auth/login with valid credentials
    - Test login with invalid credentials
    - Test login with unverified email
    - _Requirements: 3.5, 3.6, 3.7_
  
  - [x] 2.7 Write property test for valid login
    - **Property 9: Valid login succeeds**
    - **Validates: Requirements 3.5**
  
  - [x] 2.8 Write property test for invalid credentials
    - **Property 10: Invalid credentials rejected**
    - **Validates: Requirements 3.6**
  
  - [x] 2.9 Test logout and refresh endpoints
    - Test POST /api/v1/auth/logout with valid token
    - Test logout without token
    - Test POST /api/v1/auth/refresh with valid refresh token
    - _Requirements: 3.8, 3.9, 3.10_
  
  - [x] 2.10 Write property test for valid logout
    - **Property 11: Valid logout succeeds**
    - **Validates: Requirements 3.8**
  
  - [x] 2.11 Write property test for token refresh
    - **Property 12: Valid token refresh succeeds**
    - **Validates: Requirements 3.10**

- [x] 3. Implement password management controller tests
  - [x] 3.1 Create password controller test file
    - Create `tests/controllers/password.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 3.2 Test forgot password endpoint
    - Test POST /api/v1/auth/forgot-password with valid email
    - Test forgot password with invalid email
    - _Requirements: 4.1, 4.2_
  
  - [x] 3.3 Write property test for password reset request
    - **Property 13: Valid password reset request succeeds**
    - **Validates: Requirements 4.1**
  
  - [x] 3.4 Write property test for non-existent email
    - **Property 14: Non-existent email returns not found**
    - **Validates: Requirements 4.2**
  
  - [x] 3.5 Test OTP verification and password reset
    - Test POST /api/v1/auth/verify-reset-otp with valid OTP
    - Test verify OTP with invalid OTP
    - Test POST /api/v1/auth/reset-password with valid token
    - Test reset password with expired token
    - Test reset password with weak password
    - _Requirements: 4.3, 4.4, 4.5, 4.6, 4.7_
  
  - [x] 3.6 Write property test for OTP verification
    - **Property 15: Valid OTP verification succeeds**
    - **Validates: Requirements 4.3**
  
  - [x] 3.7 Write property test for invalid OTP
    - **Property 16: Invalid OTP rejected**
    - **Validates: Requirements 4.4**
  
  - [x] 3.8 Write property test for password reset
    - **Property 17: Valid password reset succeeds**
    - **Validates: Requirements 4.5**
  
  - [x] 3.9 Write property test for weak password
    - **Property 18: Weak password rejected**
    - **Validates: Requirements 4.7**

- [x] 4. Implement email verification controller tests
  - [x] 4.1 Create email verification controller test file
    - Create `tests/controllers/email-verification.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 4.2 Test email verification endpoints
    - Test POST /api/v1/auth/verify-email-otp with valid OTP
    - Test verify email with invalid OTP
    - Test verify email with expired OTP
    - Test POST /api/v1/auth/resend-verification-otp with valid email
    - Test resend OTP during cooldown
    - Test resend OTP for verified email
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
  
  - [x] 4.3 Write property test for email verification
    - **Property 19: Valid email verification succeeds**
    - **Validates: Requirements 5.1**
  
  - [x] 4.4 Write property test for invalid email OTP
    - **Property 20: Invalid email OTP rejected**
    - **Validates: Requirements 5.2**
  
  - [x] 4.5 Write property test for OTP resend
    - **Property 21: Valid OTP resend succeeds**
    - **Validates: Requirements 5.4**

- [x] 5. Implement device verification controller tests
  - [x] 5.1 Create device controller test file
    - Create `tests/controllers/device.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 5.2 Test device verification endpoints
    - Test POST /api/v1/auth/verify-device with valid OTP
    - Test verify device with invalid OTP
    - Test POST /api/v1/auth/resend-device-verification-otp
    - Test device verification from new location
    - _Requirements: 6.1, 6.2, 6.3, 6.4_
  
  - [x] 5.3 Write property test for device verification
    - **Property 22: Valid device verification succeeds**
    - **Validates: Requirements 6.1**
  
  - [x] 5.4 Write property test for invalid device OTP
    - **Property 23: Invalid device OTP rejected**
    - **Validates: Requirements 6.2**

- [x] 6. Implement OAuth controller tests
  - [x] 6.1 Create OAuth controller test file
    - Create `tests/controllers/oauth.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - Mock Google OAuth token verification
    - _Requirements: 22.1, 22.4, 2.6_
  
  - [x] 6.2 Test Google authentication endpoints
    - Test POST /api/v1/auth/google/mobile with valid token
    - Test Google auth with invalid token
    - Test Google auth with expired token
    - Test Google auth for new user
    - Test Google auth for existing user
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  
  - [x] 6.3 Write property test for valid Google auth
    - **Property 24: Valid Google token authentication succeeds**
    - **Validates: Requirements 7.1**
  
  - [x] 6.4 Write property test for invalid Google token
    - **Property 25: Invalid Google token rejected**
    - **Validates: Requirements 7.2**

- [x] 7. Implement two-factor authentication controller tests
  - [x] 7.1 Create 2FA controller test file
    - Create `tests/controllers/twoFactor.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 7.2 Test 2FA setup endpoints
    - Test POST /api/v1/auth/2fa/enable
    - Test POST /api/v1/auth/2fa/verify-setup with valid TOTP
    - Test verify setup with invalid TOTP
    - Test POST /api/v1/auth/2fa/disable
    - _Requirements: 8.1, 8.2, 8.3, 8.4_
  
  - [x] 7.3 Write property test for TOTP setup
    - **Property 26: Valid TOTP setup succeeds**
    - **Validates: Requirements 8.2**
  
  - [x] 7.4 Write property test for invalid TOTP setup
    - **Property 27: Invalid TOTP setup rejected**
    - **Validates: Requirements 8.3**
  
  - [x] 7.5 Test 2FA login endpoints
    - Test POST /api/v1/auth/2fa/verify-login with valid TOTP
    - Test verify login with invalid TOTP
    - Test verify login with backup code
    - Test GET /api/v1/auth/2fa/status
    - Test POST /api/v1/auth/2fa/regenerate-backup-codes
    - _Requirements: 8.5, 8.6, 8.7, 8.8, 8.9_
  
  - [x] 7.6 Write property test for 2FA login
    - **Property 28: Valid 2FA login succeeds**
    - **Validates: Requirements 8.5**
  
  - [x] 7.7 Write property test for invalid 2FA login
    - **Property 29: Invalid 2FA login rejected**
    - **Validates: Requirements 8.6**

- [x] 8. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Implement account management controller tests
  - [x] 9.1 Create account controller test file
    - Create `tests/controllers/account.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 9.2 Test account management endpoints
    - Test POST /api/v1/auth/account/deactivate
    - Test deactivate already deactivated account
    - Test POST /api/v1/auth/account/delete with valid password
    - Test delete account with invalid password
    - Test delete OAuth account without password
    - Test POST /api/v1/auth/account/confirm-reactivation
    - Test reactivation for active account
    - Test DELETE /api/v1/auth/account/profile-image
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8_
  
  - [x] 9.3 Write property test for invalid password deletion
    - **Property 30: Invalid password for deletion rejected**
    - **Validates: Requirements 9.4**

- [x] 10. Implement session management controller tests
  - [x] 10.1 Create sessions controller test file
    - Create `tests/controllers/sessions.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 10.2 Test session management endpoints
    - Test GET /api/v1/auth/sessions
    - Test GET /api/v1/auth/sessions/:sessionId
    - Test get session for different user
    - Test DELETE /api/v1/auth/sessions/:sessionId
    - Test DELETE /api/v1/auth/sessions/all
    - Test DELETE /api/v1/auth/sessions/cleanup
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_

- [x] 11. Implement activity tracking controller tests
  - [x] 11.1 Create activity controller test file
    - Create `tests/controllers/activity.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 11.2 Test activity tracking endpoints
    - Test GET /api/v1/auth/activity with valid token
    - Test get activity without token
    - Test get activity with pagination
    - _Requirements: 11.1, 11.2, 11.3_

- [x] 12. Implement profile management controller tests
  - [x] 12.1 Create profile controller test file
    - Create `tests/controllers/profile.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 12.2 Test profile management endpoints
    - Test GET /api/v1/profile
    - Test PUT /api/v1/profile with valid data
    - Test update profile with invalid data
    - Test POST /api/v1/profile/image
    - Test upload invalid image format
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_
  
  - [x] 12.3 Write property test for profile update
    - **Property 31: Valid profile update succeeds**
    - **Validates: Requirements 12.2**
  
  - [x] 12.4 Write property test for invalid profile data
    - **Property 32: Invalid profile data rejected**
    - **Validates: Requirements 12.3**
  
  - [x] 12.5 Write property test for invalid image format
    - **Property 33: Invalid image format rejected**
    - **Validates: Requirements 12.5**

- [x] 13. Implement onboarding controller tests
  - [x] 13.1 Create onboarding controller test file
    - Create `tests/controllers/onboarding.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 13.2 Test onboarding endpoints
    - Test POST /api/v1/onboarding with valid data
    - Test onboarding with missing fields
    - Test onboarding for already onboarded user
    - Test GET /api/v1/onboarding/status
    - _Requirements: 13.1, 13.2, 13.3, 13.4_
  
  - [x] 13.3 Write property test for onboarding completion
    - **Property 34: Valid onboarding completion succeeds**
    - **Validates: Requirements 13.1**
  
  - [x] 13.4 Write property test for missing onboarding fields
    - **Property 35: Missing onboarding fields rejected**
    - **Validates: Requirements 13.2**

- [x] 14. Implement parent link controller tests
  - [x] 14.1 Create parent link controller test file
    - Create `tests/controllers/parent-link.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 14.2 Test parent link endpoints
    - Test POST /api/v1/parent-link with valid data
    - Test POST /api/v1/parent-link/verify with valid code
    - Test POST /api/v1/parent-link/accept
    - Test POST /api/v1/parent-link/reject
    - Test verify expired parent link
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5_
  
  - [x] 14.3 Write property test for parent link creation
    - **Property 36: Valid parent link creation succeeds**
    - **Validates: Requirements 14.1**
  
  - [x] 14.4 Write property test for parent link verification
    - **Property 37: Valid parent link verification succeeds**
    - **Validates: Requirements 14.2**

- [x] 15. Implement location tracking controller tests
  - [x] 15.1 Create location controller test file
    - Create `tests/controllers/location.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 15.2 Test location tracking endpoints
    - Test POST /api/v1/location with valid coordinates
    - Test update location with invalid coordinates
    - Test GET /api/v1/location/history
    - _Requirements: 15.1, 15.2, 15.3_
  
  - [x] 15.3 Write property test for location update
    - **Property 38: Valid location update succeeds**
    - **Validates: Requirements 15.1**
  
  - [x] 15.4 Write property test for invalid coordinates
    - **Property 39: Invalid coordinates rejected**
    - **Validates: Requirements 15.2**

- [x] 16. Implement internal service controller tests
  - [x] 16.1 Create internal controller test file
    - Create `tests/controllers/internal.controller.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 16.2 Test internal service endpoints
    - Test POST /api/v1/internal/validate-token with valid token
    - Test validate token with invalid token
    - Test GET /api/v1/internal/users/:userId with valid ID
    - Test get user with invalid ID
    - _Requirements: 16.1, 16.2, 16.3, 16.4_
  
  - [x] 16.3 Write property test for token validation
    - **Property 40: Valid token validation succeeds**
    - **Validates: Requirements 16.1**
  
  - [x] 16.4 Write property test for invalid token validation
    - **Property 41: Invalid token validation fails**
    - **Validates: Requirements 16.2**
  
  - [x] 16.5 Write property test for user retrieval
    - **Property 42: Valid user retrieval succeeds**
    - **Validates: Requirements 16.3**
  
  - [x] 16.6 Write property test for invalid user ID
    - **Property 43: Invalid user ID returns not found**
    - **Validates: Requirements 16.4**

- [x] 17. Implement middleware integration tests
  - [x] 17.1 Create middleware integration test file
    - Create `tests/controllers/middleware-integration.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 17.2 Test authentication middleware integration
    - Test protected endpoint with valid token
    - Test protected endpoint without token
    - Test protected endpoint with invalid token
    - Test protected endpoint with expired token
    - Test protected endpoint with revoked session
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5_
  
  - [x] 17.3 Write property test for valid authentication
    - **Property 1: Valid authentication succeeds**
    - **Validates: Requirements 17.1**
  
  - [x] 17.4 Write property test for missing authentication
    - **Property 2: Missing authentication fails**
    - **Validates: Requirements 17.2**
  
  - [x] 17.5 Write property test for invalid authentication
    - **Property 3: Invalid authentication fails**
    - **Validates: Requirements 17.3**
  
  - [x] 17.6 Test rate limiting middleware integration
    - Test requests within rate limit
    - Test exceed login rate limit
    - Test exceed registration rate limit
    - Test exceed OTP verification rate limit
    - Test rate limit window expires
    - _Requirements: 18.1, 18.2, 18.3, 18.4, 18.5_
  
  - [x] 17.7 Test error handling middleware integration
    - Test validation error format
    - Test authentication error format
    - Test authorization error format
    - Test not found error format
    - Test internal error format
    - Test error response includes required fields
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5, 19.6_
  
  - [x] 17.8 Write property test for error response format
    - **Property 44: Error responses have consistent format**
    - **Validates: Requirements 19.6, 21.2**

- [x] 18. Implement response format validation tests
  - [x] 18.1 Create response format test file
    - Create `tests/controllers/response-format.test.ts`
    - Set up test suite with beforeEach/afterEach
    - _Requirements: 22.1, 22.4_
  
  - [x] 18.2 Test request validation
    - Test request with missing required fields
    - Test request with invalid email format
    - Test request with invalid data types
    - Test request with out-of-range values
    - Test request with malformed JSON
    - _Requirements: 20.1, 20.2, 20.3, 20.4, 20.5_
  
  - [x] 18.3 Write property test for invalid data types
    - **Property 7: Invalid data types rejected**
    - **Validates: Requirements 20.3**
  
  - [x] 18.4 Write property test for out-of-range values
    - **Property 8: Out-of-range values rejected**
    - **Validates: Requirements 20.4**
  
  - [x] 18.5 Test response formats
    - Test successful response includes expected fields
    - Test list response includes array and metadata
    - Test token response includes both tokens
    - Test user data excludes sensitive fields
    - _Requirements: 21.1, 21.3, 21.4, 21.5_
  
  - [x] 18.6 Write property test for successful response format
    - **Property 45: Successful responses include expected fields**
    - **Validates: Requirements 21.1**
  
  - [x] 18.7 Write property test for list response format
    - **Property 46: List responses include array and metadata**
    - **Validates: Requirements 21.3**
  
  - [x] 18.8 Write property test for token response format
    - **Property 47: Token responses include both tokens**
    - **Validates: Requirements 21.4**
  
  - [x] 18.9 Write property test for user data sanitization
    - **Property 48: User data excludes sensitive fields**
    - **Validates: Requirements 21.5**

- [x] 19. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks are required for comprehensive testing coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Tests use Vitest (already configured) and supertest for HTTP testing
- All external dependencies (Prisma, Redis, email, etc.) are mocked
- Target coverage: 70% for controllers
