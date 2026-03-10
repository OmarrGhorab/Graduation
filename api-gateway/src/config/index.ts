import dotenv from "dotenv";

// Load environment variables
dotenv.config();

/**
 * Server configuration for the API Gateway.
 * 
 * @property port - Port number the server listens on (1-65535)
 * @property nodeEnv - Node environment: development, production, or test
 * @property isProd - Convenience flag indicating if running in production
 */
export interface ServerConfig {
  port: number;
  nodeEnv: "development" | "production" | "test";
  isProd: boolean;
}

/**
 * CORS (Cross-Origin Resource Sharing) configuration.
 * 
 * @property allowedOrigins - List of allowed origin URLs or ["*"] for wildcard
 * @property credentials - Whether to allow credentials (cookies, authorization headers)
 * @property allowedHeaders - List of allowed HTTP headers in CORS requests
 */
export interface CorsConfig {
  allowedOrigins: string[];
  credentials: boolean;
  allowedHeaders: string[];
}

/**
 * Configuration for an upstream service endpoint.
 * 
 * @property name - Human-readable service name (e.g., "auth-service")
 * @property url - Base URL of the service (e.g., "http://localhost:6001")
 * @property healthPath - Path to the service's health check endpoint (e.g., "/health")
 */
export interface ServiceEndpoint {
  name: string;
  url: string;
  healthPath: string;
}

/**
 * Configuration for all upstream services that the gateway proxies to.
 * All services support multiple instances for load balancing.
 * 
 * @property auth - Authentication service instances
 * @property notification - Notification service instances
 * @property chat - Chat service instances
 */
export interface ServicesConfig {
  auth: ServiceEndpoint[];
  notification: ServiceEndpoint[];
  chat: ServiceEndpoint[];
  ws: ServiceEndpoint[];
  courses: ServiceEndpoint[];
  payment: ServiceEndpoint[];
}

/**
 * Security configuration for the API Gateway.
 * 
 * @property arcjetKey - Arcjet API key for bot and VPN protection (optional)
 * @property arcjetEnabled - Whether Arcjet protection is enabled
 */
export interface SecurityConfig {
  arcjetKey?: string;
  arcjetEnabled: boolean;
}

/**
 * Complete application configuration for the API Gateway.
 * 
 * This is the root configuration object that contains all settings needed
 * to run the gateway, including server, CORS, services, and security configuration.
 * 
 * @property server - Server configuration (port, environment)
 * @property cors - CORS configuration (allowed origins, headers)
 * @property services - Upstream service endpoints (all support multiple instances)
 * @property security - Security settings (Arcjet protection)
 */
export interface AppConfig {
  server: ServerConfig;
  cors: CorsConfig;
  services: ServicesConfig;
  security: SecurityConfig;
}

/**
 * Parses comma-separated URLs into ServiceEndpoint array
 */
function parseServiceUrls(urlsString: string, serviceName: string): ServiceEndpoint[] {
  return urlsString
    .split(",")
    .map((url) => url.trim())
    .filter((url) => url.length > 0)
    .map((url, index) => ({
      name: `${serviceName}-${index + 1}`,
      url,
      healthPath: "/health",
    }));
}

/**
 * Validates the application configuration
 * @param config - The configuration to validate
 * @throws Error if configuration is invalid
 */
export function validateConfig(config: AppConfig): void {
  // Validate server configuration
  if (!config.server.port || config.server.port < 1 || config.server.port > 65535) {
    throw new Error("Invalid PORT: must be a number between 1 and 65535");
  }

  const validEnvs = ["development", "production", "test"];
  if (!validEnvs.includes(config.server.nodeEnv)) {
    throw new Error(
      `Invalid NODE_ENV: must be one of ${validEnvs.join(", ")}`
    );
  }

  // Validate CORS configuration
  if (!config.cors.allowedOrigins || config.cors.allowedOrigins.length === 0) {
    throw new Error("Invalid ALLOWED_ORIGINS: must be a non-empty list");
  }

  // Validate service URLs
  const urlPattern = /^https?:\/\/.+/;

  // Validate auth service URLs
  if (config.services.auth.length === 0) {
    throw new Error("At least one AUTH_SERVICE_URL is required");
  }
  for (const service of config.services.auth) {
    if (!urlPattern.test(service.url)) {
      throw new Error(`Invalid AUTH_SERVICE_URL: ${service.url} must be a valid HTTP/HTTPS URL`);
    }
  }

  // Validate notification service URLs
  if (config.services.notification.length === 0) {
    throw new Error("At least one NOTIFICATION_SERVICE_URL is required");
  }
  for (const service of config.services.notification) {
    if (!urlPattern.test(service.url)) {
      throw new Error(`Invalid NOTIFICATION_SERVICE_URL: ${service.url} must be a valid HTTP/HTTPS URL`);
    }
  }

  // Validate chat service URLs
  if (config.services.chat.length === 0) {
    throw new Error("At least one CHAT_SERVICE_URL is required");
  }
  for (const service of config.services.chat) {
    if (!urlPattern.test(service.url)) {
      throw new Error(`Invalid CHAT_SERVICE_URL: ${service.url} must be a valid HTTP/HTTPS URL`);
    }
  }

  // Validate courses service URLs
  if (config.services.courses.length === 0) {
    throw new Error("At least one COURSES_SERVICE_URL is required");
  }
  for (const service of config.services.courses) {
    if (!urlPattern.test(service.url)) {
      throw new Error(`Invalid COURSES_SERVICE_URL: ${service.url} must be a valid HTTP/HTTPS URL`);
    }
  }

  // Validate payment service URLs
  if (config.services.payment.length === 0) {
    throw new Error("At least one PAYMENT_SERVICE_URL is required");
  }
  for (const service of config.services.payment) {
    if (!urlPattern.test(service.url)) {
      throw new Error(`Invalid PAYMENT_SERVICE_URL: ${service.url} must be a valid HTTP/HTTPS URL`);
    }
  }
}

/**
 * Loads and validates application configuration from environment variables
 * @returns The validated application configuration
 * @throws Error if required configuration is missing or invalid
 */
export function loadConfig(): AppConfig {
  // Parse PORT
  const portStr = process.env.PORT;
  if (!portStr) {
    throw new Error("Missing required environment variable: PORT");
  }
  const port = parseInt(portStr, 10);
  if (isNaN(port)) {
    throw new Error("Invalid PORT: must be a valid number");
  }

  // Parse NODE_ENV
  const nodeEnv = (process.env.NODE_ENV || "development") as
    | "development"
    | "production"
    | "test";

  // Parse ALLOWED_ORIGINS
  const allowedOriginsStr = process.env.ALLOWED_ORIGINS;
  if (!allowedOriginsStr) {
    throw new Error("Missing required environment variable: ALLOWED_ORIGINS");
  }
  const allowedOrigins = allowedOriginsStr
    .split(",")
    .map((origin) => origin.trim())
    .filter((origin) => origin.length > 0);

  // Parse service URLs (all support multiple instances)
  const authServiceUrls = process.env.AUTH_SERVICE_URLS || "http://localhost:6001,http://localhost:6011,http://localhost:6021";
  const notificationServiceUrls = process.env.NOTIFICATION_SERVICE_URLS || "http://localhost:6003,http://localhost:6013,http://localhost:6023";
  const chatServiceUrls = process.env.CHAT_SERVICE_URLS || "http://localhost:6004,http://localhost:6014,http://localhost:6024";
  const coursesServiceUrls = process.env.COURSES_SERVICE_URLS || "http://localhost:8085,http://localhost:8086";
  const paymentServiceUrls = process.env.PAYMENT_SERVICE_URLS || "http://localhost:8090";

  // Build configuration object
  const config: AppConfig = {
    server: {
      port,
      nodeEnv,
      isProd: nodeEnv === "production",
    },
    cors: {
      allowedOrigins,
      credentials: true,
      allowedHeaders: [
        "Content-Type",
        "Authorization",
        "x-refresh-token",
        "x-forwarded-for",
      ],
    },
    services: {
      auth: parseServiceUrls(authServiceUrls, "auth-service"),
      notification: parseServiceUrls(notificationServiceUrls, "notification-service"),
      chat: parseServiceUrls(chatServiceUrls, "chat-service"),
      ws: parseServiceUrls(process.env.WS_GATEWAY_URLS || "http://localhost:8001", "ws-gateway"),
      courses: parseServiceUrls(coursesServiceUrls, "courses-service"),
      payment: parseServiceUrls(paymentServiceUrls, "payment-service"),
    },
    security: {
      arcjetKey: process.env.ARCJET_KEY,
      arcjetEnabled: !!process.env.ARCJET_KEY && nodeEnv === "production",
    },
  };

  // Validate the configuration
  validateConfig(config);

  return config;
}
