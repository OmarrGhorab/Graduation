import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';

interface AuthenticatedRequest extends Request {
  user?: {
    id: string;
    email: string;
    username: string;
    role: string;
  };
}

export const authenticate = async (
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> => {
  try {
    const authHeader = req.headers.authorization;
    
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      res.status(401).json({ error: 'No token provided' });
      return;
    }

    const token = authHeader.substring(7);
    const jwtSecret = process.env.JWT_ACCESS_SECRET;
    
    if (!jwtSecret) {
      throw new Error('JWT_ACCESS_SECRET not configured');
    }

    const decoded = jwt.verify(token, jwtSecret) as any;
    
    // Check token type - only access tokens allowed
    if (decoded.type !== 'access') {
      res.status(401).json({ error: 'Invalid token type' });
      return;
    }
    
    // Cast to AuthenticatedRequest to add user property
    (req as AuthenticatedRequest).user = {
      id: decoded.sub,
      email: decoded.email,
      username: decoded.username,
      role: decoded.role
    };

    next();
  } catch (error) {
    console.error('Authentication error:', error);
    res.status(401).json({ error: 'Invalid token' });
  }
};

export type { AuthenticatedRequest };
