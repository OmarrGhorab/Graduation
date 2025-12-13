import { Request, Response } from "express";
import dotenv from "dotenv";

dotenv.config();

const isProd = process.env.NODE_ENV === "production";

/**
 * Helper function to parse cookies from Request
 * Tries req.cookies first (if cookie-parser is installed), then falls back to manual parsing
 */
export function parseCookiesFromRequest(req: Request): Record<string, string> {
  // Try req.cookies first (if cookie-parser is installed)
  if (req.cookies && typeof req.cookies === "object") {
    return req.cookies as Record<string, string>;
  }

  // Fallback: parse cookies manually from Cookie header
  const cookies: Record<string, string> = {};
  if (!req.headers.cookie) return cookies;

  req.headers.cookie.split(";").forEach((cookie) => {
    const [name, ...rest] = cookie.trim().split("=");
    if (name && rest.length > 0) {
      cookies[name] = rest.join("=");
    }
  });

  return cookies;
}

/**
 * Extract access token from request (Authorization header preferred, cookies as fallback for backward compatibility)
 * Mobile-first approach: Use Authorization header with Bearer token
 */
export function getAccessTokenFromRequest(req: Request): string | undefined {
  // Primary: Check Authorization header (mobile-first approach)
  const authHeader = req.headers.authorization;
  if (authHeader && authHeader.startsWith("Bearer ")) {
    return authHeader.substring(7);
  }

  // Fallback: Check cookies (for backward compatibility with web clients)
  const cookies = parseCookiesFromRequest(req);
  return cookies.access_token;
}

/**
 * Extract refresh token from request (x-refresh-token header preferred, cookies as fallback for backward compatibility)
 * Mobile-first approach: Use x-refresh-token header
 */
export function getRefreshTokenFromRequest(req: Request): string | undefined {
  // Primary: Check x-refresh-token header (mobile-first approach)
  const headerToken = req.headers["x-refresh-token"];
  if (headerToken) {
    if (typeof headerToken === "string") {
      return headerToken;
    } else if (Array.isArray(headerToken) && headerToken.length > 0) {
      return headerToken[0];
    }
  }

  // Fallback: Check cookies (for backward compatibility with web clients)
  const cookies = parseCookiesFromRequest(req);
  return cookies.refresh_token;
}

/**
 * @deprecated Cookie-based authentication is deprecated for mobile compatibility.
 * Tokens are now returned in response body. Use Authorization header for authentication.
 * This function is kept for backward compatibility but should not be used in new code.
 */
type CookieOptions = {
  refreshMaxAgeSec?: number;
};

export function setAuthCookies(res: Response, accessToken: string, refreshToken?: string, opts: CookieOptions = {}): void {
  // Deprecated: Mobile-first approach uses Authorization headers instead of cookies
  // This function is kept for backward compatibility only
  res.cookie("access_token", accessToken, {
    httpOnly: true,
    secure: isProd,
    sameSite: "none",
    maxAge: 1000 * 60 * 15, // 15 minutes
    path: "/",
  });

  // Only set refresh token if provided
  if (refreshToken) {
    res.cookie("refresh_token", refreshToken, {
      httpOnly: true,
      secure: isProd,
      sameSite: "none",
      maxAge: 1000 * (opts.refreshMaxAgeSec || parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10)),
      path: "/",
    });
  }
}

/**
 * @deprecated Cookie-based authentication is deprecated for mobile compatibility.
 * This function is kept for backward compatibility but should not be used in new code.
 */
export function clearAuthCookies(res: Response): void {
  res.clearCookie("access_token", { path: "/" });
  res.clearCookie("refresh_token", { path: "/" });
}


