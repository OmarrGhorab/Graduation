import arcjet, { detectBot, shield } from "@arcjet/node";
import { Request, Response, NextFunction } from "express";

/**
 * Arcjet security configuration interface.
 * 
 * @property key - Arcjet API key (optional, required if enabled is true)
 * @property enabled - Whether Arcjet protection is enabled
 * @property mode - Protection mode: LIVE blocks requests, DRY_RUN logs only (default: LIVE)
 */
export interface ArcjetConfig {
  key?: string;
  enabled: boolean;
  mode?: "LIVE" | "DRY_RUN";
}

/**
 * Creates Arcjet protection middleware for bot and VPN detection.
 * 
 * This middleware provides security protection against:
 * - Malicious bots (while allowing search engines and monitors)
 * - VPN connections
 * - Proxy servers
 * - Hosting provider IPs
 * - Relay services
 * 
 * If protection is disabled or no API key is provided, returns a no-op middleware.
 * On errors, the middleware fails open (allows the request) to prevent service disruption.
 * 
 * @param config - Arcjet configuration with API key and mode
 * @returns Express middleware function that provides bot and VPN protection
 * 
 * @example
 * ```typescript
 * const arcjetMiddleware = createArcjetMiddleware({
 *   key: process.env.ARCJET_KEY,
 *   enabled: true,
 *   mode: 'LIVE'
 * });
 * app.use(arcjetMiddleware);
 * ```
 */
export function createArcjetMiddleware(config: ArcjetConfig) {
  // Skip protection if disabled or no key provided
  // This allows the gateway to run without Arcjet in development or when the key is not configured
  if (!config.enabled || !config.key) {
    return (req: Request, res: Response, next: NextFunction) => {
      next();
    };
  }

  // Initialize Arcjet with bot detection and shield
  // Bot detection: Blocks malicious bots while allowing legitimate ones (search engines, monitors)
  // Shield: Blocks requests from VPNs, proxies, hosting providers, and relay services
  const aj = arcjet({
    key: config.key,
    rules: [
      // Detect and block bots, but allow search engines and monitoring services
      detectBot({
        mode: config.mode || "LIVE",
        allow: [
          "CATEGORY:SEARCH_ENGINE", // Allow Google, Bing, etc.
          "CATEGORY:MONITOR", // Allow uptime monitors
        ],
      }),
      // Block VPN, proxy, hosting, and relay IPs
      shield({
        mode: config.mode || "LIVE",
      }),
    ],
  });

  return async (req: Request, res: Response, next: NextFunction) => {
    try {
      // Evaluate the request against Arcjet rules
      const decision = await aj.protect(req);

      // Log blocking decisions for security monitoring and debugging
      if (decision.isDenied()) {
        console.log("Arcjet blocked request:", {
          ip: decision.ip,
          reason: decision.reason,
          ruleResults: decision.results,
          timestamp: new Date().toISOString(),
        });

        // Return 403 for blocked requests
        return res.status(403).json({
          error: "Forbidden",
          message: "Request blocked by security policy",
          statusCode: 403,
          timestamp: new Date().toISOString(),
        });
      }

      // Allow request if not denied
      next();
    } catch (error) {
      // Fail open on errors - allow the request to proceed
      // This prevents Arcjet issues from taking down the entire gateway
      console.error("Arcjet error (failing open):", error);
      next();
    }
  };
}
