import { Request, Response, NextFunction } from 'express';
import { UnauthorizedError, BadRequestError, NotFoundError, ForbiddenError } from '../utils/errors';

export const errorHandler = (
  err: Error,
  req: Request,
  res: Response,
  next: NextFunction
): void => {
  console.error('Error:', err);

  if (err instanceof UnauthorizedError) {
    res.status(401).json({
      error: 'Unauthorized',
      message: err.message
    });
    return;
  }

  if (err instanceof ForbiddenError) {
    res.status(403).json({
      error: 'Forbidden',
      message: err.message
    });
    return;
  }

  if (err instanceof BadRequestError) {
    res.status(400).json({
      error: 'Bad Request',
      message: err.message
    });
    return;
  }

  if (err instanceof NotFoundError) {
    res.status(404).json({
      error: 'Not Found',
      message: err.message
    });
    return;
  }

  // Default error response
  res.status(500).json({
    error: 'Internal Server Error',
    message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong'
  });
};
