import arcjet, { shield, detectBot, tokenBucket, ArcjetDecision } from "@arcjet/node";
import { isSpoofedBot } from "@arcjet/inspect";
import { Response } from "express";
import { ARCJET_CONFIG } from "../config/index.js";
import { sendErrorResponse } from "../utils/responses.js";

/**
 * Initialize Arcjet protection service
 */
export const initializeArcjet = (key: string) => {
  if (!key) {
    console.warn("WARNING: ARCJET_KEY environment variable is not set. Arcjet protection is disabled.");
  }

  return arcjet({
    key,
    rules: [
      // Shield protects your app from common attacks e.g. SQL injection
      shield({ mode: ARCJET_CONFIG.shieldMode }),
      // Create a bot detection rule
      detectBot({
        mode: ARCJET_CONFIG.botDetectionMode, // Blocks requests. Use "DRY_RUN" to log only
        // Block all bots except the following
        allow: [...ARCJET_CONFIG.allowedBotCategories],
      }),
      // Create a token bucket rate limit. Other algorithms are supported.
      tokenBucket({
        mode: ARCJET_CONFIG.shieldMode,
        // Tracked by IP address by default, but this can be customized
        // See https://docs.arcjet.com/fingerprints
        //characteristics: ["ip.src"],
        refillRate: ARCJET_CONFIG.rateLimitConfig.refillRate,
        interval: ARCJET_CONFIG.rateLimitConfig.interval,
        capacity: ARCJET_CONFIG.rateLimitConfig.capacity,
      }),
    ],
  });
};

/**
 * Handle Arcjet decision and send appropriate response
 */
export const handleArcjetDecision = (decision: ArcjetDecision, res: Response): void => {
  console.log("Arcjet decision", decision);

  if (decision.isDenied()) {
    if (decision.reason.isRateLimit()) {
      sendErrorResponse(res, 429, "Too Many Requests");
    } else if (decision.reason.isBot()) {
      sendErrorResponse(res, 403, "No bots allowed");
    } else {
      sendErrorResponse(res, 403, "Forbidden");
    }
    return;
  }

  if (decision.ip.isHosting()) {
    // Requests from hosting IPs are likely from bots, so they can usually be
    // blocked. However, consider your use case - if this is an API endpoint
    // then hosting IPs might be legitimate.
    // https://docs.arcjet.com/blueprints/vpn-proxy-detection
    sendErrorResponse(res, 403, "Forbidden");
    return;
  }

  if (decision.results.some(isSpoofedBot)) {
    // Paid Arcjet accounts include additional verification checks using IP data.
    // Verification isn't always possible, so we recommend checking the decision
    // separately.
    // https://docs.arcjet.com/bot-protection/reference#bot-verification
    sendErrorResponse(res, 403, "Forbidden");
    return;
  }

  // Request is allowed - this will be handled by the calling function
  console.log("Request passed Arcjet protection");
};

/**
 * Check if Arcjet protection should be applied
 */
export const shouldApplyArcjetProtection = (key: string): boolean => {
  if (!key) return false;
  const environment = process.env.NODE_ENV || "development";
  return environment === "production";
};
