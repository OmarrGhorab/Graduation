import { Request, Response, NextFunction } from "express";
import { verifyAccessToken } from "../utils/tokens";
import { UnauthorizedError } from "../utils/errors";

/**
 * Helper function to parse cookies from Cookie header
 * Used as fallback if cookie-parser middleware is not installed
 */
function parseCookies(cookieHeader?: string): Record<string, string> {
  const cookies: Record<string, string> = {};
  if (!cookieHeader) return cookies;

  cookieHeader.split(";").forEach((cookie) => {
    const [name, ...rest] = cookie.trim().split("=");
    if (name && rest.length > 0) {
      cookies[name] = rest.join("=");
    }
  });

  return cookies;
}

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
    // Extract token from cookies (preferred method)
    // Try req.cookies first (if cookie-parser is installed)
    let token = (req.cookies as Record<string, string> | undefined)?.access_token;

    // Fallback: parse cookies manually from Cookie header if cookie-parser is not installed
    if (!token && req.headers.cookie) {
      const cookies = parseCookies(req.headers.cookie);
      token = cookies.access_token;
    }

    // Fallback to Authorization header if not in cookies
    if (!token) {
      const authHeader = req.headers.authorization;
      if (authHeader && authHeader.startsWith("Bearer ")) {
        token = authHeader.substring(7);
      }
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

    // Attach user info to request object
    req.user = {
      id: payload.sub,
      role: payload.role,
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

