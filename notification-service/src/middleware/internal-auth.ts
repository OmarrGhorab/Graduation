import { Request, Response, NextFunction } from 'express';

/**
 * Internal service authentication middleware
 * Validates a shared secret header for inter-service communication
 */
export const authenticateInternalService = async (
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> => {
  try {
    const internalSecret = req.headers['x-internal-service-secret'] as string;
    const expectedSecret = process.env.INTERNAL_SERVICE_SECRET;

    if (!expectedSecret) {
      console.error('INTERNAL_SERVICE_SECRET not configured');
      res.status(500).json({ error: 'Service configuration error' });
      return;
    }

    if (!internalSecret || internalSecret !== expectedSecret) {
      res.status(401).json({ error: 'Unauthorized internal service' });
      return;
    }

    next();
  } catch (error) {
    console.error('Internal authentication error:', error);
    res.status(500).json({ error: 'Internal authentication failed' });
  }
};
