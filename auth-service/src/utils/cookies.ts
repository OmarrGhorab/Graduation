import { Response } from "express";

const isProd = process.env.NODE_ENV === "production";

type CookieOptions = {
  refreshMaxAgeSec?: number;
};

export function setAuthCookies(res: Response, accessToken: string, refreshToken: string, opts: CookieOptions = {}): void {
  // Access token short-lived; recommend header on APIs; also set cookie if needed
  res.cookie("access_token", accessToken, {
    httpOnly: true,
    secure: isProd,
    sameSite: "lax",
    maxAge: 1000 * 60 * 15, // 15 minutes
    path: "/",
  });

  res.cookie("refresh_token", refreshToken, {
    httpOnly: true,
    secure: isProd,
    sameSite: "lax",
    maxAge: 1000 * (opts.refreshMaxAgeSec || parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10)),
    path: "/",
  });
}

export function clearAuthCookies(res: Response): void {
  res.clearCookie("access_token", { path: "/" });
  res.clearCookie("refresh_token", { path: "/" });
}


