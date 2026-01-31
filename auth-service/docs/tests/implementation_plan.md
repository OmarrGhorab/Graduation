# Test Fix Implementation Plan

## Goal Description
Fix the 5 failed test suites and 24 failed tests in `auth-service`. The primary focus is to resolve the `response-format.test.ts` failure and identifying/fixing the remaining failures which appear to be related to Refresh Token validation and potential mock inconsistencies.

## Proposed Changes

### 1. Fix Response Format Tests
#### [MODIFY] [response-format.test.ts](file:///d:/Graduation/auth-service/tests/controllers/response-format.test.ts)
- Investigate why `should include both accessToken and refreshToken in auth responses` returns 400 instead of 200.
- Verify if the `createVerifiedUserFixture` password hash matches the expectations of the login controller.
- Ensure mocks for `prisma.user.findUnique` and `bcrypt.compare` (if applicable) are correctly set up.
- Fix potentially missing mocks for `checkPassword` or similar utilities if they are separated.

### 2. Identify and Fix Remaining Failures
- Run `npx jest` again after fixing the first issue to clearly list the remaining 4 failed suites.
- Address issues related to "Refresh Token" console errors (likely in `auth.controller.test.ts`, `sessions.controller.test.ts`, etc.).
- Common suspects for "jwt malformed" or "Invalid refresh token format":
    - `src/controllers/auth.core.controller.ts`
    - `tests/controllers/auth.controller.test.ts`
    - `tests/controllers/sessions.controller.test.ts`
    - `tests/controllers/oauth.controller.test.ts`

## Verification Plan

### Automated Tests
- Run specific test file during debugging:
    ```powershell
    npx jest tests/controllers/response-format.test.ts
    ```
- Run full test suite to confirm all fixes:
    ```powershell
    npx jest
    ```
