import { createTimeoutMiddleware } from '../../src/middleware/timeout.middleware';
import { mockRequest, mockResponse, mockNext, wait } from '../helpers/mocks';

describe('Timeout Middleware', () => {
  describe('Requests completing within timeout', () => {
    it('should allow requests that complete within timeout', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(1000);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // Apply timeout middleware
      timeoutMiddleware(req as any, res as any, next);

      // Simulate quick processing (well within timeout)
      await wait(50);

      // Apply halt-on-timeout check
      haltOnTimeout(req as any, res as any, next);

      // Should proceed normally
      expect(next).toHaveBeenCalledTimes(2); // Once for timeout middleware, once for halt check
      expect(res.status).not.toHaveBeenCalled();
      expect(res.json).not.toHaveBeenCalled();
    });

    it('should allow requests with default timeout (30 seconds)', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware();
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);
      await wait(100);
      haltOnTimeout(req as any, res as any, next);

      expect(next).toHaveBeenCalledTimes(2);
      expect(res.status).not.toHaveBeenCalled();
    });

    it('should allow requests that complete just before timeout', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(200);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);
      await wait(150); // Just before 200ms timeout
      haltOnTimeout(req as any, res as any, next);

      expect(next).toHaveBeenCalledTimes(2);
      expect(res.status).not.toHaveBeenCalled();
    });
  });

  describe('Requests exceeding timeout', () => {
    it('should return 408 when request exceeds timeout', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // Apply timeout middleware
      timeoutMiddleware(req as any, res as any, next);

      // Wait for timeout to occur
      await wait(150);

      // Manually set timedout flag (connect-timeout sets this)
      (req as any).timedout = true;

      // Apply halt-on-timeout check
      haltOnTimeout(req as any, res as any, next);

      // Should return timeout error
      expect(res.status).toHaveBeenCalledWith(408);
      expect(res.json).toHaveBeenCalledWith({
        error: 'Request Timeout',
        message: 'Request exceeded 100ms timeout',
        statusCode: 408,
        timestamp: expect.any(String),
      });
      expect(next).toHaveBeenCalledTimes(1); // Only timeout middleware calls next, halt does not
    });

    it('should include correct timeout value in error message', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(5000);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);
      (req as any).timedout = true;
      haltOnTimeout(req as any, res as any, next);

      expect(res.json).toHaveBeenCalledWith({
        error: 'Request Timeout',
        message: 'Request exceeded 5000ms timeout',
        statusCode: 408,
        timestamp: expect.any(String),
      });
    });

    it('should include valid ISO 8601 timestamp in error response', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);
      (req as any).timedout = true;
      haltOnTimeout(req as any, res as any, next);

      const jsonCall = (res.json as jest.Mock).mock.calls[0][0];
      const timestamp = jsonCall.timestamp;

      // Verify it's a valid ISO 8601 timestamp
      expect(timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
      expect(new Date(timestamp).toISOString()).toBe(timestamp);
    });
  });

  describe('Halt-on-timeout prevents further processing', () => {
    it('should not call next() when request has timed out', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);
      (req as any).timedout = true;
      haltOnTimeout(req as any, res as any, next);

      // next should only be called once (by timeout middleware), not by halt
      expect(next).toHaveBeenCalledTimes(1);
    });

    it('should prevent middleware chain continuation after timeout', async () => {
      const [timeoutMiddleware, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // Simulate middleware chain
      timeoutMiddleware(req as any, res as any, next);
      (req as any).timedout = true;

      // Halt should stop the chain
      haltOnTimeout(req as any, res as any, next);

      // Response should be sent
      expect(res.status).toHaveBeenCalledWith(408);
      expect(res.json).toHaveBeenCalled();

      // Next middleware should not be called
      const nextCallCount = (next as jest.Mock).mock.calls.length;
      expect(nextCallCount).toBe(1); // Only from timeout middleware
    });

    it('should return early when timedout flag is set', async () => {
      const [, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // Set timedout flag before halt check
      (req as any).timedout = true;

      // Call halt-on-timeout
      haltOnTimeout(req as any, res as any, next);

      // Should return immediately with error
      expect(res.status).toHaveBeenCalledWith(408);
      expect(res.json).toHaveBeenCalled();
      expect(next).not.toHaveBeenCalled();
    });

    it('should allow processing to continue when not timed out', async () => {
      const [, haltOnTimeout] = createTimeoutMiddleware(100);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // timedout flag is not set
      (req as any).timedout = false;

      haltOnTimeout(req as any, res as any, next);

      // Should call next to continue processing
      expect(next).toHaveBeenCalledTimes(1);
      expect(res.status).not.toHaveBeenCalled();
      expect(res.json).not.toHaveBeenCalled();
    });
  });

  describe('Middleware array structure', () => {
    it('should return array with two middleware functions', () => {
      const middlewares = createTimeoutMiddleware(1000);

      expect(Array.isArray(middlewares)).toBe(true);
      expect(middlewares).toHaveLength(2);
      expect(typeof middlewares[0]).toBe('function');
      expect(typeof middlewares[1]).toBe('function');
    });

    it('should return timeout middleware as first element', () => {
      const [timeoutMiddleware] = createTimeoutMiddleware(1000);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      timeoutMiddleware(req as any, res as any, next);

      // Timeout middleware should call next
      expect(next).toHaveBeenCalledTimes(1);
    });

    it('should return halt-on-timeout middleware as second element', () => {
      const [, haltOnTimeout] = createTimeoutMiddleware(1000);
      const req = mockRequest();
      const res = mockResponse();
      const next = mockNext();

      // Without timedout flag, halt should call next
      haltOnTimeout(req as any, res as any, next);
      expect(next).toHaveBeenCalledTimes(1);

      // With timedout flag, halt should not call next
      (req as any).timedout = true;
      const next2 = mockNext();
      haltOnTimeout(req as any, res as any, next2);
      expect(next2).not.toHaveBeenCalled();
    });
  });
});
