import { Request, Response, NextFunction } from "express";
import { verifyAccessToken } from "../utils/tokens";
import { UnauthorizedError } from "../utils/errors";
import { getAccessTokenFromRequest } from "../utils/cookies";
import prisma from "../libs/prisma";
import { updateSessionActivity } from "../utils/sessions";

/**
 * Authentication middleware to verify access token and attach user info to request
 * Extracts token from cookies (access_token) or Authorization header (Bearer token)
 * 
 * Note: For cookie parsing to work, you need to install and use cookie-parser middleware:
 * npm install cookie-parser
 * npm install --save-dev @types/cookie-parser
 * Then in main.ts: import cookieParser from "cookie-parser"; app.use(cookieParser());
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

    // Check user account status
    const user = await prisma.user.findUnique({
      where: { id: payload.sub },
      select: {
        id: true,
        isActive: true,
        deletedAt: true,
        role: true,
      },
    });

    if (!user) {
      throw new UnauthorizedError("User not found");
    }

    if (user.deletedAt) {
      throw new UnauthorizedError("Account has been deleted");
    }

    if (!user.isActive) {
      throw new UnauthorizedError("Account is deactivated");
    }

    // Verify session exists in database and is not revoked (ensures immediate logout after session revocation)
    const session = await prisma.session.findFirst({
      where: {
        sessionToken: payload.jti,
        userId: user.id,
        isRevoked: false, // Exclude revoked sessions
        expiresAt: {
          gt: new Date(), // Exclude expired sessions
        },
      },
      select: {
        id: true,
      },
    });

    if (!session) {
      throw new UnauthorizedError("Session not found, has been revoked, or has expired");
    }

    // Attach user info to request object
    req.user = {
      id: user.id,
      role: user.role,
      jti: payload.jti,
    };

    // Update session activity (non-blocking)
    updateSessionActivity(payload.jti).catch((err) => {
      // Log error but don't block request
      console.error("Failed to update session activity:", err);
    });

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

