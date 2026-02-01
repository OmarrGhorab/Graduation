# Auth Service Unit Tests Spec - Updates Summary

## Changes Made

### 1. Testing Framework Migration: Vitest → Jest

**Rationale:** Align auth-service testing framework with api-gateway which already uses Jest, providing consistency across the codebase.

**Files Updated:**
- `requirements.md` - Updated Requirement 1 to specify Jest instead of Vitest
- `design.md` - Updated all configuration examples, mock syntax, and test utilities to use Jest
- `tasks.md` - Completely rewritten with comprehensive task breakdown

### 2. Requirements Document Updates

**Changes:**
- Glossary: Changed "Vitest" to "Jest"
- Requirement 1.1: Changed from "Vitest" to "Jest"
- Requirement 1.4: Changed config file from `vitest.config.ts` to `jest.config.cjs`

### 3. Design Document Updates

**Changes:**
- Overview: Updated to specify Jest as the testing framework
- Test Structure: Changed setup file from `vitest.setup.ts` to `jest.setup.ts`
- Mock Strategy: Changed from `vi.mock()` to `jest.mock()`
- Jest Configuration: Replaced Vitest config with comprehensive Jest config including:
  - `ts-jest` preset for TypeScript support
  - Path aliases for `@/` imports
  - Coverage thresholds per directory (utils: 80%, services: 75%, middleware: 70%)
  - Test timeout and mock reset configuration
- Mock Implementations: Changed all `vi.fn()` to `jest.fn()`
- Test Utilities: Changed `vi.advanceTimersByTime()` to `jest.advanceTimersByTime()`
- Test Execution: Updated commands to use Jest instead of Vitest

### 4. Tasks Document - Complete Rewrite

**New Structure:**
- **20 major tasks** covering all testing requirements
- **100+ subtasks** with specific implementation details
- Each task references specific requirements and properties
- Clear progression: Infrastructure → Mocks → Utils → Services → Middleware → Verification

**Task Breakdown:**

1. **Task 1:** Test infrastructure setup (Jest installation and configuration)
2. **Task 2:** Mock infrastructure (5 subtasks for Prisma, Redis, Email, Cloudinary, Fetch)
3. **Task 3:** Test utilities and factories
4. **Tasks 4-10:** Utils testing (7 tasks covering all util modules)
   - Token Management (8 subtasks)
   - Session Management (7 subtasks)
   - OTP Management (7 subtasks)
   - Two-Factor Authentication (7 subtasks)
   - Email Verification (6 subtasks)
   - Password Reset (5 subtasks)
   - Additional Utils (3 subtasks for cookies, device, errors)
5. **Tasks 11-13:** Services testing (3 tasks)
   - Auth Session Service (7 subtasks)
   - Location Service (3 subtasks)
   - Parent Link Service (5 subtasks)
6. **Tasks 14-16:** Middleware testing (3 tasks)
   - Authentication Middleware (7 subtasks)
   - Rate Limiting Middleware (4 subtasks)
   - Error Handler Middleware (4 subtasks)
7. **Task 17:** Test infrastructure property verification (6 subtasks)
8. **Task 18:** Coverage verification
9. **Task 19:** Performance optimization
10. **Task 20:** Final verification and documentation

**Key Features:**
- Every subtask includes both unit tests AND property-based tests
- All 64 correctness properties from the design document are covered
- Each task references specific requirements
- Clear acceptance criteria for each subtask
- Comprehensive edge case testing

### 5. Coverage Targets

**Per-Directory Thresholds:**
- Utils: 80% (lines, functions, statements), 75% (branches)
- Services: 75% (lines, functions, statements), 70% (branches)
- Middleware: 70% (lines, functions, statements), 65% (branches)

### 6. Property-Based Testing

All tasks include property-based tests using `fast-check`:
- Minimum 100 iterations per property test
- References to specific properties from design document
- Proper test tagging format

## Migration Steps

To migrate from Vitest to Jest:

1. **Remove Vitest:**
   ```bash
   npm uninstall vitest @vitest/coverage-v8
   ```

2. **Install Jest:**
   ```bash
   npm install --save-dev jest @types/jest ts-jest @jest/globals
   ```

3. **Update Configuration:**
   - Delete `vitest.config.ts`
   - Create `jest.config.cjs` (see design document for full config)

4. **Update Test Files:**
   - Rename `tests/setup/vitest.setup.ts` to `tests/setup/jest.setup.ts`
   - Replace `vi` imports with `jest` from `@jest/globals`
   - Update all `vi.fn()` to `jest.fn()`
   - Update all `vi.mock()` to `jest.mock()`

5. **Update package.json scripts:**
   ```json
   {
     "test": "jest",
     "test:watch": "jest --watch",
     "test:coverage": "jest --coverage"
   }
   ```

## Test Execution

After migration:

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

## Verification Completed

All 64 correctness properties have been verified as covered in the tasks.md implementation plan. See the following documents for detailed verification:

- `property-coverage-verification.md` - Complete property-to-task mapping
- `verification-summary.md` - Comprehensive verification results and analysis

## Next Steps

1. Review the updated spec documents
2. Execute Task 1 to set up Jest infrastructure
3. Proceed through tasks sequentially
4. Each task builds on previous work
5. Verify coverage after completing all tests

## Verification Checklist

- [x] All references to Vitest changed to Jest
- [x] Mock syntax updated (vi.fn → jest.fn)
- [x] Configuration files updated
- [x] All 18 requirements covered in tasks
- [x] All 64 correctness properties covered in tasks
- [x] Property-based tests included for all applicable tasks
- [x] Edge cases covered for all modules
- [x] Coverage thresholds properly configured
- [x] Test execution commands updated
