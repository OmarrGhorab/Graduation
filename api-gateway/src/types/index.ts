import { Request, Response, NextFunction } from "express";
import { ArcjetWellKnownBot, ArcjetBotCategory } from "@arcjet/node";

// Server configuration interface
export interface ServerConfig {
  host: string;
  port: number;
}

// API response interfaces
export interface ApiResponse {
  message?: string;
  error?: string;
}

export interface ErrorResponse extends ApiResponse {
  error: string;
}

// Express handler types
export type RequestHandler = (req: Request, res: Response, next: NextFunction) => void | Promise<void>;
export type ErrorHandler = (err: Error, req: Request, res: Response, next: NextFunction) => void;

// Proxy configuration interface
export interface ProxyConfig {
  authServiceUrl: string;
  timeout: number;
}

// Arcjet configuration interface
export interface ArcjetConfig {
  shieldMode: "LIVE" | "DRY_RUN";
  botDetectionMode: "LIVE" | "DRY_RUN";
  allowedBotCategories: readonly (ArcjetWellKnownBot | ArcjetBotCategory)[];
  rateLimitConfig: {
    refillRate: number;
    interval: number;
    capacity: number;
  };
  tokensRequested: number;
}
