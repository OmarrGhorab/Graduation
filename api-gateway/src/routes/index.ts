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

// Round-robin counter for chat service load balancing
let chatServiceIndex = 0;

/**
 * Gets the next chat service URL using round-robin load balancing
 */
function getNextChatServiceUrl(config: AppConfig): string {
  const chatServices = config.services.chat;
  if (chatServices.length === 0) {
    throw new Error("No chat services configured");
  }
  const url = chatServices[chatServiceIndex].url;
  chatServiceIndex = (chatServiceIndex + 1) % chatServices.length;
  return url;
}

/**
 * Sets up all routes for the API Gateway.
 * 
 * Routes are applied in priority order (most specific first):
 * 1. /health - Gateway health check endpoint (not proxied)
 * 2. /api/v1/chat - Proxied to chat service (with load balancing)
 * 3. /api/v1/conversations - Proxied to chat service
 * 4. /api/v1/typing - Proxied to chat service
 * 5. /api/v1/media - Proxied to chat service
 * 6. /api/v1/notifications - Proxied to notification service
 * 7. /api/v1/location/request - Proxied to notification service (silent push)
 * 8. / - Catch-all proxied to auth service
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
      const services = [
        config.services.auth,
        config.services.notification,
        ...config.services.chat
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

  // Chat service routes (load balanced across multiple instances)
  app.use(
    "/api/v1/conversations",
    proxy(() => getNextChatServiceUrl(config), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/typing",
    proxy(() => getNextChatServiceUrl(config), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  app.use(
    "/api/v1/media",
    proxy(() => getNextChatServiceUrl(config), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

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
