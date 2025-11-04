import proxy from 'express-http-proxy';
import { Request, Response, NextFunction } from "express";
import { PROXY_CONFIG } from "../config/index.js";
import { sendErrorResponse } from "../utils/responses.js";

/**
 * Create proxy middleware for auth service
 */
export const createAuthServiceProxy = () => {
  return proxy(PROXY_CONFIG.authServiceUrl, {
    timeout: PROXY_CONFIG.timeout,
    proxyReqPathResolver: (req: Request): string => {
      // Include query string in the proxied path - use originalUrl to preserve query params
      const path = req.originalUrl || req.url || req.path;
      console.log(`Proxying request to auth service: ${req.method} ${path}`);
      return path;
    },
    proxyErrorHandler: (err: Error, res: Response, next: NextFunction): void => {
      console.error("Proxy error:", err.message);
      sendErrorResponse(res, 502, "Auth service unavailable");
    },
    userResDecorator: (proxyRes: any, proxyResData: Buffer, userReq: Request, userRes: Response): Buffer => {
      console.log(`Auth service response: ${proxyRes.statusCode}`);
      return proxyResData;
    }
  });
};
