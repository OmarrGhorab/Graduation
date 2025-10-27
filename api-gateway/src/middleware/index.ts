import express, { Request, Response, NextFunction } from "express";
import morgan from "morgan";
import cors from "cors";
import cookieParser from "cookie-parser";
import { CORS_CONFIG, REQUEST_LIMIT } from "../config/index.js";
import { ErrorHandler } from "../types/index.js";

/**
 * Configure CORS middleware
 */
export const corsMiddleware = cors({
  origin: CORS_CONFIG.origins,
  allowedHeaders: CORS_CONFIG.allowedHeaders,
  credentials: CORS_CONFIG.credentials,
});

/**
 * Setup basic Express middleware
 */
export const setupBasicMiddleware = (app: express.Application): void => {
  app.use(morgan("dev"));
  app.use(express.json());
  app.use(express.urlencoded({ limit: REQUEST_LIMIT, extended: true }));
  app.use(cookieParser());
  
  // Trust proxy (useful when behind reverse proxy like Nginx)
  app.set("trust proxy", 1);
};

/**
 * Global error handler middleware
 */
export const globalErrorHandler: ErrorHandler = (err: Error, req: Request, res: Response, next: NextFunction): void => {
  console.error("Unhandled error:", err);
  res.status(500).json({ error: "Internal Server Error" });
};

/**
 * 404 handler for unknown routes
 */
export const notFoundHandler = (req: Request, res: Response): void => {
  res.status(404).json({ error: "Not Found" });
};
