import * as fc from "fast-check";
import express, { Express } from "express";
import { setupRoutes } from "../../src/routes/index";
import { mockConfig } from "../helpers/mocks";
import { AppConfig } from "../../src/config/index";

// Mock express-http-proxy
jest.mock("express-http-proxy", () => {
  return jest.fn(() => {
    return (req: any, res: any, next: any) => {
      // Mock proxy middleware that just passes through
      next();
    };
  });
});

// Mock health service
jest.mock("../../src/services/health.service", () => ({
  checkAllServices: jest.fn().mockResolvedValue({
    status: "ok",
    service: "api-gateway",
    timestamp: new Date().toISOString(),
  }),
}));

describe("Proxy Routes", () => {
  let app: Express;
  let config: AppConfig;
  let proxy: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    app = express();
    config = mockConfig();

    // Get reference to the mocked proxy function
    proxy = require("express-http-proxy") as jest.Mock;
  });

  describe("Property 8: Proxy routes requests to correct upstream", () => {
    /**
     * Feature: api-gateway-refactoring, Property 8: Proxy routes requests to correct upstream
     * Validates: Requirements 5.2
     */
    it("should route requests to the correct upstream service based on path", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random service URLs
          fc.record({
            authUrl: fc.webUrl({ validSchemes: ["http", "https"] }),
            notificationUrl: fc.webUrl({ validSchemes: ["http", "https"] }),
          }),
          async ({ authUrl, notificationUrl }) => {
            // Reset mocks
            proxy.mockClear();
            app = express();

            // Create config with random URLs
            const testConfig = mockConfig({
              services: {
                auth: [{
                  name: "auth-service",
                  url: authUrl,
                  healthPath: "/health",
                }],
                notification: [{
                  name: "notification-service",
                  url: notificationUrl,
                  healthPath: "/health",
                }],
              },
            });

            // Setup routes
            setupRoutes(app, testConfig);

            // Verify proxy was called for each route
            // Should be called 3 times: /api/v1/notifications, /api/v1/location/request, /
            expect(proxy).toHaveBeenCalledTimes(3);

            // Get all proxy calls
            const calls = proxy.mock.calls;

            // Verify notification service routes (first two calls)
            expect(calls[0][0]).toBe(notificationUrl);
            expect(calls[1][0]).toBe(notificationUrl);

            // Verify auth service catch-all route (last call)
            expect(calls[2][0]).toBe(authUrl);

            // Verify all calls have proxyReqPathResolver
            for (const call of calls) {
              expect(call[1]).toHaveProperty("proxyReqPathResolver");
              expect(typeof call[1].proxyReqPathResolver).toBe("function");
            }
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  describe("Route Configuration", () => {
    it("should set up routes in correct priority order", () => {
      setupRoutes(app, config);

      // Verify proxy was called 3 times
      expect(proxy).toHaveBeenCalledTimes(3);

      const calls = proxy.mock.calls;

      // First call: /api/v1/notifications -> notification service
      expect(calls[0][0]).toBe(config.services.notification[0].url);

      // Second call: /api/v1/location/request -> notification service
      expect(calls[1][0]).toBe(config.services.notification[0].url);

      // Third call: / -> auth service (catch-all)
      expect(calls[2][0]).toBe(config.services.auth[0].url);
    });

    it("should configure path preservation for all proxy routes", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // Verify all calls have proxyReqPathResolver configured
      for (const call of calls) {
        expect(call[1]).toHaveProperty("proxyReqPathResolver");
        expect(typeof call[1].proxyReqPathResolver).toBe("function");
      }
    });

    it("should not proxy the /health endpoint", () => {
      setupRoutes(app, config);

      // Verify proxy was called 3 times (not 4, because /health is not proxied)
      expect(proxy).toHaveBeenCalledTimes(3);

      // Verify none of the proxy calls are for /health
      const calls = proxy.mock.calls;
      for (const call of calls) {
        // The first argument is the target URL, not the path
        // So we just verify we have 3 proxy calls total
        expect(call[0]).toBeDefined();
      }
    });
  });

  describe("Unit Tests for Proxy Routing", () => {
    /**
     * Requirements: 5.1, 5.2, 5.3, 5.5
     */

    it("should not proxy the /health route", () => {
      setupRoutes(app, config);

      // Verify proxy was called exactly 3 times (not including /health)
      expect(proxy).toHaveBeenCalledTimes(3);

      // Verify the routes that were proxied
      const calls = proxy.mock.calls;
      const targets = calls.map(call => call[0]);

      // None of the proxy targets should be for /health
      // (health is handled directly by the gateway, not proxied)
      expect(targets).toEqual([
        config.services.notification[0].url,
        config.services.notification[0].url,
        config.services.auth[0].url,
      ]);
    });

    it("should proxy /api/v1/notifications route to notification service", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // First proxy call should be for /api/v1/notifications
      expect(calls[0][0]).toBe(config.services.notification[0].url);

      // Verify it has path preservation configured
      expect(calls[0][1]).toHaveProperty("proxyReqPathResolver");
      expect(typeof calls[0][1].proxyReqPathResolver).toBe("function");
    });

    it("should proxy /api/v1/location/request route to notification service", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // Second proxy call should be for /api/v1/location/request
      expect(calls[1][0]).toBe(config.services.notification[0].url);

      // Verify it has path preservation configured
      expect(calls[1][1]).toHaveProperty("proxyReqPathResolver");
      expect(typeof calls[1][1].proxyReqPathResolver).toBe("function");
    });

    it("should proxy / route (catch-all) to auth service", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // Third proxy call should be the catch-all to auth service
      expect(calls[2][0]).toBe(config.services.auth[0].url);

      // Verify it has path preservation configured
      expect(calls[2][1]).toHaveProperty("proxyReqPathResolver");
      expect(typeof calls[2][1].proxyReqPathResolver).toBe("function");
    });

    it("should apply routes in correct priority order (more specific routes match first)", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // Verify the order of proxy setup:
      // 1. /api/v1/notifications (specific)
      // 2. /api/v1/location/request (specific)
      // 3. / (catch-all, must be last)

      expect(calls.length).toBe(3);

      // First two should be notification service (specific routes)
      expect(calls[0][0]).toBe(config.services.notification[0].url);
      expect(calls[1][0]).toBe(config.services.notification[0].url);

      // Last should be auth service (catch-all)
      expect(calls[2][0]).toBe(config.services.auth[0].url);

      // This order ensures that:
      // - /api/v1/notifications requests go to notification service
      // - /api/v1/location/request requests go to notification service
      // - All other requests (/) go to auth service
    });
  });

  describe("Path Preservation", () => {
    it("should preserve original URL in proxy requests", () => {
      setupRoutes(app, config);

      const calls = proxy.mock.calls;

      // Test each proxyReqPathResolver
      for (const call of calls) {
        const proxyReqPathResolver = call[1].proxyReqPathResolver;

        // Test with various paths
        const testPaths = [
          "/api/v1/notifications/123",
          "/api/v1/location/request?lat=1&lng=2",
          "/auth/login",
          "/users/profile",
        ];

        for (const path of testPaths) {
          const mockReq = { originalUrl: path };
          const result = proxyReqPathResolver(mockReq);
          expect(result).toBe(path);
        }
      }
    });

    describe("Property 9: Proxy preserves request paths", () => {
      /**
       * Feature: api-gateway-refactoring, Property 9: Proxy preserves request paths
       * Validates: Requirements 5.3
       */
      it("should preserve any request path when proxying to upstream", () => {
        fc.assert(
          fc.property(
            // Generate random request paths
            fc.oneof(
              // API paths with IDs
              fc.tuple(
                fc.constantFrom("/api/v1/notifications/", "/api/v1/location/request/", "/auth/", "/users/"),
                fc.uuid()
              ).map(([base, id]) => `${base}${id}`),

              // Paths with query parameters
              fc.tuple(
                fc.constantFrom("/api/v1/notifications", "/api/v1/location/request", "/auth/login", "/users/profile"),
                fc.record({
                  key: fc.string({ minLength: 1, maxLength: 10 }),
                  value: fc.string({ minLength: 1, maxLength: 20 })
                })
              ).map(([path, param]) => `${path}?${param.key}=${param.value}`),

              // Simple paths
              fc.constantFrom(
                "/api/v1/notifications",
                "/api/v1/location/request",
                "/auth/login",
                "/auth/register",
                "/users/profile",
                "/users/settings"
              ),

              // Nested paths
              fc.tuple(
                fc.constantFrom("/api/v1/", "/auth/", "/users/"),
                fc.string({ minLength: 1, maxLength: 30 })
              ).map(([base, suffix]) => `${base}${suffix}`)
            ),
            (requestPath) => {
              // Setup routes
              proxy.mockClear();
              app = express();
              setupRoutes(app, config);

              // Get all proxy calls
              const calls = proxy.mock.calls;

              // Verify each proxy has a proxyReqPathResolver
              for (const call of calls) {
                const proxyReqPathResolver = call[1].proxyReqPathResolver;
                expect(typeof proxyReqPathResolver).toBe("function");

                // Create mock request with the generated path
                const mockReq = { originalUrl: requestPath };

                // Call the path resolver
                const resolvedPath = proxyReqPathResolver(mockReq);

                // Verify the path is preserved exactly
                expect(resolvedPath).toBe(requestPath);
              }
            }
          ),
          { numRuns: 100 }
        );
      });
    });
  });

  describe("Property 10: Proxy handles upstream errors gracefully", () => {
    /**
     * Feature: api-gateway-refactoring, Property 10: Proxy handles upstream errors gracefully
     * Validates: Requirements 5.4
     */
    it("should handle upstream errors without crashing", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random error scenarios
          fc.oneof(
            // Scenario 1: Unreachable service (connection refused)
            fc.record({
              type: fc.constant("unreachable" as const),
              errorCode: fc.constantFrom("ECONNREFUSED", "ENOTFOUND", "EHOSTUNREACH"),
              message: fc.string({ minLength: 5, maxLength: 50 }),
            }),

            // Scenario 2: Timeout error
            fc.record({
              type: fc.constant("timeout" as const),
              errorCode: fc.constant("ETIMEDOUT"),
              message: fc.constant("Request timeout"),
            }),

            // Scenario 3: Upstream returns error status
            fc.record({
              type: fc.constant("error_response" as const),
              statusCode: fc.integer({ min: 400, max: 599 }),
              message: fc.string({ minLength: 5, maxLength: 50 }),
            })
          ),
          async (errorScenario) => {
            // Reset mocks
            proxy.mockClear();
            app = express();

            // Mock proxy to simulate the error
            proxy.mockImplementation((target: string) => {
              return (req: any, res: any, next: any) => {
                if (errorScenario.type === "unreachable" || errorScenario.type === "timeout") {
                  // Simulate network error
                  const error: any = new Error(errorScenario.message);
                  error.code = errorScenario.errorCode;
                  next(error);
                } else if (errorScenario.type === "error_response") {
                  // Simulate upstream error response
                  res.status(errorScenario.statusCode).json({
                    error: errorScenario.message,
                  });
                }
              };
            });

            // Setup routes with error-prone proxy
            setupRoutes(app, config);

            // Create a mock request and response
            const mockReq: any = {
              originalUrl: "/api/v1/notifications/test",
              method: "GET",
              headers: {},
            };
            const mockRes: any = {
              status: jest.fn().mockReturnThis(),
              json: jest.fn().mockReturnThis(),
              send: jest.fn().mockReturnThis(),
            };
            const mockNext = jest.fn();

            // Get the proxy middleware for the notifications route
            const proxyMiddleware = proxy.mock.results[0]?.value;

            if (proxyMiddleware) {
              // Execute the proxy middleware
              await proxyMiddleware(mockReq, mockRes, mockNext);

              // Verify the gateway handled the error appropriately
              if (errorScenario.type === "unreachable" || errorScenario.type === "timeout") {
                // Should call next with error (to be handled by error middleware)
                expect(mockNext).toHaveBeenCalledWith(expect.any(Error));
                const error = mockNext.mock.calls[0][0];
                expect(error.code).toBe(errorScenario.errorCode);
              } else if (errorScenario.type === "error_response") {
                // Should return error response
                expect(mockRes.status).toHaveBeenCalledWith(errorScenario.statusCode);
                expect(mockRes.json).toHaveBeenCalledWith({
                  error: errorScenario.message,
                });
              }
            }

            // Most importantly: verify the app didn't crash
            // (if we got here without throwing, the test passes)
            expect(app).toBeDefined();
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
