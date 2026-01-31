import { NextFunction, Request, Response } from "express";
import { AppError } from "../utils/errors";

export function errorHandler(err: unknown, req: Request, res: Response, _next: NextFunction) {
  const isKnown = err instanceof AppError;
  const status = isKnown ? err.statusCode : 500;
  const message = isKnown ? err.message : "Internal server error";
  const payload: Record<string, unknown> = {
    message,
    statusCode: status,
    timestamp: new Date().toISOString()
  };
  if (isKnown && err.details) payload.details = err.details;
  if (process.env.NODE_ENV !== "production" && !isKnown) {
    payload.error = String(err);
  }
  res.status(status).json(payload);
}


