import { Request, Response, NextFunction } from "express";

/**
 * Standard error response format returned by the API Gateway.
 * 
 * @property error - Error type or category (e.g., "TimeoutError", "ProxyError")
 * @property message - Human-readable error message (optional)
 * @property statusCode - HTTP status code
 * @property timestamp - ISO 8601 timestamp when the error occurred
 */
export interface ErrorResponse {
  error: string;
  message?: string;
  statusCode: number;
  timestamp: string;
}

/**
 * Centralized error handling middleware for the API Gateway.
 * 
 * This middleware:
 * - Logs all errors with stack traces and request context
 * - Maps error types to appropriate HTTP status codes
 * - Returns consistent JSON error responses
 * - Hides internal error details in production
 * 
 * Must be registered last in the middleware chain to catch all errors.
 * 
 * @param err - Error object (may include statusCode or status properties)
 * @param req - Express request object
 * @param res - Express response object
 * @param next - Express next function (unused but required by Express)
 * @returns void
 * 
 * @example
 * ```typescript
 * // Register as the last middleware
 * app.use(errorHandler);
 * ```
 */
export function errorHandler(
  err: Error & { statusCode?: number; status?: number },
  req: Request,
  res: Response,
  next: NextFunction
): void {
  // Log error with stack trace
  console.error("Error occurred:", {
    error: err.name,
    message: err.message,
    stack: err.stack,
    method: req.method,
    path: req.path,
    ip: req.ip,
    timestamp: new Date().toISOString(),
  });

  // Determine status code
  let statusCode = 500;

  // Check for explicit status code on error object
  if (err.statusCode) {
    statusCode = err.statusCode;
  } else if (err.status) {
    statusCode = err.status;
  } else {
    // Map error types to HTTP status codes
    switch (err.name) {
      case "TimeoutError":
        statusCode = 408;
        break;
      case "ProxyError":
        statusCode = 502;
        break;
      case "ValidationError":
        statusCode = 400;
        break;
      case "UnauthorizedError":
        statusCode = 401;
        break;
      case "ForbiddenError":
        statusCode = 403;
        break;
      case "NotFoundError":
        statusCode = 404;
        break;
      default:
        statusCode = 500;
    }
  }

  // Build error response
  const errorResponse: ErrorResponse = {
    error: err.name || "Internal Server Error",
    message: err.message || "An unexpected error occurred",
    statusCode,
    timestamp: new Date().toISOString(),
  };

  // In production, hide internal error details
  if (process.env.NODE_ENV === "production" && statusCode === 500) {
    errorResponse.message = "An unexpected error occurred";
  }

  // Send error response
  res.status(statusCode).json(errorResponse);
}
