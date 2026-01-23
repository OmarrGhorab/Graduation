import { createArcjetMiddleware, ArcjetConfig } from '../../src/middleware/arcjet.middleware';
import { mockRequest, mockResponse, mockNext } from '../helpers/mocks';

// Mock the @arcjet/node module
jest.mock('@arcjet/node', () => {
  const mockProtect = jest.fn();
  const mockArcjet = jest.fn(() => ({
    protect: mockProtect,
  }));

  return {
    __esModule: true,
    default: mockArcjet,
    detectBot: jest.fn((config) => ({ type: 'detectBot', config })),
    shield: jest.fn((config) => ({ type: 'shield', config })),
    mockProtect, // Expose for test access
  };
});

// Import the mocked module to access mockProtect
import arcjet from '@arcjet/node';
const { mockProtect } = require('@arcjet/node');

describe('Arcjet Middleware', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Disabled protection', () => {
    it('should allow all requests when protection is disabled', async () => {
      const config: ArcjetConfig = {
        enabled: false,
        key: 'test-key',
      };

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();
      expect(arcjet).not.toHaveBeenCalled();
    });

    it('should allow all requests when no key is provided', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: undefined,
      };

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();
      expect(arcjet).not.toHaveBeenCalled();
    });

    it('should allow all requests when both disabled and no key', async () => {
      const config: ArcjetConfig = {
        enabled: false,
        key: undefined,
      };

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();
    });
  });

  describe('Bot detection', () => {
    it('should block malicious bots', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
        mode: 'LIVE',
      };

      // Mock a denied decision for a malicious bot
      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '192.168.1.1',
        reason: { type: 'BOT', botType: 'MALICIOUS' },
        results: [{ type: 'detectBot', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '192.168.1.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(mockProtect).toHaveBeenCalledWith(req);
      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        error: 'Forbidden',
        message: 'Request blocked by security policy',
        statusCode: 403,
        timestamp: expect.any(String),
      });
      expect(next).not.toHaveBeenCalled();
    });

    it('should block automated scrapers', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '10.0.0.1',
        reason: { type: 'BOT', botType: 'SCRAPER' },
        results: [{ type: 'detectBot', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '10.0.0.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(next).not.toHaveBeenCalled();
    });
  });

  describe('VPN/Proxy detection', () => {
    it('should block VPN connections', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '172.16.0.1',
        reason: { type: 'SHIELD', shieldType: 'VPN' },
        results: [{ type: 'shield', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '172.16.0.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(mockProtect).toHaveBeenCalledWith(req);
      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        error: 'Forbidden',
        message: 'Request blocked by security policy',
        statusCode: 403,
        timestamp: expect.any(String),
      });
      expect(next).not.toHaveBeenCalled();
    });

    it('should block proxy connections', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '192.168.100.1',
        reason: { type: 'SHIELD', shieldType: 'PROXY' },
        results: [{ type: 'shield', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '192.168.100.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(next).not.toHaveBeenCalled();
    });

    it('should block hosting provider IPs', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '54.123.45.67',
        reason: { type: 'SHIELD', shieldType: 'HOSTING' },
        results: [{ type: 'shield', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '54.123.45.67' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(next).not.toHaveBeenCalled();
    });

    it('should block relay IPs', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '203.0.113.1',
        reason: { type: 'SHIELD', shieldType: 'RELAY' },
        results: [{ type: 'shield', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '203.0.113.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(next).not.toHaveBeenCalled();
    });
  });

  describe('Allowed bots', () => {
    it('should allow search engine bots (Google)', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '66.249.66.1',
        reason: { type: 'BOT', botType: 'SEARCH_ENGINE', botName: 'Googlebot' },
        results: [{ type: 'detectBot', denied: false, allowed: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '66.249.66.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(mockProtect).toHaveBeenCalledWith(req);
      expect(res.status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });

    it('should allow search engine bots (Bing)', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '40.77.167.1',
        reason: { type: 'BOT', botType: 'SEARCH_ENGINE', botName: 'Bingbot' },
        results: [{ type: 'detectBot', denied: false, allowed: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '40.77.167.1' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });

    it('should allow monitoring services', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '1.2.3.4',
        reason: { type: 'BOT', botType: 'MONITOR', botName: 'UptimeRobot' },
        results: [{ type: 'detectBot', denied: false, allowed: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '1.2.3.4' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });

    it('should allow legitimate user requests', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '192.168.1.100',
        reason: { type: 'NONE' },
        results: [
          { type: 'detectBot', denied: false },
          { type: 'shield', denied: false },
        ],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '192.168.1.100' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(res.status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });
  });

  describe('Error handling', () => {
    it('should fail open when Arcjet throws an error', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      mockProtect.mockRejectedValue(new Error('Arcjet service unavailable'));

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Arcjet error (failing open):',
        expect.any(Error)
      );
      expect(res.status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should fail open on network timeout', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      mockProtect.mockRejectedValue(new Error('Network timeout'));

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(consoleErrorSpy).toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should fail open on API key error', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'invalid-key',
      };

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      mockProtect.mockRejectedValue(new Error('Invalid API key'));

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(consoleErrorSpy).toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should fail open on unexpected errors', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      mockProtect.mockRejectedValue(new Error('Unexpected error'));

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Arcjet error (failing open):',
        expect.any(Error)
      );
      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('Configuration modes', () => {
    it('should use LIVE mode by default', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '127.0.0.1',
        results: [],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(arcjet).toHaveBeenCalledWith({
        key: 'test-key',
        rules: expect.arrayContaining([
          expect.objectContaining({ type: 'detectBot' }),
          expect.objectContaining({ type: 'shield' }),
        ]),
      });
      expect(next).toHaveBeenCalled();
    });

    it('should support DRY_RUN mode', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
        mode: 'DRY_RUN',
      };

      mockProtect.mockResolvedValue({
        isDenied: () => false,
        ip: '127.0.0.1',
        results: [],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(arcjet).toHaveBeenCalledWith({
        key: 'test-key',
        rules: expect.arrayContaining([
          expect.objectContaining({ type: 'detectBot' }),
          expect.objectContaining({ type: 'shield' }),
        ]),
      });
      expect(next).toHaveBeenCalled();
    });
  });

  describe('Logging', () => {
    it('should log blocking decisions', async () => {
      const config: ArcjetConfig = {
        enabled: true,
        key: 'test-key',
      };

      const consoleLogSpy = jest.spyOn(console, 'log').mockImplementation();
      mockProtect.mockResolvedValue({
        isDenied: () => true,
        ip: '1.2.3.4',
        reason: { type: 'BOT' },
        results: [{ type: 'detectBot', denied: true }],
      });

      const middleware = createArcjetMiddleware(config);
      const req = mockRequest({ ip: '1.2.3.4' });
      const res = mockResponse();
      const next = mockNext();

      await middleware(req as any, res as any, next);

      expect(consoleLogSpy).toHaveBeenCalledWith(
        'Arcjet blocked request:',
        expect.objectContaining({
          ip: '1.2.3.4',
          reason: expect.any(Object),
          ruleResults: expect.any(Array),
          timestamp: expect.any(String),
        })
      );

      consoleLogSpy.mockRestore();
    });
  });
});
