import { Express, Request, Response } from "express";
import proxy from "express-http-proxy";
import { AppConfig } from "../config/index.js";
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

/**
 * Sets up all routes for the API Gateway.
 * 
 * Routes are applied in priority order (most specific first):
 * 1. /health - Gateway health check endpoint (not proxied)
 * 2. /api/v1/notifications - Proxied to notification service
 * 3. /api/v1/location/request - Proxied to notification service (silent push)
 * 4. / - Catch-all proxied to auth service
 * 
 * All proxy routes preserve the original request path when forwarding to upstream services.
 * 
 * @param app - Express application instance to configure
 * @param config - Application configuration containing service endpoints
 * @returns void
 * 
 * @example
 * ```typescript
 * const app = express();
 * const config = loadConfig();
 * setupRoutes(app, config);
 * ```
 */
export function setupRoutes(app: Express, config: AppConfig): void {
  // Health check endpoint (not proxied)
  app.get("/health", async (req: Request, res: Response) => {
    try {
      const services = [config.services.auth, config.services.notification];
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

  // Notification service routes
  app.use(
    "/api/v1/notifications",
    proxy(config.services.notification.url, {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/location/request",
    proxy(config.services.notification.url, {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  // Auth service catch-all (must be last)
  app.use(
    "/",
    proxy(config.services.auth.url, {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );
}
