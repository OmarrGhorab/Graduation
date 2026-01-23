import { Express, json, urlencoded } from "express";
import { AppConfig } from "../config/index.js";
import { createCompressionMiddleware } from "./compression.middleware.js";
import { createTimeoutMiddleware } from "./timeout.middleware.js";
import { createCorsMiddleware } from "./cors.middleware.js";
import { createArcjetMiddleware } from "./arcjet.middleware.js";

/**
 * Sets up all middleware in the correct order for the API Gateway.
 * 
 * Middleware is applied in a specific order to ensure proper request processing:
 * 1. Compression - Compresses responses early to reduce bandwidth
 * 2. Timeout - Sets request timeout limits
 * 3. Body parsing - Parses JSON and URL-encoded request bodies
 * 4. CORS - Handles Cross-Origin Resource Sharing
 * 5. Arcjet - Provides bot and VPN protection
 * 
 * Halt-on-timeout checks are inserted between body parsers to prevent
 * processing timed-out requests.
 * 
 * @param app - Express application instance to configure
 * @param config - Application configuration containing CORS and security settings
 * @returns void
 * 
 * @example
 * ```typescript
 * const app = express();
 * const config = loadConfig();
 * setupMiddleware(app, config);
 * ```
 */
export function setupMiddleware(app: Express, config: AppConfig): void {
  // Middleware ordering is critical for correct request processing
  // Each middleware is positioned to maximize efficiency and security
  
  // 1. Compression (should be early to compress all responses)
  // Applied first so all subsequent responses can be compressed
  app.use(createCompressionMiddleware());

  // 2. Timeout (set timeout for all requests)
  // Applied early to ensure all requests have a timeout limit
  const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(30000);
  app.use(timeoutMiddleware);

  // 3. Body parsing with halt-on-timeout checks between parsers
  // Parse JSON bodies (up to 10MB)
  app.use(json({ limit: "10mb" }));
  app.use(haltOnTimeout); // Check timeout after JSON parsing to prevent processing timed-out requests
  
  // Parse URL-encoded bodies (up to 10MB)
  app.use(urlencoded({ extended: true, limit: "10mb" }));
  app.use(haltOnTimeout); // Check timeout after URL-encoded parsing

  // 4. CORS (after body parsing, before business logic)
  // Applied after body parsing so we can read request bodies if needed
  // Applied before routes to ensure CORS headers are set for all responses
  app.use(createCorsMiddleware(config.cors));

  // 5. Arcjet protection (after CORS, before routes)
  // Applied last in middleware chain (before routes) to protect all endpoints
  // Positioned after CORS so preflight requests are handled correctly
  app.use(
    createArcjetMiddleware({
      key: config.security.arcjetKey,
      enabled: config.security.arcjetEnabled,
    })
  );
}
