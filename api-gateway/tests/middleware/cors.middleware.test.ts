import { createCorsMiddleware } from '../../src/middleware/cors.middleware';
import { mockRequest, mockResponse, mockNext } from '../helpers/mocks';
import { CorsConfig } from '../../src/config/index';

describe('CORS Middleware', () => {
  const defaultConfig: CorsConfig = {
    allowedOrigins: ['http://localhost:3000', 'https://example.com'],
    credentials: true,
    allowedHeaders: ['Content-Type', 'Authorization'],
  };

  describe('Requests with no origin', () => {
    it('should allow requests with no origin header', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ headers: {} });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', '*');
      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Credentials', 'true');
      expect(res.setHeader).toHaveBeenCalledWith(
        'Access-Control-Allow-Headers',
        'Content-Type, Authorization'
      );
      expect(res.setHeader).toHaveBeenCalledWith(
        'Access-Control-Allow-Methods',
        'GET, POST, PUT, DELETE, PATCH, OPTIONS'
      );
      expect(next).toHaveBeenCalled();
    });

    it('should handle OPTIONS preflight requests with no origin', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ method: 'OPTIONS', headers: {} });
      const res = mockResponse();
      const next = mockNext();

      // Add sendStatus method to mock response
      (res as any).sendStatus = jest.fn().mockReturnThis();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', '*');
      expect((res as any).sendStatus).toHaveBeenCalledWith(204);
      expect(next).not.toHaveBeenCalled();
    });
  });

  describe('Requests with whitelisted origin', () => {
    it('should allow requests from whitelisted origin', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ headers: { origin: 'http://localhost:3000' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'http://localhost:3000');
      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Credentials', 'true');
      expect(res.setHeader).toHaveBeenCalledWith(
        'Access-Control-Allow-Headers',
        'Content-Type, Authorization'
      );
      expect(res.setHeader).toHaveBeenCalledWith(
        'Access-Control-Allow-Methods',
        'GET, POST, PUT, DELETE, PATCH, OPTIONS'
      );
      expect(next).toHaveBeenCalled();
    });

    it('should allow requests from another whitelisted origin', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ headers: { origin: 'https://example.com' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'https://example.com');
      expect(next).toHaveBeenCalled();
    });

    it('should handle OPTIONS preflight requests from whitelisted origin', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({
        method: 'OPTIONS',
        headers: { origin: 'http://localhost:3000' },
      });
      const res = mockResponse();
      const next = mockNext();

      // Add sendStatus method to mock response
      (res as any).sendStatus = jest.fn().mockReturnThis();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'http://localhost:3000');
      expect((res as any).sendStatus).toHaveBeenCalledWith(204);
      expect(next).not.toHaveBeenCalled();
    });
  });

  describe('Requests with non-whitelisted origin', () => {
    it('should block requests from non-whitelisted origin', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ headers: { origin: 'https://malicious.com' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        error: 'Forbidden',
        message: 'Origin not allowed',
        statusCode: 403,
        timestamp: expect.any(String),
      });
      expect(next).not.toHaveBeenCalled();
    });

    it('should not set CORS headers for blocked origins', () => {
      const middleware = createCorsMiddleware(defaultConfig);
      const req = mockRequest({ headers: { origin: 'https://evil.com' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).not.toHaveBeenCalled();
      expect(res.status).toHaveBeenCalledWith(403);
      expect(next).not.toHaveBeenCalled();
    });
  });

  describe('Wildcard origin', () => {
    const wildcardConfig: CorsConfig = {
      allowedOrigins: ['*'],
      credentials: true,
      allowedHeaders: ['Content-Type', 'Authorization'],
    };

    it('should allow all origins when wildcard is configured', () => {
      const middleware = createCorsMiddleware(wildcardConfig);
      const req = mockRequest({ headers: { origin: 'https://any-domain.com' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'https://any-domain.com');
      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Credentials', 'true');
      expect(next).toHaveBeenCalled();
    });

    it('should allow different origins with wildcard', () => {
      const middleware = createCorsMiddleware(wildcardConfig);
      const req = mockRequest({ headers: { origin: 'http://localhost:8080' } });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'http://localhost:8080');
      expect(next).toHaveBeenCalled();
    });

    it('should handle OPTIONS preflight with wildcard', () => {
      const middleware = createCorsMiddleware(wildcardConfig);
      const req = mockRequest({
        method: 'OPTIONS',
        headers: { origin: 'https://random-site.com' },
      });
      const res = mockResponse();
      const next = mockNext();

      // Add sendStatus method to mock response
      (res as any).sendStatus = jest.fn().mockReturnThis();

      middleware(req as any, res as any, next);

      expect(res.setHeader).toHaveBeenCalledWith('Access-Control-Allow-Origin', 'https://random-site.com');
      expect((res as any).sendStatus).toHaveBeenCalledWith(204);
      expect(next).not.toHaveBeenCalled();
    });
  });
});
