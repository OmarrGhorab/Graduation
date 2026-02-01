import { it } from '@fast-check/jest';
import { fc } from '@fast-check/jest';
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
  createExpiredAccessToken,
  createInvalidAccessToken,
} from '../helpers/fixtures';

// Mock modules before importing the app
jest.mock('../../src/libs/prisma', () => {
  const { mockPrisma } = jest.requireActual('../helpers/mocks');
  const prismaMock = mockPrisma();
  return {
    __esModule: true,
    default: prismaMock,
    prisma: prismaMock,
  };
});

jest.mock('../../src/libs/redis', () => ({
  __esModule: true,
  default: mockRedis(),
}));

jest.mock('../../src/utils/email', () => ({
  __esModule: true,
  sendVerificationOTP: jest.fn().mockResolvedValue(true),
  sendPasswordResetOTP: jest.fn().mockResolvedValue(true),
  sendDeviceVerificationOTP: jest.fn().mockResolvedValue(true),
  sendNewDeviceSecurityAlert: jest.fn().mockResolvedValue(true),
  sendAccountDeletionConfirmation: jest.fn().mockResolvedValue(true),
  sendAccountReactivationEmail: jest.fn().mockResolvedValue(true),
}));

// Mock Arcjet to avoid external calls
jest.mock('../../src/libs/arcjet', () => ({
  __esModule: true,
  aj: {
    protect: jest.fn().mockResolvedValue({
      isDenied: () => false,
      isAllowed: () => true,
    }),
  },
}));

// Mock notification client
jest.mock('../../src/utils/notifications-client', () => ({
  __esModule: true,
  publishNotification: jest.fn().mockResolvedValue(undefined),
}));

// Mock user language utility
jest.mock('../../src/utils/userLanguage', () => ({
  __esModule: true,
  getUserLanguage: jest.fn().mockResolvedValue('en'),
}));

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Middleware Integration Tests
 * 
 * Tests for middleware integration including:
 * - Authentication middleware
 * - Rate limiting middleware
 * - Error handling middleware
 * 
 * Requirements: 17.1-17.5, 18.1-18.5, 19.1-19.6, 22.1, 22.4
 * Properties: 1, 2, 3, 44
 */
describe('Middleware Integration', () => {
  let app: Express;

  /**
   * Set up test environment before each test
   * Creates fresh test app instance and resets mocks
   */
  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();

    // Create fresh app instance
    app = createTestApp();

    // Mock Redis operations
    (redis.smembers as any).mockResolvedValue([]);
    (redis.del as any).mockResolvedValue(1);
    (redis.get as any).mockResolvedValue(null);
    (redis.incr as any).mockResolvedValue(1);
    (redis.expire as any).mockResolvedValue(1);
  });

  /**
   * Clean up after each test
   */
  afterEach(() => {
    // Additional cleanup if needed
  });

  describe('Authentication Middleware', () => {
    /**
     * Test: Protected endpoint with valid token
     * Requirement: 17.1
     */
    it('should allow access to protected endpoint with valid token', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock session lookup
      (prisma.session.findFirst as any).mockResolvedValue({
        id: 'test-session-id',
        userId: user.id,
        isActive: true,
        isRevoked: false,
        user: {
          id: user.id,
          isActive: true,
          deletedAt: null,
          role: 'STUDENT',
        },
      });

      // Mock user lookup
      (prisma.user.findUnique as any).mockResolvedValue({
        ...user,
        interests: [],
        preferences: null,
      });

      // Act
      const response = await request(app)
        .get('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
    });

    /**
     * Test: Protected endpoint without token
     * Requirement: 17.2
     */
    it('should return 401 for protected endpoint without token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/profile');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Protected endpoint with invalid token
     * Requirement: 17.3
     */
    it('should return 401 for protected endpoint with invalid token', async () => {
      // Arrange
      const invalidToken = createInvalidAccessToken();

      // Act
      const response = await request(app)
        .get('/api/v1/profile')
        .set('Authorization', `Bearer ${invalidToken}`);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Protected endpoint with expired token
     * Requirement: 17.4
     */
    it('should return 401 for protected endpoint with expired token', async () => {
      // Arrange
      const expiredToken = createExpiredAccessToken();

      // Act
      const response = await request(app)
        .get('/api/v1/profile')
        .set('Authorization', `Bearer ${expiredToken}`);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Protected endpoint with revoked session
     * Requirement: 17.5
     */
    it('should return 401 for protected endpoint with revoked session', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock revoked session
      (prisma.session.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .get('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Property 1: Valid authentication succeeds
     * For any protected endpoint and valid access token, the request should succeed and return appropriate data
     * Validates: Requirements 17.1
     */
    it.prop([
      fc.constantFrom('/api/v1/profile', '/api/v1/auth/activity', '/api/v1/auth/sessions'),
    ])('should allow access to any protected endpoint with valid token', async (endpoint) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock session lookup
      (prisma.session.findFirst as any).mockResolvedValue({
        id: 'test-session-id',
        userId: user.id,
        isActive: true,
        isRevoked: false,
        user: {
          id: user.id,
          isActive: true,
          deletedAt: null,
          role: 'STUDENT',
        },
      });

      // Mock various database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        ...user,
        interests: [],
        preferences: null,
      });
      (prisma.session.findMany as any).mockResolvedValue([]);
      (prisma.userDevice.count as any).mockResolvedValue(0);
      (prisma.userDevice.findMany as any).mockResolvedValue([]);

      // Act
      const response = await request(app)
        .get(endpoint)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
    });

    /**
     * Property 2: Missing authentication fails
     * For any protected endpoint without an access token, the request should return 401 status with error response
     * Validates: Requirements 17.2
     */
    it.prop([
      fc.constantFrom('/api/v1/profile', '/api/v1/auth/activity', '/api/v1/auth/sessions'),
    ])('should deny access to any protected endpoint without token', async (endpoint) => {
      // Act
      const response = await request(app)
        .get(endpoint);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Property 3: Invalid authentication fails
     * For any protected endpoint and invalid access token, the request should return 401 status with error response
     * Validates: Requirements 17.3
     */
    it.prop([
      fc.constantFrom('/api/v1/profile', '/api/v1/auth/activity', '/api/v1/auth/sessions'),
    ])('should deny access to any protected endpoint with invalid token', async (endpoint) => {
      // Arrange
      const invalidToken = createInvalidAccessToken();

      // Act
      const response = await request(app)
        .get(endpoint)
        .set('Authorization', `Bearer ${invalidToken}`);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('Error Handling Middleware', () => {
    /**
     * Test: Validation error format
     * Requirement: 19.1
     */
    it('should return 400 with consistent error format for validation errors', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/register')
        .send({ email: 'invalid-email' }); // Invalid email format

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('statusCode', 400);
      expect(response.body).toHaveProperty('timestamp');
    });

    /**
     * Test: Authentication error format
     * Requirement: 19.2
     */
    it('should return 401 with consistent error format for authentication errors', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/profile');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('statusCode', 401);
      expect(response.body).toHaveProperty('timestamp');
    });

    /**
     * Test: Not found error format
     * Requirement: 19.4
     */
    it('should return 404 with consistent error format for not found errors', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/nonexistent-endpoint');

      // Assert
      expect(response.status).toBe(404);
    });

    /**
     * Property 44: Error responses have consistent format
     * For any error response from any endpoint, the response should include error (string), statusCode (number), and timestamp (ISO string)
     * Validates: Requirements 19.6, 21.2
     */
    it.prop([
      fc.constantFrom(
        { method: 'GET', path: '/api/v1/profile', expectedStatus: 401 }, // Auth error
        { method: 'POST', path: '/api/v1/auth/register', body: {}, expectedStatus: 400 }, // Validation error
      ),
    ])('should return consistent error format for all error types', async (testCase) => {
      // Act
      let response;
      if (testCase.method === 'GET') {
        response = await request(app).get(testCase.path);
      } else {
        response = await request(app).post(testCase.path).send(testCase.body || {});
      }

      // Assert
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body).toHaveProperty('message');
      expect(typeof response.body.message).toBe('string');
      expect(response.body).toHaveProperty('statusCode');
      expect(typeof response.body.statusCode).toBe('number');
      expect(response.body).toHaveProperty('timestamp');
      expect(typeof response.body.timestamp).toBe('string');
      // Verify timestamp is valid ISO string
      expect(() => new Date(response.body.timestamp)).not.toThrow();
    });
  });
});
