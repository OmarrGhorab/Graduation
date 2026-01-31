import request from 'supertest';
import { createTestApp } from './testApp';
import { mockPrisma, mockRedis } from './mocks';

// Mock dependencies
// Mock dependencies
jest.mock('../../src/libs/prisma', () => {
  const { mockPrisma } = jest.requireActual('./mocks');
  const prismaMock = mockPrisma();
  return {
    __esModule: true,
    default: prismaMock,
    prisma: prismaMock,
  };
});

jest.mock('../../src/libs/redis', () => {
  const { mockRedis } = jest.requireActual('./mocks');
  return {
    __esModule: true,
    default: mockRedis(),
  };
});

jest.mock('../../src/libs/arcjet', () => ({
  __esModule: true,
  aj: {
    protect: jest.fn().mockResolvedValue({
      isDenied: () => false,
      isAllowed: () => true,
    }),
  },
}));

jest.mock('../../src/utils/email', () => ({
  __esModule: true,
  sendVerificationOTP: jest.fn().mockResolvedValue(true),
  sendPasswordResetOTP: jest.fn().mockResolvedValue(true),
}));

describe('Test App Factory', () => {
  describe('createTestApp', () => {
    it('should create an Express app instance', () => {
      const app = createTestApp();
      expect(app).toBeDefined();
      expect(typeof app).toBe('function'); // Express app is a function
    });

    it('should respond to health check endpoint', async () => {
      const app = createTestApp();
      const response = await request(app).get('/health');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('status', 'ok');
      expect(response.body).toHaveProperty('service', 'auth-service-test');
      expect(response.body).toHaveProperty('timestamp');
    });

    it('should respond to root endpoint', async () => {
      const app = createTestApp();
      const response = await request(app).get('/');

      expect(response.status).toBe(200);
      expect(response.text).toBe('auth service test app is running');
    });

    it('should have auth routes mounted', async () => {
      const app = createTestApp();
      // Test that auth routes exist (will fail with 400/401 but not 404)
      const response = await request(app).post('/api/v1/auth/register');

      // Should not be 404 (route exists)
      expect(response.status).not.toBe(404);
    });

    it('should have onboarding routes mounted', async () => {
      const app = createTestApp();
      const response = await request(app).post('/api/v1/onboarding');

      // Should not be 404 (route exists, will return 401 without auth)
      expect(response.status).not.toBe(404);
    });

    it('should have profile routes mounted', async () => {
      const app = createTestApp();
      const response = await request(app).get('/api/v1/profile');

      // Should not be 404 (route exists, will return 401 without auth)
      expect(response.status).not.toBe(404);
    });

    it('should have location routes mounted', async () => {
      const app = createTestApp();
      const response = await request(app).post('/api/v1/location');

      // Should not be 404 (route exists, will return 401 without auth)
      expect(response.status).not.toBe(404);
    });

    it('should have internal routes mounted', async () => {
      const app = createTestApp();
      const response = await request(app).get('/api/v1/internal/users/123/preferences');

      // Should not be 404 (route exists, will return 401 without auth)
      expect(response.status).not.toBe(404);
    });

    it('should have parent-link routes mounted', async () => {
      const app = createTestApp();
      const response = await request(app).get('/api/v1/parent-link/search');

      // Should not be 404 (route exists, will return 401 without auth)
      expect(response.status).not.toBe(404);
    });

    it('should handle 404 for unknown routes', async () => {
      const app = createTestApp();
      const response = await request(app).get('/api/v1/unknown-route');

      expect(response.status).toBe(404);
    });

    it('should accept options parameter', () => {
      const app = createTestApp({
        skipAuth: true,
        skipRateLimit: true,
      });

      expect(app).toBeDefined();
    });

    it('should accept custom mocks in options', () => {
      const mockPrisma = { user: { findUnique: () => { } } };
      const mockRedis = { get: () => { }, set: () => { } };

      const app = createTestApp({
        customMocks: {
          prisma: mockPrisma,
          redis: mockRedis,
        },
      });

      expect(app).toBeDefined();
    });
  });
});
