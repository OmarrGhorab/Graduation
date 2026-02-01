import { Express, Request, Response } from "express";
import proxy from "express-http-proxy";
import { AppConfig, ServiceEndpoint } from "../config/index.js";
import { checkAllServices } from "../services/health.service.js";

/**
 * Proxy route configuration interface.
 * 
 * @property path - The route path to match (e.g., "/api/v1/notifications")
 * @property target - The upstream service URL to proxy to
 * @property preservePath - Whether to preserve the original request path (optional)
 */
export interface ProxyRoute {
  path: string;
  target: string;
  preservePath?: boolean;
}

// Round-robin counters for load balancing
const serviceIndexes: Record<string, number> = {
  auth: 0,
  notification: 0,
  chat: 0,
};

/**
 * Gets the next service URL using round-robin load balancing
 */
function getNextServiceUrl(services: ServiceEndpoint[], serviceKey: string): string {
  if (services.length === 0) {
    throw new Error(`No ${serviceKey} services configured`);
  }
  const url = services[serviceIndexes[serviceKey]].url;
  serviceIndexes[serviceKey] = (serviceIndexes[serviceKey] + 1) % services.length;
  return url;
}

/**
 * Sets up all routes for the API Gateway.
 * 
 * Routes are applied in priority order (most specific first):
 * 1. /health - Gateway health check endpoint (not proxied)
 * 2. /api/v1/conversations - Proxied to chat service (load balanced)
 * 3. /api/v1/typing - Proxied to chat service (load balanced)
 * 4. /api/v1/media - Proxied to chat service (load balanced)
 * 5. /api/v1/notifications - Proxied to notification service (load balanced)
 * 6. /api/v1/location/request - Proxied to notification service (load balanced)
 * 7. / - Catch-all proxied to auth service (load balanced)
 * 
 * All proxy routes preserve the original request path when forwarding to upstream services.
 * All services support multiple instances with round-robin load balancing.
 * 
 * @param app - Express application instance to configure
 * @param config - Application configuration containing service endpoints
 * @returns void
 */
export function setupRoutes(app: Express, config: AppConfig): void {
  // Health check endpoint (not proxied)
  app.get("/health", async (req: Request, res: Response) => {
    try {
      // Flatten all service instances for health checking
      const services = [
        ...config.services.auth,
        ...config.services.notification,
        ...config.services.chat,
      ];
      const healthCheck = await checkAllServices(services);

      // Return 200 if all healthy, 503 if any unhealthy
      const statusCode = healthCheck.status === "ok" ? 200 : 503;
      res.status(statusCode).json(healthCheck);
    } catch (error) {
      res.status(503).json({
        status: "error",
        service: "api-gateway",
        error: error instanceof Error ? error.message : "Health check failed",
        timestamp: new Date().toISOString(),
      });
    }
  });

  // Chat service routes (load balanced)
  app.use(
    "/api/v1/conversations",
    proxy(() => getNextServiceUrl(config.services.chat, "chat"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/typing",
    proxy(() => getNextServiceUrl(config.services.chat, "chat"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/media",
    proxy(() => getNextServiceUrl(config.services.chat, "chat"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  // Notification service routes (load balanced)
  app.use(
    "/api/v1/notifications",
    proxy(() => getNextServiceUrl(config.services.notification, "notification"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/location/request",
    proxy(() => getNextServiceUrl(config.services.notification, "notification"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  // Auth service catch-all (load balanced, must be last)
  app.use(
    "/",
    proxy(() => getNextServiceUrl(config.services.auth, "auth"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );
}
