import * as fc from "fast-check";
import {
  checkServiceHealth,
  checkAllServices,
  ServiceHealth,
  HealthCheckResponse,
} from "../../src/services/health.service";
import { ServiceEndpoint } from "../../src/config/index";

// Mock fetch globally
global.fetch = jest.fn();

describe("Health Service", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("Property 3: Health check verifies all upstream services", () => {
    /**
     * Feature: api-gateway-refactoring, Property 3: Health check verifies all upstream services
     * Validates: Requirements 4.2
     */
    it("should attempt to reach each configured upstream service", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random lists of upstream services (1-5 services)
          fc.array(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 20 }),
              url: fc.webUrl(),
              healthPath: fc.constantFrom("/health", "/status", "/ping"),
            }),
            { minLength: 1, maxLength: 5 }
          ),
          async (services: ServiceEndpoint[]) => {
            // Reset mock before each property test run
            (global.fetch as jest.Mock).mockClear();
            
            // Mock fetch to return successful responses
            (global.fetch as jest.Mock).mockResolvedValue({
              ok: true,
              status: 200,
            });

            const result = await checkAllServices(services);

            // Verify that fetch was called for each service
            expect(global.fetch).toHaveBeenCalledTimes(services.length);

            // Verify each service appears in the upstreams
            if (result.upstreams) {
              for (const service of services) {
                expect(result.upstreams[service.name]).toBeDefined();
              }
            }
          }
        ),
        { numRuns: 20 }
      );
    });
  });

  describe("Property 4: Health check includes latency for responsive services", () => {
    /**
     * Feature: api-gateway-refactoring, Property 4: Health check includes latency for responsive services
     * Validates: Requirements 4.3
     */
    it("should measure and include latency for successful responses", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random lists of upstream services (1-5 services)
          fc.array(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 20 }),
              url: fc.webUrl(),
              healthPath: fc.constantFrom("/health", "/status", "/ping"),
            }),
            { minLength: 1, maxLength: 5 }
          ),
          // Generate random response delays (0-100ms)
          fc.array(fc.integer({ min: 0, max: 100 }), { minLength: 1, maxLength: 5 }),
          async (services: ServiceEndpoint[], delays: number[]) => {
            // Reset mock before each property test run
            (global.fetch as jest.Mock).mockClear();
            
            // Mock fetch to return successful responses with delays
            (global.fetch as jest.Mock).mockImplementation(() => {
              const delay = delays[Math.min((global.fetch as jest.Mock).mock.calls.length - 1, delays.length - 1)];
              return new Promise((resolve) => {
                setTimeout(() => {
                  resolve({
                    ok: true,
                    status: 200,
                  });
                }, delay);
              });
            });

            const result = await checkAllServices(services);

            // Verify each successful service has latency measurement
            if (result.upstreams) {
              for (const service of services) {
                const serviceHealth = result.upstreams[service.name];
                if (serviceHealth && serviceHealth.status === "ok") {
                  expect(serviceHealth.latency).toBeDefined();
                  expect(typeof serviceHealth.latency).toBe("number");
                  expect(serviceHealth.latency).toBeGreaterThanOrEqual(0);
                }
              }
            }
          }
        ),
        { numRuns: 20 }
      );
    });
  });

  describe("Property 5: Health check returns 503 for unhealthy services", () => {
    /**
     * Feature: api-gateway-refactoring, Property 5: Health check returns 503 for unhealthy services
     * Validates: Requirements 4.5
     */
    it("should return error status when any service is unhealthy", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random lists of upstream services (2-5 services) with unique names
          fc.uniqueArray(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 20 }),
              url: fc.webUrl(),
              healthPath: fc.constantFrom("/health", "/status", "/ping"),
            }),
            { minLength: 2, maxLength: 5, selector: (service) => service.name }
          ),
          async (services: ServiceEndpoint[]) => {
            // Generate health flags matching the number of services
            const healthyFlags = services.map((_, index) => index !== 0); // First service is unhealthy
            
            // Reset mock before each property test run
            (global.fetch as jest.Mock).mockClear();
            
            // Mock fetch to return responses based on health flags
            let callIndex = 0;
            (global.fetch as jest.Mock).mockImplementation(() => {
              const isHealthy = healthyFlags[callIndex];
              callIndex++;
              
              if (isHealthy) {
                return Promise.resolve({
                  ok: true,
                  status: 200,
                });
              } else {
                return Promise.reject(new Error("Service unavailable"));
              }
            });

            const result = await checkAllServices(services);

            // Verify overall status is error when any service is unhealthy
            expect(result.status).toBe("error");
            
            // Verify at least one service is marked as error
            if (result.upstreams) {
              const hasErrorService = Object.values(result.upstreams).some(
                (service) => service.status === "error"
              );
              expect(hasErrorService).toBe(true);
            }
          }
        ),
        { numRuns: 20 }
      );
    });
  });

  describe("Property 6: Health check includes timestamps", () => {
    /**
     * Feature: api-gateway-refactoring, Property 6: Health check includes timestamps
     * Validates: Requirements 4.6
     */
    it("should include valid ISO 8601 timestamps in all responses", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random lists of upstream services (1-5 services)
          fc.array(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 20 }),
              url: fc.webUrl(),
              healthPath: fc.constantFrom("/health", "/status", "/ping"),
            }),
            { minLength: 1, maxLength: 5 }
          ),
          // Generate random health statuses
          fc.array(fc.boolean(), { minLength: 1, maxLength: 5 }),
          async (services: ServiceEndpoint[], healthyFlags: boolean[]) => {
            // Reset mock before each property test run
            (global.fetch as jest.Mock).mockClear();
            
            // Mock fetch to return responses based on health flags
            let callIndex = 0;
            (global.fetch as jest.Mock).mockImplementation(() => {
              const isHealthy = healthyFlags[Math.min(callIndex, healthyFlags.length - 1)];
              callIndex++;
              
              if (isHealthy) {
                return Promise.resolve({
                  ok: true,
                  status: 200,
                });
              } else {
                return Promise.reject(new Error("Service unavailable"));
              }
            });

            const result = await checkAllServices(services);

            // Verify timestamp exists
            expect(result.timestamp).toBeDefined();
            
            // Verify timestamp is a valid ISO 8601 string
            const timestamp = new Date(result.timestamp);
            expect(timestamp.toISOString()).toBe(result.timestamp);
            
            // Verify timestamp is recent (within last 5 seconds)
            const now = Date.now();
            const timestampMs = timestamp.getTime();
            expect(now - timestampMs).toBeLessThan(5000);
          }
        ),
        { numRuns: 20 }
      );
    });
  });

  describe("Property 7: Health check handles timeouts gracefully", () => {
    /**
     * Feature: api-gateway-refactoring, Property 7: Health check handles timeouts gracefully
     * Validates: Requirements 4.7
     */
    it("should mark timed-out services as unhealthy without crashing", async () => {
      await fc.assert(
        fc.asyncProperty(
          // Generate random lists of upstream services (1-2 services for faster tests)
          fc.uniqueArray(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 20 }),
              url: fc.webUrl(),
              healthPath: fc.constantFrom("/health", "/status", "/ping"),
            }),
            { minLength: 1, maxLength: 2, selector: (service) => service.name }
          ),
          async (services: ServiceEndpoint[]) => {
            // Reset mock before each property test run
            (global.fetch as jest.Mock).mockClear();
            
            // Mock fetch to reject immediately with an AbortError (simulating timeout)
            (global.fetch as jest.Mock).mockRejectedValue(
              new Error("The operation was aborted")
            );

            const startTime = Date.now();
            const result = await checkAllServices(services);
            const duration = Date.now() - startTime;

            // Verify health check completes quickly (should not wait for timeout)
            // Should complete within ~6 seconds (5s timeout + 1s buffer)
            expect(duration).toBeLessThan(7000);
            
            // Verify all services are marked as error due to timeout
            if (result.upstreams) {
              for (const service of services) {
                const serviceHealth = result.upstreams[service.name];
                expect(serviceHealth).toBeDefined();
                expect(serviceHealth.status).toBe("error");
              }
            }
            
            // Verify overall status is error
            expect(result.status).toBe("error");
          }
        ),
        { numRuns: 20 } // Can use more runs now since we're not waiting for timeouts
      );
    });
  });

  describe("Unit Tests", () => {
    describe("checkAllServices", () => {
      it("should return 200 status when all services are healthy", async () => {
        // Mock fetch to return successful responses
        (global.fetch as jest.Mock).mockResolvedValue({
          ok: true,
          status: 200,
        });

        const services: ServiceEndpoint[] = [
          {
            name: "auth-service",
            url: "http://localhost:3001",
            healthPath: "/health",
          },
          {
            name: "notification-service",
            url: "http://localhost:3002",
            healthPath: "/health",
          },
        ];

        const result = await checkAllServices(services);

        expect(result.status).toBe("ok");
        expect(result.service).toBe("api-gateway");
        expect(result.upstreams).toBeDefined();
        expect(result.upstreams!["auth-service"].status).toBe("ok");
        expect(result.upstreams!["notification-service"].status).toBe("ok");
        expect(result.timestamp).toBeDefined();
      });

      it("should return 503 status when one service is unhealthy", async () => {
        // Mock fetch to return mixed responses
        (global.fetch as jest.Mock)
          .mockResolvedValueOnce({
            ok: true,
            status: 200,
          })
          .mockRejectedValueOnce(new Error("Service unavailable"));

        const services: ServiceEndpoint[] = [
          {
            name: "auth-service",
            url: "http://localhost:3001",
            healthPath: "/health",
          },
          {
            name: "notification-service",
            url: "http://localhost:3002",
            healthPath: "/health",
          },
        ];

        const result = await checkAllServices(services);

        expect(result.status).toBe("error");
        expect(result.service).toBe("api-gateway");
        expect(result.upstreams).toBeDefined();
        expect(result.upstreams!["auth-service"].status).toBe("ok");
        expect(result.upstreams!["notification-service"].status).toBe("error");
        expect(result.timestamp).toBeDefined();
      });

      it("should return 503 status when all services are unhealthy", async () => {
        // Mock fetch to reject all requests
        (global.fetch as jest.Mock).mockRejectedValue(
          new Error("Service unavailable")
        );

        const services: ServiceEndpoint[] = [
          {
            name: "auth-service",
            url: "http://localhost:3001",
            healthPath: "/health",
          },
          {
            name: "notification-service",
            url: "http://localhost:3002",
            healthPath: "/health",
          },
        ];

        const result = await checkAllServices(services);

        expect(result.status).toBe("error");
        expect(result.service).toBe("api-gateway");
        expect(result.upstreams).toBeDefined();
        expect(result.upstreams!["auth-service"].status).toBe("error");
        expect(result.upstreams!["notification-service"].status).toBe("error");
        expect(result.timestamp).toBeDefined();
      });

      it("should mark service as unhealthy on timeout", async () => {
        // Mock fetch to reject with abort error (simulating timeout)
        (global.fetch as jest.Mock).mockRejectedValue(
          new Error("The operation was aborted")
        );

        const services: ServiceEndpoint[] = [
          {
            name: "auth-service",
            url: "http://localhost:3001",
            healthPath: "/health",
          },
        ];

        const result = await checkAllServices(services);

        expect(result.status).toBe("error");
        expect(result.upstreams).toBeDefined();
        expect(result.upstreams!["auth-service"].status).toBe("error");
        expect(result.upstreams!["auth-service"].latency).toBeUndefined();
      });

      it("should mark service as unhealthy on network error", async () => {
        // Mock fetch to reject with network error
        (global.fetch as jest.Mock).mockRejectedValue(
          new Error("Network request failed")
        );

        const services: ServiceEndpoint[] = [
          {
            name: "auth-service",
            url: "http://localhost:3001",
            healthPath: "/health",
          },
        ];

        const result = await checkAllServices(services);

        expect(result.status).toBe("error");
        expect(result.upstreams).toBeDefined();
        expect(result.upstreams!["auth-service"].status).toBe("error");
        expect(result.upstreams!["auth-service"].latency).toBeUndefined();
      });
    });
  });
});
