import { Request } from "express";

/**
 * Extract access token from request Authorization header
 * Mobile-first approach: Uses Authorization header with Bearer token
 */
export function getAccessTokenFromRequest(req: Request): string | undefined {
  const authHeader = req.headers.authorization;
  if (authHeader && authHeader.startsWith("Bearer ")) {
    return authHeader.substring(7);
  }
  return undefined;
}

/**
 * Extract refresh token from request x-refresh-token header
 * Mobile-first approach: Uses x-refresh-token header
 */
export function getRefreshTokenFromRequest(req: Request): string | undefined {
  const headerToken = req.headers["x-refresh-token"];
  if (headerToken) {
    if (typeof headerToken === "string") {
      return headerToken;
    } else if (Array.isArray(headerToken) && headerToken.length > 0) {
      return headerToken[0];
    }
  }
  return undefined;
}


