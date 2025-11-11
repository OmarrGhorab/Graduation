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
 * Extract access token from request (cookies or Authorization header)
 */
export function getAccessTokenFromRequest(req: Request): string | undefined {
  const cookies = parseCookiesFromRequest(req);
  let token = cookies.access_token;

  // Fallback to Authorization header if not in cookies
  if (!token) {
    const authHeader = req.headers.authorization;
    if (authHeader && authHeader.startsWith("Bearer ")) {
      token = authHeader.substring(7);
    }
  }

  return token;
}

/**
 * Extract refresh token from request (cookies or Authorization header)
 */
export function getRefreshTokenFromRequest(req: Request): string | undefined {
  const cookies = parseCookiesFromRequest(req);
  let token = cookies.refresh_token;

  // Fallback to Authorization header if not in cookies
  if (!token) {
    const authHeader = req.headers.authorization;
    if (authHeader && authHeader.startsWith("Bearer ")) {
      token = authHeader.substring(7);
    }
    // Also check x-refresh-token header as fallback
    if (!token) {
      const headerToken = req.headers["x-refresh-token"];
      if (typeof headerToken === "string") {
        token = headerToken;
      } else if (Array.isArray(headerToken) && headerToken.length > 0) {
        token = headerToken[0];
      }
    }
  }

  return token;
}

type CookieOptions = {
  refreshMaxAgeSec?: number;
};

export function setAuthCookies(res: Response, accessToken: string, refreshToken: string, opts: CookieOptions = {}): void {
  // Access token short-lived; recommend header on APIs; also set cookie if needed
  res.cookie("access_token", accessToken, {
    httpOnly: true,
    secure: isProd,
    sameSite: "none",
    maxAge: 1000 * 60 * 15, // 15 minutes
    path: "/",
  });

  res.cookie("refresh_token", refreshToken, {
    httpOnly: true,
    secure: isProd,
    sameSite: "none",
    maxAge: 1000 * (opts.refreshMaxAgeSec || parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10)),
    path: "/",
  });
}

export function clearAuthCookies(res: Response): void {
  res.clearCookie("access_token", { path: "/" });
  res.clearCookie("refresh_token", { path: "/" });
}


