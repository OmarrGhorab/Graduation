import { Request, Response, NextFunction } from "express";
import { isSpoofedBot } from "@arcjet/inspect";
import { ARCJET_CONFIG } from "../config/index.js";
import { initializeArcjet, handleArcjetDecision, shouldApplyArcjetProtection } from "../services/arcjet.js";
import { createAuthServiceProxy } from "../services/proxy.js";
import { createNotificationServiceProxy } from "../services/notification-proxy.js";
import { sendErrorResponse } from "../utils/responses.js";

/**
 * Create root endpoint handler with Arcjet protection and proxy to auth service
 */
export const createRootHandler = (arcjetKey: string) => {
  const aj = initializeArcjet(arcjetKey);
  const authServiceProxy = createAuthServiceProxy();
  const notificationServiceProxy = createNotificationServiceProxy();

  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    try {
      // Route notification endpoints to notification service
      if (req.path.startsWith('/api/v1/notifications')) {
        console.log("Routing to notification service:", req.method, req.path);
        notificationServiceProxy(req, res, next);
        return;
      }

      // Only run Arcjet protection if key is configured
      if (!shouldApplyArcjetProtection(arcjetKey)) {
        console.log("Skipping Arcjet protection (no key configured)");
        // Proxy directly to auth service without protection
        authServiceProxy(req, res, next);
        return;
      }

      const decision = await aj.protect(req, { requested: ARCJET_CONFIG.tokensRequested });
      
      if (decision.isDenied()) {
        handleArcjetDecision(decision, res);
        return;
      }

      if (decision.ip.isHosting()) {
        sendErrorResponse(res, 403, "Forbidden");
        return;
      }

      if (decision.results.some(isSpoofedBot)) {
        sendErrorResponse(res, 403, "Forbidden");
        return;
      }

      // Request is allowed, proxy to auth service
      console.log("Request passed Arcjet protection, proxying to auth service");
      authServiceProxy(req, res, next);
      
    } catch (error) {
      console.error("Error in Arcjet protection:", error);
      
      // Log detailed error information for debugging
      if (error instanceof Error) {
        console.error("Error name:", error.name);
        console.error("Error message:", error.message);
        console.error("Error stack:", error.stack);
      }
      
      sendErrorResponse(res, 500, "Internal Server Error");
    }
  };
};
