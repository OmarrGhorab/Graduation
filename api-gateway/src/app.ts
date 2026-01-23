import express, { Express } from "express";
import { AppConfig } from "./config/index.js";
import { setupMiddleware } from "./middleware/index.js";
import { setupRoutes } from "./routes/index.js";
import { errorHandler } from "./middleware/error.middleware.js";

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
export function createApp(config: AppConfig): Express {
  const app = express();

  // Set up middleware (compression, timeout, body parsing, CORS, Arcjet)
  setupMiddleware(app, config);

  // Set up routes (health check, proxy routes)
  setupRoutes(app, config);

  // Add error handler (must be last)
  app.use(errorHandler);

  return app;
}
