import { errorHandler, ErrorResponse } from '../../src/middleware/error.middleware';
import { mockRequest, mockResponse, mockNext } from '../helpers/mocks';
import * as fc from 'fast-check';

describe('Error Middleware', () => {
  describe('Property-Based Tests', () => {
    /**
     * Feature: api-gateway-refactoring, Property 11: Errors produce consistent response format
     * Validates: Requirements 6.2, 6.3
     */
    it('should produce consistent response format for all errors', () => {
      fc.assert(
        fc.property(
          // Generate random error configurations
          fc.record({
            name: fc.oneof(
              fc.constant('Error'),
              fc.constant('TimeoutError'),
              fc.constant('ProxyError'),
              fc.constant('ValidationError'),
              fc.constant('UnauthorizedError'),
              fc.constant('ForbiddenError'),
              fc.constant('NotFoundError'),
              fc.string({ minLength: 1, maxLength: 50 })
            ),
            message: fc.string({ minLength: 1, maxLength: 200 }),
            statusCode: fc.option(fc.integer({ min: 400, max: 599 }), { nil: undefined }),
            status: fc.option(fc.integer({ min: 400, max: 599 }), { nil: undefined }),
          }),
          (errorConfig) => {
            // Create error with random properties
            const error = new Error(errorConfig.message) as Error & {
              statusCode?: number;
              status?: number;
            };
            error.name = errorConfig.name;
            if (errorConfig.statusCode !== undefined) {
              error.statusCode = errorConfig.statusCode;
            }
            if (errorConfig.status !== undefined) {
              error.status = errorConfig.status;
            }

            // Create mock request and response
            const req = mockRequest();
            const res = mockResponse();
            const next = mockNext();

            // Call error handler
            errorHandler(error, req as any, res as any, next);

            // Verify response was called with status
            expect(res.status).toHaveBeenCalled();
            const statusCall = (res.status as jest.Mock).mock.calls[0][0];
            expect(typeof statusCall).toBe('number');
            expect(statusCall).toBeGreaterThanOrEqual(400);
            expect(statusCall).toBeLessThan(600);

            // Verify json was called with proper structure
            expect(res.json).toHaveBeenCalled();
            const jsonCall = (res.json as jest.Mock).mock.calls[0][0] as ErrorResponse;

            // Property: All error responses must contain error, statusCode, and timestamp fields
            expect(jsonCall).toHaveProperty('error');
            expect(jsonCall).toHaveProperty('statusCode');
            expect(jsonCall).toHaveProperty('timestamp');

            // Verify field types
            expect(typeof jsonCall.error).toBe('string');
            expect(typeof jsonCall.statusCode).toBe('number');
            expect(typeof jsonCall.timestamp).toBe('string');

            // Verify statusCode is valid HTTP error code
            expect(jsonCall.statusCode).toBeGreaterThanOrEqual(400);
            expect(jsonCall.statusCode).toBeLessThan(600);

            // Verify timestamp is valid ISO 8601 format
            expect(() => new Date(jsonCall.timestamp)).not.toThrow();
            expect(new Date(jsonCall.timestamp).toISOString()).toBe(jsonCall.timestamp);

            // Verify error field is non-empty
            expect(jsonCall.error.length).toBeGreaterThan(0);
          }
        ),
        { numRuns: 100 } // Run 100 iterations as specified in design
      );
    });
  });
});
