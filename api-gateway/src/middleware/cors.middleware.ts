import { Request, Response, NextFunction } from "express";
import { CorsConfig } from "../config/index.js";

/**
 * Creates CORS (Cross-Origin Resource Sharing) middleware based on configuration.
 * 
 * This middleware handles CORS headers for cross-origin requests, supporting:
 * - Requests with no origin (mobile apps, Postman, curl)
 * - Wildcard origins (*) for development environments
 * - Whitelisted origins for production
 * - Preflight OPTIONS requests
 * 
 * @param config - CORS configuration with allowed origins and headers
 * @returns Express middleware function that handles CORS
 * 
 * @example
 * ```typescript
 * const corsMiddleware = createCorsMiddleware({
 *   allowedOrigins: ['https://example.com'],
 *   credentials: true,
 *   allowedHeaders: ['Content-Type', 'Authorization']
 * });
 * app.use(corsMiddleware);
 * ```
 */
export function createCorsMiddleware(config: CorsConfig) {
  return (req: Request, res: Response, next: NextFunction) => {
    const origin = req.headers.origin as string | undefined;

    // Allow requests with no origin (mobile apps, Postman, curl)
    // Also bypass CORS for the WebSocket path which uses Token-based auth
    if (!origin || req.path === "/ws") {
      res.setHeader("Access-Control-Allow-Origin", "*");
      res.setHeader("Access-Control-Allow-Credentials", "true");
      res.setHeader(
        "Access-Control-Allow-Headers",
        config.allowedHeaders.join(", ")
      );
      res.setHeader(
        "Access-Control-Allow-Methods",
        "GET, POST, PUT, DELETE, PATCH, OPTIONS"
      );

      // Handle preflight requests
      if (req.method === "OPTIONS") {
        return res.sendStatus(204);
      }

      return next();
    }

    // Support wildcard for development
    if (config.allowedOrigins.includes("*")) {
      res.setHeader("Access-Control-Allow-Origin", origin);
      res.setHeader("Access-Control-Allow-Credentials", "true");
      res.setHeader(
        "Access-Control-Allow-Headers",
        config.allowedHeaders.join(", ")
      );
      res.setHeader(
        "Access-Control-Allow-Methods",
        "GET, POST, PUT, DELETE, PATCH, OPTIONS"
      );

      // Handle preflight requests
      if (req.method === "OPTIONS") {
        return res.sendStatus(204);
      }

      return next();
    }

    // Check origin against whitelist
    if (config.allowedOrigins.includes(origin)) {
      res.setHeader("Access-Control-Allow-Origin", origin);
      res.setHeader("Access-Control-Allow-Credentials", "true");
      res.setHeader(
        "Access-Control-Allow-Headers",
        config.allowedHeaders.join(", ")
      );
      res.setHeader(
        "Access-Control-Allow-Methods",
        "GET, POST, PUT, DELETE, PATCH, OPTIONS"
      );

      // Handle preflight requests
      if (req.method === "OPTIONS") {
        return res.sendStatus(204);
      }

      return next();
    }

    // Origin not allowed
    console.log(`[CORS] Rejected origin: ${origin} for ${req.method} ${req.path}`);
    return res.status(403).json({
      error: "Forbidden",
      message: "Origin not allowed",
      statusCode: 403,
      timestamp: new Date().toISOString(),
    });
  };
}
