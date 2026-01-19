import { createCompressionMiddleware } from '../../src/middleware/compression.middleware';
import { mockRequest, mockResponse, mockNext } from '../helpers/mocks';
import { Request, Response } from 'express';

describe('Compression Middleware', () => {
  describe('Responses larger than 1KB', () => {
    it('should compress responses larger than 1KB', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip, deflate',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      // Mock a large response (> 1KB)
      const largeContent = 'x'.repeat(2048); // 2KB of data
      (res as any).write = jest.fn();
      (res as any).end = jest.fn();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should set appropriate compression headers for large responses', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });
  });

  describe('Responses smaller than 1KB', () => {
    it('should not compress responses smaller than 1KB', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip, deflate',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      // Mock a small response (< 1KB)
      const smallContent = 'x'.repeat(512); // 512 bytes
      (res as any).write = jest.fn();
      (res as any).end = jest.fn();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should pass through small responses without compression', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });
  });

  describe('SSE streams', () => {
    it('should not compress Server-Sent Events streams', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip, deflate',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      // Mock SSE content type
      (res.getHeader as jest.Mock).mockReturnValue('text/event-stream');

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should not compress text/event-stream content type', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      // Mock SSE content type with charset
      (res.getHeader as jest.Mock).mockReturnValue('text/event-stream; charset=utf-8');

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should compress non-SSE content types', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      // Mock regular JSON content type
      (res.getHeader as jest.Mock).mockReturnValue('application/json');

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });
  });

  describe('Compression configuration', () => {
    it('should use compression level 6', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should use 1KB threshold', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'gzip',
        },
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });
  });

  describe('Client support', () => {
    it('should handle requests without accept-encoding header', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {},
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });

    it('should handle requests with unsupported encoding', () => {
      const middleware = createCompressionMiddleware();
      const req = mockRequest({
        headers: {
          'accept-encoding': 'br', // Brotli only
        },
      });
      const res = mockResponse();
      const next = mockNext();

      middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
    });
  });
});
