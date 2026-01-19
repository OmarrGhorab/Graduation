import { Request, Response, NextFunction } from "express";
import { verifyAccessToken } from "../utils/tokens";
import { UnauthorizedError } from "../utils/errors";
import { getAccessTokenFromRequest } from "../utils/cookies";
import { prisma } from "../libs/prisma";
import { updateSessionActivity } from "../utils/sessions";
import { getDeviceLocationFromRequest, hasValidLocation } from "./deviceInfo.middleware";
import { updateSessionLocation } from "../services/location.service";

/**
 * Authentication middleware to verify access token and attach user info to request
 * Extracts token from cookies (access_token) or Authorization header (Bearer token)
 * 
 * OPTIMIZED: Combined user and session queries into a single DB call for better performance
 */
export const authenticate = async (req: Request, res: Response, next: NextFunction) => {
  try {
    // Extract token from request (cookies, Authorization header, or query param for SSE)
    let token = getAccessTokenFromRequest(req);
    
    // For SSE connections, also check query parameter
    if (!token && req.query.token) {
      token = req.query.token as string;
    }

    if (!token) {
      throw new UnauthorizedError("Access token is required");
    }

    // Verify the access token
    const payload = await verifyAccessToken(token);

    // Ensure it's an access token (not a refresh token)
    if (payload.type !== "access") {
      throw new UnauthorizedError("Invalid token type");
    }

    // OPTIMIZED: Single query to get session with user data
    // This reduces 2 DB queries to 1, improving latency on every authenticated request
    const session = await prisma.session.findFirst({
      where: {
        sessionToken: payload.jti,
        userId: payload.sub,
        isRevoked: false,
        expiresAt: {
          gt: new Date(),
        },
      },
      select: {
        id: true,
        user: {
          select: {
            id: true,
            isActive: true,
            deletedAt: true,
            role: true,
          },
        },
      },
    });

    if (!session) {
      throw new UnauthorizedError("Session not found, has been revoked, or has expired");
    }

    const user = session.user;

    if (!user) {
      throw new UnauthorizedError("User not found");
    }

    if (user.deletedAt) {
      throw new UnauthorizedError("Account has been deleted");
    }

    if (!user.isActive) {
      throw new UnauthorizedError("Account is deactivated");
    }

    // Attach user info to request object
    req.user = {
      id: user.id,
      role: user.role,
      jti: payload.jti,
    };

    // Update session activity (non-blocking)
    updateSessionActivity(payload.jti).catch((err) => {
      console.error("Failed to update session activity:", err);
    });

    // Update session location if valid location headers present (non-blocking)
    const deviceLocation = getDeviceLocationFromRequest(req);
    if (hasValidLocation(deviceLocation)) {
      updateSessionLocation(payload.jti, deviceLocation).catch((err) => {
        console.error("Failed to update session location:", err);
      });
    }

    next();
  } catch (err) {
    // Handle JWT verification errors
    if (err instanceof Error) {
      if (err.name === "JsonWebTokenError" || err.name === "TokenExpiredError") {
        return next(new UnauthorizedError("Invalid or expired token"));
      }
    }
    next(err);
  }
};

/**
 * Optional middleware to check if user has a specific role
 * Must be used after authenticate middleware
 */
export const requireRole = (...allowedRoles: string[]) => {
  return (req: Request, res: Response, next: NextFunction) => {
    if (!req.user) {
      return next(new UnauthorizedError("Authentication required"));
    }

    if (!req.user.role || !allowedRoles.includes(req.user.role)) {
      return next(new UnauthorizedError("Insufficient permissions"));
    }

    next();
  };
};

