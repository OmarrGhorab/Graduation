import { ServerConfig, ProxyConfig, ArcjetConfig } from "../types/index.js";
import { ArcjetBotCategory } from "@arcjet/node";
import dotenv from "dotenv";

dotenv.config();

// Default server configuration
export const DEFAULT_CONFIG: ServerConfig = {
  host: "localhost",
  port: 3000,
};

// CORS configuration
export const CORS_CONFIG = {
  origins: ["http://localhost:3000"],
  allowedHeaders: ["Authorization", "Content-Type"],
  credentials: true,
};

// Request limits
export const REQUEST_LIMIT = "10mb";

// Proxy configuration
export const PROXY_CONFIG: ProxyConfig = {
  authServiceUrl: "http://localhost:6001",
  notificationServiceUrl: "http://localhost:6003",
  timeout: 30000, // 30 seconds
};

// Arcjet configuration constants
export const ARCJET_CONFIG: ArcjetConfig = {
  shieldMode: process.env.NODE_ENV === "production" ? "LIVE" : "DRY_RUN",
  botDetectionMode: process.env.NODE_ENV === "production" ? "LIVE" : "DRY_RUN",
  allowedBotCategories: [
    "CATEGORY:SEARCH_ENGINE", // Google, Bing, etc
    "CATEGORY:MONITOR", // Uptime monitoring services
    "CATEGORY:PREVIEW", // Link previews e.g. Slack, Discord
  ] as const satisfies readonly ArcjetBotCategory[],
  rateLimitConfig: {
    refillRate: 5,
    interval: 10,
    capacity: 10,
  },
  tokensRequested: 5,
} as const;

/**
 * Get server configuration from environment variables
 */
export const getServerConfig = (): ServerConfig => ({
  host: process.env.HOST || DEFAULT_CONFIG.host,
  port: parseInt(process.env.PORT || DEFAULT_CONFIG.port.toString(), 10),
});

/**
 * Get Arcjet key from environment variables
 */
export const getArcjetKey = (): string => {
  return process.env.ARCJET_KEY || "";
};
