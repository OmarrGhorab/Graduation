# Implementation Plan: API Gateway Refactoring

## Overview

This plan refactors the API Gateway from a monolithic single-file application into a modular, maintainable, and well-tested codebase. Tasks are organized to build incrementally, with testing integrated throughout to catch issues early.

## Tasks

- [x] 1. Set up testing infrastructure
  - Install Jest, ts-jest, @types/jest, and fast-check
  - Create jest.config.js with TypeScript support
  - Create tests/helpers/mocks.ts with utilities for mocking Express req/res and external dependencies
  - Update package.json with test scripts (test, test:watch, test:coverage)
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [x] 2. Create configuration module
  - [x] 2.1 Implement config/index.ts with loadConfig() and validateConfig()
    - Define TypeScript interfaces: ServerConfig, CorsConfig, ServiceEndpoint, ServicesConfig, SecurityConfig, AppConfig
    - Load environment variables with dotenv
    - Provide default values for optional configuration
    - Validate required values and throw descriptive errors if missing
    - Parse ALLOWED_ORIGINS comma-separated list
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 9.1_

  - [ ]* 2.2 Write property test for configuration validation
    - **Property 1: Configuration validation rejects invalid inputs**
    - **Validates: Requirements 2.3, 2.5**
    - Generate random configurations with missing required fields
    - Verify each throws an error identifying the missing field

  - [ ]* 2.3 Write property test for default values
    - **Property 2: Configuration provides defaults for optional values**
    - **Validates: Requirements 2.4**
    - Generate configurations with various optional fields missing
    - Verify defaults are applied correctly

  - [ ]* 2.4 Write unit tests for configuration module
    - Test valid configuration loading
    - Test invalid PORT values
    - Test invalid NODE_ENV values
    - Test empty ALLOWED_ORIGINS
    - Test ALLOWED_ORIGINS parsing
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 3. Create middleware modules
  - [x] 3.1 Implement middleware/cors.middleware.ts
    - Export createCorsMiddleware(config: CorsConfig)
    - Handle requests with no origin (mobile apps)
    - Check origin against whitelist
    - Support wildcard (*) for development
    - _Requirements: 3.1_

  - [x] 3.2 Implement middleware/compression.middleware.ts
    - Export createCompressionMiddleware()
    - Set compression level to 6
    - Set threshold to 1KB
    - Exclude SSE streams (text/event-stream)
    - _Requirements: 3.2_

  - [x] 3.3 Implement middleware/timeout.middleware.ts
    - Export createTimeoutMiddleware(timeoutMs: number)
    - Return array with timeout middleware and halt-on-timeout middleware
    - Default timeout: 30 seconds
    - _Requirements: 3.3_

  - [x] 3.4 Implement middleware/arcjet.middleware.ts
    - Define ArcjetConfig interface
    - Export createArcjetMiddleware(config: ArcjetConfig)
    - Skip protection if disabled or no key
    - Detect and block bots (allow search engines and monitors)
    - Block VPN, proxy, hosting, relay IPs
    - Return 403 for blocked requests
    - Fail open on errors
    - Log blocking decisions
    - _Requirements: 3.4_

  - [x] 3.5 Implement middleware/error.middleware.ts
    - Define ErrorResponse interface
    - Export errorHandler function
    - Log errors with stack traces
    - Return consistent JSON error format
    - Map error types to HTTP status codes
    - Include timestamp
    - _Requirements: 6.1, 6.2, 6.3_

  - [x] 3.6 Implement middleware/index.ts
    - Export setupMiddleware(app: Express, config: AppConfig)
    - Apply middleware in correct order: compression, timeout, body parsing, CORS, Arcjet
    - Ensure halt-on-timeout checks between body parsers
    - _Requirements: 3.5_

  - [x] 3.7 Write unit tests for CORS middleware

    - Test requests with no origin (should allow)
    - Test requests with whitelisted origin (should allow)
    - Test requests with non-whitelisted origin (should block)
    - Test wildcard origin (should allow all)
    - _Requirements: 3.1_

  - [x] 3.8 Write unit tests for compression middleware

    - Test responses > 1KB are compressed
    - Test responses < 1KB are not compressed
    - Test SSE streams are not compressed
    - _Requirements: 3.2_

  - [x] 3.9 Write unit tests for timeout middleware

    - Test requests completing within timeout
    - Test requests exceeding timeout
    - Test halt-on-timeout prevents further processing
    - _Requirements: 3.3_

  - [x] 3.10 Write unit tests for Arcjet middleware

    - Test disabled protection (should allow all)
    - Test bot detection (should block malicious bots)
    - Test VPN/proxy detection (should block)
    - Test allowed bots (search engines, monitors)
    - Test error handling (should fail open)
    - _Requirements: 3.4_


  - [x] 3.11 Write property test for error handling

    - **Property 11: Errors produce consistent response format**
    - **Validates: Requirements 6.2, 6.3**
    - Generate random errors
    - Verify all responses contain error, statusCode, and timestamp fields
    - _Requirements: 6.2, 6.3_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Create health check service
  - [x] 5.1 Implement services/health.service.ts
    - Define ServiceHealth and HealthCheckResponse interfaces
    - Export checkServiceHealth(url, name, timeoutMs)
    - Export checkAllServices(services)
    - Check services in parallel
    - Measure latency
    - 5-second timeout per service
    - Handle network errors gracefully
    - Return 200 if all healthy, 503 if any unhealthy
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7_

  - [x] 5.2 Write property test for upstream service verification

    - **Property 3: Health check verifies all upstream services**
    - **Validates: Requirements 4.2**
    - Generate random lists of upstream services
    - Verify health check attempts to reach each one
    - _Requirements: 4.2_

  - [x] 5.3 Write property test for latency measurement

    - **Property 4: Health check includes latency for responsive services**
    - **Validates: Requirements 4.3**
    - Generate random upstream services with varying response times
    - Verify latency is measured and included for successful responses
    - _Requirements: 4.3_

  - [x] 5.4 Write property test for unhealthy service status

    - **Property 5: Health check returns 503 for unhealthy services**
    - **Validates: Requirements 4.5**
    - Generate random upstream services with some unhealthy
    - Verify 503 status when any service is unhealthy
    - _Requirements: 4.5_

  - [x] 5.5 Write property test for timestamps

    - **Property 6: Health check includes timestamps**
    - **Validates: Requirements 4.6**
    - Generate random health check scenarios
    - Verify all responses contain valid ISO 8601 timestamps
    - _Requirements: 4.6_

  - [x] 5.6 Write property test for timeout handling

    - **Property 7: Health check handles timeouts gracefully**
    - **Validates: Requirements 4.7**
    - Generate upstream services that timeout
    - Verify health check marks them as unhealthy without crashing
    - Verify health check completes within reasonable time
    - _Requirements: 4.7_

  - [x] 5.7 Write unit tests for health check service

    - Test all services healthy (should return 200)
    - Test one service unhealthy (should return 503)
    - Test all services unhealthy (should return 503)
    - Test service timeout (should mark as unhealthy)
    - Test network error (should mark as unhealthy)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7_

- [x] 6. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Create proxy routing module
  - [x] 7.1 Implement routes/index.ts
    - Define ProxyRoute interface
    - Export setupRoutes(app: Express, config: AppConfig)
    - Set up /health route (not proxied)
    - Set up /api/v1/notifications route to notification service
    - Set up /api/v1/location/request route to notification service
    - Set up / catch-all route to auth service
    - Preserve original request paths
    - _Requirements: 5.1, 5.2, 5.3, 5.5_

  - [x] 7.2 Write property test for proxy routing

    - **Property 8: Proxy routes requests to correct upstream**
    - **Validates: Requirements 5.2**
    - Generate random request paths
    - Verify each is routed to the correct upstream service
    - _Requirements: 5.2_

  - [x] 7.3 Write property test for path preservation

    - **Property 9: Proxy preserves request paths**
    - **Validates: Requirements 5.3**
    - Generate random request paths
    - Verify upstream receives the same path
    - _Requirements: 5.3_

  - [x] 7.4 Write property test for proxy error handling

    - **Property 10: Proxy handles upstream errors gracefully**
    - **Validates: Requirements 5.4**
    - Generate random proxy errors (unreachable, timeout, error response)
    - Verify gateway returns appropriate error without crashing
    - _Requirements: 5.4_

  - [x] 7.5 Write unit tests for proxy routing

    - Test /health route (should not proxy)
    - Test /api/v1/notifications route (should proxy to notification service)
    - Test /api/v1/location/request route (should proxy to notification service)
    - Test / route (should proxy to auth service)
    - Test route priority (more specific routes match first)
    - _Requirements: 5.1, 5.2, 5.3, 5.5_

- [x] 8. Create application bootstrap
  - [x] 8.1 Implement app.ts
    - Export createApp(config: AppConfig)
    - Set up middleware using setupMiddleware()
    - Set up routes using setupRoutes()
    - Add error handler (must be last)
    - Return configured Express app
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [x] 8.2 Refactor main.ts
    - Load configuration using loadConfig()
    - Create Express app using createApp()
    - Start HTTP server
    - Log startup message
    - Handle startup errors
    - _Requirements: 1.5_

  - [x] 8.3 Write integration tests for application bootstrap

    - Test app starts successfully with valid config
    - Test app fails to start with invalid config
    - Test all middleware is applied
    - Test all routes are registered
    - Test error handler is last middleware
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 9. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Add documentation
  - [x] 10.1 Add JSDoc comments to all exported functions and interfaces
    - Document config/index.ts exports
    - Document middleware exports
    - Document health service exports
    - Document routes exports
    - _Requirements: 10.1, 10.2_

  - [x] 10.2 Add inline comments for complex logic
    - Comment Arcjet protection logic
    - Comment health check parallel execution
    - Comment middleware ordering rationale
    - _Requirements: 10.3_

  - [x] 10.3 Create README.md
    - Architecture overview with directory structure
    - Configuration guide (environment variables)
    - Development guide (running, testing)
    - Deployment guide
    - _Requirements: 10.4, 10.5_

- [x] 11. Final verification and cleanup
  - [x] 11.1 Run full test suite and verify 80% coverage
    - Run `npm test`
    - Run `npm run test:coverage`
    - Verify coverage meets 80% threshold
    - _Requirements: 7.6_

  - [x] 11.2 Verify TypeScript compilation
    - Run `tsc --noEmit` to check for type errors
    - Ensure no 'any' types except where necessary
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

  - [x] 11.3 Test refactored gateway with existing services
    - Start auth-service and notification-service
    - Start refactored api-gateway
    - Test all proxy routes work correctly
    - Test health check endpoint
    - Verify backward compatibility
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [x] 11.4 Update .gitignore if needed
    - Ensure coverage/ directory is ignored
    - Ensure dist/ directory is ignored

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- All modules are designed to be independently testable with mocked dependencies
