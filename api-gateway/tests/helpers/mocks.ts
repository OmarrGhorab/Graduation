import { Request, Response, NextFunction } from 'express';

/**
 * Mock Express Request object for testing
 */
export function mockRequest(overrides: Partial<Request> = {}): Partial<Request> {
  const listeners: { [event: string]: Function[] } = {};

  const req: Partial<Request> = {
    body: {},
    params: {},
    query: {},
    headers: {},
    method: 'GET',
    path: '/',
    url: '/',
    ip: '127.0.0.1',
    get: jest.fn((header: string) => {
      const value = req.headers?.[header.toLowerCase()];
      // Handle set-cookie header which returns string[] | undefined
      if (header.toLowerCase() === 'set-cookie') {
        return Array.isArray(value) ? value : undefined;
      }
      // All other headers return string | undefined
      return Array.isArray(value) ? value[0] : value;
    }) as Request['get'],
    // Add event emitter methods for connect-timeout
    on: jest.fn((event: string, callback: Function) => {
      if (!listeners[event]) {
        listeners[event] = [];
      }
      listeners[event].push(callback);
      return req as any;
    }),
    emit: jest.fn((event: string, ...args: any[]) => {
      if (listeners[event]) {
        listeners[event].forEach(callback => callback(...args));
      }
      return true;
    }),
    removeListener: jest.fn((event: string, callback: Function) => {
      if (listeners[event]) {
        listeners[event] = listeners[event].filter(cb => cb !== callback);
      }
      return req as any;
    }),
    ...overrides,
  };
  return req;
}

/**
 * Mock Express Response object for testing
 */
export function mockResponse(): Partial<Response> {
  const res: Partial<Response> = {
    status: jest.fn().mockReturnThis(),
    json: jest.fn().mockReturnThis(),
    send: jest.fn().mockReturnThis(),
    sendStatus: jest.fn().mockReturnThis(),
    set: jest.fn().mockReturnThis(),
    setHeader: jest.fn().mockReturnThis(),
    getHeader: jest.fn(),
    removeHeader: jest.fn(),
    end: jest.fn().mockReturnThis(),
    locals: {},
  };
  return res;
}

/**
 * Mock Express NextFunction for testing
 */
export function mockNext(): NextFunction {
  return jest.fn();
}

/**
 * Mock fetch for testing HTTP calls
 */
export function mockFetch(response: {
  ok?: boolean;
  status?: number;
  statusText?: string;
  json?: () => Promise<any>;
  text?: () => Promise<string>;
}): jest.Mock {
  const defaultResponse = {
    ok: true,
    status: 200,
    statusText: 'OK',
    json: async () => ({}),
    text: async () => '',
    ...response,
  };

  return jest.fn().mockResolvedValue(defaultResponse);
}

/**
 * Mock Arcjet decision for testing
 */
export interface MockArcjetDecision {
  isErrored: () => boolean;
  isDenied: () => boolean;
  isAllowed: () => boolean;
  reason?: {
    isBot?: () => boolean;
    isVpn?: () => boolean;
  };
}

export function mockArcjetDecision(overrides: Partial<MockArcjetDecision> = {}): MockArcjetDecision {
  return {
    isErrored: jest.fn().mockReturnValue(false),
    isDenied: jest.fn().mockReturnValue(false),
    isAllowed: jest.fn().mockReturnValue(true),
    ...overrides,
  };
}

/**
 * Mock Arcjet protect function
 */
export function mockArcjetProtect(decision: MockArcjetDecision = mockArcjetDecision()) {
  return jest.fn().mockResolvedValue(decision);
}

/**
 * Create a mock configuration for testing
 */
export function mockConfig(overrides: any = {}) {
  return {
    server: {
      port: 3000,
      nodeEnv: 'test' as const,
      isProd: false,
    },
    cors: {
      allowedOrigins: ['http://localhost:3000'],
      credentials: true,
      allowedHeaders: ['Content-Type', 'Authorization'],
    },
    services: {
      auth: [{
        name: 'auth-service',
        url: 'http://localhost:3001',
        healthPath: '/health',
      }],
      notification: [{
        name: 'notification-service',
        url: 'http://localhost:3002',
        healthPath: '/health',
      }],
      chat: [{
        name: 'chat-service',
        url: 'http://localhost:3003',
        healthPath: '/health',
      }],
      ws: [{
        name: 'ws-gateway',
        url: 'http://localhost:8001',
        healthPath: '/health',
      }],
    },
    security: {
      arcjetKey: 'test-key',
      arcjetEnabled: false,
    },
    ...overrides,
  };
}

/**
 * Wait for a specified amount of time (useful for timeout tests)
 */
export function wait(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Create a mock service endpoint response
 */
export function mockServiceResponse(overrides: {
  status?: number;
  data?: any;
  delay?: number;
} = {}) {
  const { status = 200, data = { status: 'ok' }, delay = 0 } = overrides;

  return async () => {
    if (delay > 0) {
      await wait(delay);
    }
    return {
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 200 ? 'OK' : 'Error',
      json: async () => data,
      text: async () => JSON.stringify(data),
    };
  };
}
