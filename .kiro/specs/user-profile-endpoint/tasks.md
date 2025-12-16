# Implementation Plan

- [ ] 1. Implement getMyProfile controller function
  - Add getMyProfile function to auth.core.controller.ts
  - Extract user ID from req.user (populated by authenticate middleware)
  - Query database using Prisma to get user profile
  - Select only safe fields (exclude password, deletedAt, deviceBlocked, pendingDeviceFingerprint, twoFactorSecret, twoFactorBackupCodes)
  - Return 404 error if user not found
  - Return user profile data in response
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3_

- [ ] 1.1 Write property test for authenticated access requirement
  - **Property 1: Authenticated access only**
  - **Validates: Requirements 1.5**
  - Test that requests without valid tokens return 401
  - Use fast-check to generate various invalid token scenarios

- [ ] 1.2 Write property test for user data retrieval
  - **Property 2: User data retrieval**
  - **Validates: Requirements 1.1, 1.2, 1.3**
  - Test that valid user IDs return correct profile data
  - Use fast-check to generate random valid user IDs

- [ ] 1.3 Write property test for sensitive data exclusion
  - **Property 3: Sensitive data exclusion**
  - **Validates: Requirements 2.1, 2.2, 2.3**
  - Test that response never contains sensitive fields
  - Use fast-check to verify field exclusion across all responses

- [ ] 1.4 Write property test for non-existent user handling
  - **Property 4: Non-existent user handling**
  - **Validates: Requirements 1.4**
  - Test that invalid user IDs return 404
  - Use fast-check to generate random non-existent user IDs

- [x] 2. Add route to auth router





  - Import getMyProfile function in auth.route.ts
  - Add GET /myprofile route with authenticate middleware
  - Ensure route is properly positioned in router (before parameterized routes)
  - _Requirements: 3.1, 3.2, 3.3_   

- [x] 2.1 Write unit test for route integration


  - Test that route is accessible at /myprofile
  - Test that authenticate middleware is applied
  - Test successful profile retrieval flow

- [ ] 3. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.
