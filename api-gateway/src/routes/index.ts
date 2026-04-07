import { Express, Request, Response } from "express";
import proxy from "express-http-proxy";
import { createProxyMiddleware } from "http-proxy-middleware";
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
  ws: 0,
  courses: 0,
  payment: 0,
  recommendation: 0,
};

/**
 * Gets the next service URL using round-robin load balancing
 */
function getNextServiceUrl(services: ServiceEndpoint[], serviceKey: string): string {
  if (!services || services.length === 0) {
    console.error(`[LoadBalancer] No services configured for: ${serviceKey}`);
    throw new Error(`No ${serviceKey} services configured`);
  }

  const index = serviceIndexes[serviceKey] ?? 0;
  const service = services[index];

  if (!service) {
    console.error(`[LoadBalancer] Service at index ${index} is undefined for: ${serviceKey}`, {
      servicesCount: services.length,
      index
    });
    // Fallback to first instance
    return services[0].url;
  }

  const url = service.url;
  serviceIndexes[serviceKey] = (index + 1) % services.length;
  console.log(`[LoadBalancer] Routing ${serviceKey} to: ${url}`);
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
export function setupRoutes(app: Express, config: AppConfig): { wsProxy: any } {
  // Health check endpoint (not proxied)
  app.get("/health", async (req: Request, res: Response) => {
    try {
      // Flatten all service instances for health checking
      const services = [
        ...config.services.auth,
        ...config.services.notification,
        ...config.services.chat,
        ...config.services.courses,
        ...config.services.payment,
        ...config.services.recommendation,
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

  // WebSocket Gateway routes (WS Upgrade + HTTP)
  console.log(`[Routes] Initializing WS Proxy for path /ws`);
  const wsProxy = createProxyMiddleware({
    target: getNextServiceUrl(config.services.ws, "ws"),
    ws: true,
    changeOrigin: true,
    secure: false, // For local development
    on: {
      proxyReqWs: (proxyReq, req, socket, options, head) => {
        console.log(`[WS Proxy] Proxying upgrade request for: ${req.url}`);
        // Ensure some headers are set correctly for the internal gateway
        proxyReq.setHeader('X-Forwarded-Proto', 'http');
      },
      error: (err, req: any, res: any) => {
        // Handle both HTTP and WS errors
        console.error(`[WS Proxy] Error:`, err.message);
        if (res.writeHead && !res.headersSent) {
          res.writeHead(502);
          res.end("Bad Gateway (WS Proxy Error)");
        }
      },
      open: (proxySocket) => {
        console.log(`[WS Proxy] Connection opened to target`);
      },
      close: (res, socket, head) => {
        console.log(`[WS Proxy] Connection closed`);
      },
    }
  });

  // Verify route hits
  app.get("/ws-test", (req, res) => {
    res.send("WS Path is reachable");
  });

  // Courses & Attendance service routes
  const coursePaths = [
    "/api/v1/courses",
    "/api/v1/subjects",
    "/api/v1/lessons",
    "/api/v1/attendance",
    "/api/v1/absences",
    "/api/v1/progress",
    "/api/v1/calendar",
    "/api/v1/internal",
    "/api/v1/watch",
  ];


  coursePaths.forEach(path => {
    app.use(
      path,
      (req, res, next) => {
        console.log(`[Proxy] Routing ${req.method} ${req.originalUrl} to Courses Service`);
        next();
      },
      proxy(() => getNextServiceUrl(config.services.courses, "courses"), {
        proxyReqPathResolver: (req) => req.originalUrl,
        parseReqBody: false, // Let the underlying service handle the body
        limit: "2gb" // Set limit for the proxy as well
      })
    );
  });

  app.use("/ws", wsProxy);

  // Payment service routes
  const paymentPaths = ["/api/v1/payments", "/api/v1/cart", "/api/v1/subscriptions"];
  paymentPaths.forEach(path => {
    app.use(
      path,
      // Skip authentication for the Paymob webhook and the redirect status page
      (req, res, next) => {
        const isWebhook = req.path.includes("/webhook/paymob");
        const isStatus = req.path.includes("/payments/status");
        if (isWebhook || isStatus) {
            return next();
        }
        // Otherwise, run normal auth (checkAuth should be here if it's not global)
        next();
      },
      proxy(() => getNextServiceUrl(config.services.payment, "payment"), {
        proxyReqPathResolver: (req) => req.originalUrl,
        parseReqBody: false, // Essential for Paymob Webhook HMAC validation
      })
    );
  });

  // Recommendation service routes
  app.use(
    "/api/v1/recommendations",
    proxy(() => getNextServiceUrl(config.services.recommendation, "recommendation"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  // Auth service with Pre-warming trigger on login
  app.use(
    "/api/v1/auth",
    proxy(() => getNextServiceUrl(config.services.auth, "auth"), {
      proxyReqPathResolver: (req) => req.originalUrl,
      userResDecorator: (proxyRes, proxyResData, userReq, userRes) => {
        // Trigger pre-warming ONLY on successful login
        if (userReq.path.includes("/login") && proxyRes.statusCode === 200) {
          try {
            const data = JSON.parse(proxyResData.toString('utf8'));
            const token = data.data?.accessToken;
            
            if (token) {
              const recUrl = getNextServiceUrl(config.services.recommendation, "recommendation");
              console.log(`[PreWarming] Triggering background AI warming for token...`);
              
              // Fire and forget (don't await)
              fetch(`${recUrl}/api/v1/recommendations`, {
                headers: { 'Authorization': `Bearer ${token}` }
              }).catch((err: any) => console.error(`[PreWarming] Failed: ${err.message}`));
            }
          } catch (e: any) {
            console.error(`[PreWarming] Error: ${e.message}`);
          }
        }
        return proxyResData;
      }
    })
  );

  // Auth service catch-all (load balanced, must be last)
  app.use(
    "/",
    proxy(() => getNextServiceUrl(config.services.auth, "auth"), {
      proxyReqPathResolver: (req) => req.originalUrl,
    })
  );

  return { wsProxy };
}
