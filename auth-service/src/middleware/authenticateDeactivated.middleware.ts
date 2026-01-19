import { Request, Response, NextFunction } from "express";
import { verifyAccessToken } from "../utils/tokens";
import { UnauthorizedError } from "../utils/errors";
import { getAccessTokenFromRequest } from "../utils/cookies";
import { prisma } from "../libs/prisma";

/**
 * Special authentication middleware for deactivated accounts
 * Allows deactivated users to authenticate with temp token for reactivation
 * Similar to authenticate middleware but skips the isActive check
 */
export const authenticateDeactivated = async (req: Request, res: Response, next: NextFunction) => {
  try {
    // Extract token from request (cookies, Authorization header, or query param)
    let token = getAccessTokenFromRequest(req);
    
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

    // Check user account status (but allow deactivated accounts)
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

    // NOTE: We intentionally skip the isActive check here
    // This allows deactivated users to authenticate for reactivation

    // Attach user info to request object
    req.user = {
      id: user.id,
      role: user.role,
      jti: payload.jti,
    };

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
