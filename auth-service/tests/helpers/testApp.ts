import express, { Express, Request, Response, NextFunction } from 'express';
import cors from 'cors';
import authRouter from '../../src/routes/auth.route';
import onboardingRouter from '../../src/routes/onboarding.route';
import parentLinkRouter from '../../src/routes/parent-link.route';
import profileRouter from '../../src/routes/profile.route';
import locationRouter from '../../src/routes/location.route';
import internalRouter from '../../src/routes/internal.route';
import { errorHandler } from '../../src/middleware/errorHandler';
import { extractDeviceInfo } from '../../src/middleware/deviceInfo.middleware';
import { verifyAccessToken } from '../../src/utils/tokens';
import { getAccessTokenFromRequest } from '../../src/utils/cookies';
import { UnauthorizedError } from '../../src/utils/errors';

/**
 * Options for creating a test Express application
 */
export interface TestAppOptions {
  /**
   * Skip authentication middleware for testing unauthenticated scenarios
   * When true, authentication middleware will be bypassed
   */
  skipAuth?: boolean;
  
  /**
   * Skip rate limiting middleware for testing without rate limit constraints
   * When true, rate limiting will be disabled
   */
  skipRateLimit?: boolean;
  
  /**
   * Custom mocks for external dependencies
   * Allows injection of mocked Prisma, Redis, Resend, etc.
   */
  customMocks?: {
    prisma?: any;
    redis?: any;
    resend?: any;
    cloudinary?: any;
  };
}

/**
 * Simplified authentication middleware for testing
 * Bypasses session lookup and directly attaches user to request
 * This makes tests faster and more reliable by avoiding database mocking complexity
 */
export const mockAuthMiddleware = async (req: Request, res: Response, next: NextFunction) => {
  try {
    // Extract token from request
    let token = getAccessTokenFromRequest(req);
    
    // For SSE connections, also check query parameter
    if (!token && req.query.token) {
      token = req.query.token as string;
    }

    if (!token) {
      throw new UnauthorizedError("Access token is required");
    }

    // Verify the access token
    const payload = await verifyAccessToken(token);

    // Ensure it's an access token (not a refresh token)
    if (payload.type !== "access") {
      throw new UnauthorizedError("Invalid token type");
    }

    // Attach user info to request WITHOUT database lookup
    // This is the key simplification for testing
    req.user = {
      id: payload.sub,
      role: payload.role || 'STUDENT',
      jti: payload.jti,
    };

    next();
  } catch (err) {
    // Handle JWT verification errors
    if (err instanceof Error) {
      if (err.name === "JsonWebTokenError" || err.name === "TokenExpiredError") {
        return next(new UnauthorizedError("Invalid or expired token"));
      }
    }
    next(err);
  }
};

/**
 * Creates a test Express application instance with all routes and middleware
 * configured for testing purposes.
 * 
 * This factory function creates an Express app similar to the production app
 * but with options to skip authentication and rate limiting for easier testing.
 * 
 * @param options - Configuration options for the test app
 * @returns Express application instance configured for testing
 * 
 * @example
 * ```typescript
 * // Create app with authentication enabled
 * const app = createTestApp();
 * 
 * // Create app without authentication for testing public endpoints
 * const app = createTestApp({ skipAuth: true });
 * 
 * // Create app without rate limiting for load testing
 * const app = createTestApp({ skipRateLimit: true });
 * ```
 */
export function createTestApp(options: TestAppOptions = {}): Express {
  const app = express();
  
  // Trust proxy for correct IP detection behind reverse proxies
  app.set('trust proxy', true);
  
  // Body parsing middleware
  app.use(express.json({ limit: '10mb' }));
  app.use(express.urlencoded({ extended: true, limit: '10mb' }));
  
  // CORS configuration
  app.use(cors({
    origin: true, // Allow all origins (mobile apps don't send origin header)
    credentials: false, // Not needed for mobile - tokens sent via Authorization header
    allowedHeaders: [
      'Content-Type',
      'Authorization',
      'x-refresh-token',
      'x-forwarded-for',
      'x-real-ip',
      // Custom device info headers
      'x-device-name',
      'x-device-model',
      'x-device-os-version',
      'x-app-version',
      'x-device-location',
      'x-device-timezone',
      'x-device-platform',
      // GPS location headers
      'x-device-latitude',
      'x-device-longitude',
      'x-device-location-accuracy',
    ],
  }));
  
  // Extract device info from headers on all requests
  app.use(extractDeviceInfo);
  
  // Health check endpoint
  app.get('/health', (req, res) => {
    res.status(200).json({
      status: 'ok',
      service: 'auth-service-test',
      timestamp: new Date().toISOString(),
    });
  });
  
  // Root endpoint
  app.get('/', (req, res) => {
    res.send('auth service test app is running');
  });
  
  // Apply routes
  app.use('/api/v1/auth', authRouter);
  app.use('/api/v1/onboarding', onboardingRouter);
  app.use('/api/v1/parent-link', parentLinkRouter);
  app.use('/api/v1/profile', profileRouter);
  app.use('/api/v1/location', locationRouter);
  app.use('/api/v1/internal', internalRouter);
  
  // Error handler must be last
  app.use(errorHandler);
  
  return app;
}
