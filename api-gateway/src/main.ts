import express from "express";
import { getServerConfig, getArcjetKey, PROXY_CONFIG } from "./config/index.js";
import { corsMiddleware, setupBasicMiddleware, globalErrorHandler, notFoundHandler } from "./middleware/index.js";
import { createRootHandler } from "./routes/index.js";

/**
 * Initialize and configure the Express application
 */
const createApp = (): express.Application => {
  const app = express();
  
  // Setup middleware
  app.use(corsMiddleware);
  setupBasicMiddleware(app);
  
  return app;
};

/**
 * Setup routes
 */
const setupRoutes = (app: express.Application, arcjetKey: string): void => {
  // Root endpoint with Arcjet protection and proxy to auth service
  app.get("/", createRootHandler(arcjetKey));
  
  // Error handling middleware (must be last)
  app.use(globalErrorHandler);
  app.use(notFoundHandler);
};

/**
 * Start the server
 */
const startServer = (app: express.Application, config: ReturnType<typeof getServerConfig>, arcjetKey: string): void => {
  app.listen(config.port, config.host, (): void => {
    console.log(`🚀 Server is running on http://${config.host}:${config.port}`);
    console.log(`📝 Environment: ${process.env.NODE_ENV || "development"}`);
    console.log(`🔄 Proxy: Root endpoint (/) → ${PROXY_CONFIG.authServiceUrl}`);
    
    if (!arcjetKey) {
      console.log(`Arcjet protection: DISABLED (set ARCJET_KEY to enable)`);
    } else {
      console.log(`Arcjet protection: ENABLED`);
    }
  });
};

/**
 * Setup graceful shutdown handlers
 */
const setupGracefulShutdown = (): void => {
  process.on('SIGTERM', (): void => {
    console.log('SIGTERM received, shutting down gracefully');
    process.exit(0);
  });

  process.on('SIGINT', (): void => {
    console.log('SIGINT received, shutting down gracefully');
    process.exit(0);
  });
};

/**
 * Main application entry point
 */
const main = (): void => {
  // Get configuration
  const config = getServerConfig();
  const arcjetKey = getArcjetKey();
  
  // Create and configure app
  const app = createApp();
  setupRoutes(app, arcjetKey);
  
  // Setup graceful shutdown
  setupGracefulShutdown();
  
  // Start server
  startServer(app, config, arcjetKey);
};

// Start the application
main();
