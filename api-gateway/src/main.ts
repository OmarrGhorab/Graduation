import http from "http";
import { loadConfig } from "./config/index.js";
import { createApp } from "./app.js";

/**
 * Main entry point for the API Gateway application.
 * 
 * This function:
 * - Loads and validates configuration from environment variables
 * - Creates and configures the Express application
 * - Starts the HTTP server on the configured port
 * - Sets up graceful shutdown handlers for SIGTERM and SIGINT
 * 
 * If configuration loading or server startup fails, the process exits with code 1.
 * 
 * @returns Promise that resolves when the server is running
 * 
 * @example
 * ```typescript
 * // Called automatically when the module is executed
 * main();
 * ```
 */
async function main(): Promise<void> {
  try {
    // Load and validate configuration
    const config = loadConfig();

    // Create and configure the Express application
    const { app, wsProxy } = createApp(config);

    // Create HTTP server explicitly
    const server = http.createServer(app);

    // Configure server timeouts for large file uploads (2GB+)
    // Set timeout to 15 minutes (900,000ms)
    server.timeout = 900000;
    // Set headers timeout and keep-alive to be slightly higher than normal to handle slow clients
    server.headersTimeout = 905000;
    server.keepAliveTimeout = 900000;

    // Start server
    server.listen(config.server.port, () => {
      console.log(
        `api-gateway is running on port ${config.server.port} (${config.server.nodeEnv})`
      );
    });

    // Handle WebSocket upgrade manually for the proxy
    server.on("upgrade", (req, socket, head) => {
      const url = req.url || "";
      console.log(`[Server] Upgrade request received: ${url}`);

      if (url.startsWith("/ws")) {
        console.log(`[Server] Routing upgrade to wsProxy`);
        wsProxy.upgrade(req, socket, head);
      } else {
        console.log(`[Server] Unknown upgrade path: ${url}`);
        socket.destroy();
      }
    });

    // Keep process alive if it tries to exit
    process.stdin.resume();

    // Handle graceful shutdown
    const shutdown = () => {
      console.log("Shutting down API Gateway...");
      server.close(() => {
        console.log("HTTP server closed");
        process.exit(0);
      });
    };

    process.on("SIGTERM", shutdown);
    process.on("SIGINT", shutdown);
  } catch (error) {
    console.error("Failed to start API Gateway:", error);
    process.exit(1);
  }
}

// Start the application
main();
