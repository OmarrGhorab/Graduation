import timeout from "connect-timeout";
import { Request, Response, NextFunction, RequestHandler } from "express";

/**
 * Creates timeout middleware with halt-on-timeout handler.
 * 
 * This function returns two middleware functions:
 * 1. Timeout middleware - Sets a timeout for the request
 * 2. Halt-on-timeout middleware - Checks if request timed out and returns 408 error
 * 
 * The halt-on-timeout middleware should be placed after body parsers to prevent
 * processing timed-out requests.
 * 
 * @param timeoutMs - Timeout in milliseconds (default: 30000ms = 30 seconds)
 * @returns Array containing [timeoutMiddleware, haltOnTimeoutMiddleware]
 * 
 * @example
 * ```typescript
 * const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(30000);
 * app.use(timeoutMiddleware);
 * app.use(express.json());
 * app.use(haltOnTimeout); // Check timeout after body parsing
 * ```
 */
export function createTimeoutMiddleware(
  timeoutMs: number = 30000
): RequestHandler[] {
  // Timeout middleware that sets the timeout
  const timeoutMiddleware = timeout(timeoutMs);

  // Halt-on-timeout middleware that checks if request timed out
  const haltOnTimeout: RequestHandler = (
    req: Request,
    res: Response,
    next: NextFunction
  ) => {
    if (req.timedout) {
      return res.status(408).json({
        error: "Request Timeout",
        message: `Request exceeded ${timeoutMs}ms timeout`,
        statusCode: 408,
        timestamp: new Date().toISOString(),
      });
    }
    next();
  };

  return [timeoutMiddleware, haltOnTimeout];
}
