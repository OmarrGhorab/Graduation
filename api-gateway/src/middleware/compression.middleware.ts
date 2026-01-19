import compression from "compression";
import { Request, Response, RequestHandler } from "express";

/**
 * Creates compression middleware for response compression.
 * 
 * Compresses HTTP responses using gzip to reduce bandwidth usage.
 * Configuration:
 * - Compression level: 6 (balanced between speed and compression ratio)
 * - Threshold: 1KB (only compress responses larger than 1KB)
 * - Excludes Server-Sent Events (SSE) streams from compression
 * 
 * @returns Express middleware function that handles response compression
 * 
 * @example
 * ```typescript
 * const compressionMiddleware = createCompressionMiddleware();
 * app.use(compressionMiddleware);
 * ```
 */
export function createCompressionMiddleware(): RequestHandler {
  return compression({
    // Compression level: 6 (balanced between speed and compression ratio)
    level: 6,

    // Only compress responses larger than 1KB
    threshold: 1024,

    // Filter function to exclude certain content types
    filter: (req: Request, res: Response) => {
      // Don't compress Server-Sent Events (SSE) streams
      const contentType = res.getHeader("Content-Type");
      if (contentType && contentType.toString().includes("text/event-stream")) {
        return false;
      }

      // Use compression's default filter for everything else
      return compression.filter(req, res);
    },
  });
}
