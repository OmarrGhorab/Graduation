import express, { Express } from "express";
import { AppConfig } from "./config/index.js";
import { setupMiddleware } from "./middleware/index.js";
import { setupRoutes } from "./routes/index.js";
import { errorHandler } from "./middleware/error.middleware.js";
import { initObservability, setupSentryErrorHandler } from "./observability/index.js";

/**
 * Creates and configures the Express application for the API Gateway.
 * 
 * This function:
 * - Creates a new Express application instance
 * - Sets up all middleware in the correct order
 * - Configures proxy routes to upstream services
 * - Adds centralized error handling
 * 
 * The returned app is ready to be started with app.listen().
 * 
 * @param config - Application configuration containing server, CORS, services, and security settings
 * @returns Configured Express application ready to handle requests
 * 
 * @example
 * ```typescript
 * const config = loadConfig();
 * const app = createApp(config);
 * app.listen(config.server.port);
 * ```
 */
export function createApp(config: AppConfig): { app: Express, wsProxy: any } {
  const app = express();

  // Initialize observability (Tracing, Metrics, Logger, Sentry)
  initObservability(app);

  // Set up middleware (compression, timeout, body parsing, CORS, Arcjet)
  setupMiddleware(app, config);

  // Sentry Debug Endpoint (Must be BEFORE setupRoutes to prevent proxying)
  app.get("/debug-sentry", (req, res) => {
    throw new Error("Sentry Debug Test: API Gateway is connected!");
  });

  // Set up routes (health check, proxy routes)
  const routes = setupRoutes(app, config);

  // Add Sentry error handler (must be before custom error handler)
  setupSentryErrorHandler(app);

  // Add error handler (must be last)
  app.use(errorHandler);

  return { app, wsProxy: routes.wsProxy };
}
