import { ServiceEndpoint } from "../config/index.js";

/**
 * Health status for an individual upstream service.
 * 
 * @property name - Service name (e.g., "auth-service", "notification-service")
 * @property status - Health status: "ok" if healthy, "error" if unhealthy
 * @property latency - Response time in milliseconds (only present if status is "ok")
 */
export interface ServiceHealth {
  name: string;
  status: "ok" | "error";
  latency?: number;
}

/**
 * Overall health check response for the API Gateway.
 * 
 * @property status - Overall status: "ok" if all healthy, "error" if any unhealthy
 * @property service - Service name (always "api-gateway")
 * @property upstreams - Health status of each upstream service (optional)
 * @property timestamp - ISO 8601 timestamp of the health check
 * @property error - Error message if health check failed (optional)
 */
export interface HealthCheckResponse {
  status: "ok" | "degraded" | "error";
  service: string;
  upstreams?: Record<string, ServiceHealth>;
  timestamp: string;
  error?: string;
}

/**
 * Checks the health of a single upstream service.
 * 
 * Makes an HTTP GET request to the service's health endpoint and measures
 * response time. Returns error status if the service is unreachable, times out,
 * or returns a non-2xx status code.
 * 
 * @param url - The full URL of the service's health endpoint
 * @param name - The name of the service for identification
 * @param timeoutMs - Timeout in milliseconds (default: 5000ms = 5 seconds)
 * @returns Promise resolving to ServiceHealth object with status and latency
 * 
 * @example
 * ```typescript
 * const health = await checkServiceHealth(
 *   'http://localhost:6001/health',
 *   'auth-service',
 *   5000
 * );
 * console.log(health); // { name: 'auth-service', status: 'ok', latency: 45 }
 * ```
 */
export async function checkServiceHealth(
  url: string,
  name: string,
  timeoutMs: number = 5000
): Promise<ServiceHealth> {
  const startTime = Date.now();

  try {
    // Create an AbortController for timeout
    // This allows us to cancel the fetch request if it takes too long
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    // Make the health check request
    const response = await fetch(url, {
      method: "GET",
      signal: controller.signal,
    });

    // Clear the timeout since the request completed
    clearTimeout(timeoutId);

    // Calculate latency in milliseconds
    const latency = Date.now() - startTime;

    // Check if response is successful (2xx status code)
    if (response.ok) {
      return {
        name,
        status: "ok",
        latency,
      };
    } else {
      // Service returned an error status code
      return {
        name,
        status: "error",
      };
    }
  } catch (error) {
    // Handle timeout, network errors, or other failures
    // All failures result in error status without latency
    return {
      name,
      status: "error",
    };
  }
}

/**
 * Checks the health of all configured upstream services in parallel.
 * 
 * This function:
 * - Checks all services concurrently for faster results
 * - Returns 200 status if all services are healthy
 * - Returns 503 status if any service is unhealthy
 * - Includes individual service health and latency in the response
 * - Handles errors gracefully without crashing
 * 
 * @param services - Array of service endpoints to check
 * @returns Promise resolving to HealthCheckResponse with overall status and individual service statuses
 * 
 * @example
 * ```typescript
 * const services = [
 *   { name: 'auth-service', url: 'http://localhost:6001', healthPath: '/health' },
 *   { name: 'notification-service', url: 'http://localhost:6003', healthPath: '/health' }
 * ];
 * const health = await checkAllServices(services);
 * console.log(health.status); // 'ok' or 'error'
 * ```
 */
export async function checkAllServices(
  services: ServiceEndpoint[]
): Promise<HealthCheckResponse> {
  try {
    // Check all services in parallel for faster results
    // Using Promise.all ensures we wait for all checks to complete
    // but they execute concurrently rather than sequentially
    const healthChecks = services.map((service) =>
      checkServiceHealth(
        `${service.url}${service.healthPath}`,
        service.name,
        5000
      )
    );

    const results = await Promise.all(healthChecks);

    // Build upstreams object from results
    const upstreams: Record<string, ServiceHealth> = {};
    let allHealthy = true;

    for (const result of results) {
      upstreams[result.name] = result;
      if (result.status === "error") {
        allHealthy = false;
      }
    }

    // Determine overall status based on upstream health
    // If any service is unhealthy, the overall status is error
    const status = allHealthy ? "ok" : "error";

    return {
      status,
      service: "api-gateway",
      upstreams,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    // Handle unexpected errors during health check execution
    // This should rarely happen since individual service checks handle their own errors
    return {
      status: "error",
      service: "api-gateway",
      timestamp: new Date().toISOString(),
      error: error instanceof Error ? error.message : "Unknown error",
    };
  }
}
