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

    // Create Express app with configuration
    const app = createApp(config);

    // Start HTTP server
    const server = app.listen(config.server.port, () => {
      console.log(
        `api-gateway is running on port ${config.server.port} (${config.server.nodeEnv})`
      );
    });

    // Handle graceful shutdown
    process.on("SIGTERM", () => {
      console.log("SIGTERM signal received: closing HTTP server");
      server.close(() => {
        console.log("HTTP server closed");
      });
    });

    process.on("SIGINT", () => {
      console.log("SIGINT signal received: closing HTTP server");
      server.close(() => {
        console.log("HTTP server closed");
      });
    });
  } catch (error) {
    console.error("Failed to start API Gateway:", error);
    process.exit(1);
  }
}

// Start the application
main();
