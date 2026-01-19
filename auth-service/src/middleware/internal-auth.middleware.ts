import { Request, Response, NextFunction } from "express";

/**
 * Middleware to authenticate internal service-to-service calls
 * Validates the x-internal-service-secret header
 */
export const authenticateInternalService = (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const internalSecret = req.headers["x-internal-service-secret"] as string;
    const expectedSecret = process.env.INTERNAL_SERVICE_SECRET;

    if (!expectedSecret) {
      console.error("[Internal Auth] INTERNAL_SERVICE_SECRET not configured");
      res.status(500).json({ error: "Service configuration error" });
      return;
    }

    if (!internalSecret || internalSecret !== expectedSecret) {
      res.status(401).json({ error: "Unauthorized: Invalid internal service secret" });
      return;
    }

    next();
  } catch (error) {
    console.error("[Internal Auth] Error:", error);
    res.status(500).json({ error: "Internal server error" });
  }
};
