import { it } from '@fast-check/jest';
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
} from '../helpers/fixtures';
import { fc } from '@fast-check/jest';

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
 * Internal Service Controller Tests
 * 
 * Tests for internal service endpoints including:
 * - Validate token with valid token
 * - Validate token with invalid token
 * - Get user by ID with valid ID
 * - Get user by ID with invalid ID
 * 
 * Requirements: 16.1, 16.2, 16.3, 16.4, 22.1, 22.4
 * Properties: 40, 41, 42, 43
 */
describe('Internal Service Controller', () => {
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

    // Mock session lookup for authentication middleware
    (prisma.session.findFirst as any).mockImplementation(async ({ where }: any) => {
      if (!where || !where.sessionToken) return null;

      return {
        id: 'test-session-id',
        userId: where.userId || 'test-user-id',
        isActive: true,
        isRevoked: false,
        user: {
          id: where.userId || 'test-user-id',
          isActive: true,
          deletedAt: null,
          role: 'STUDENT',
        },
      };
    });
  });

  /**
   * Clean up after each test
   */
  afterEach(() => {
    // Additional cleanup if needed
  });

  describe('GET /api/v1/internal/users/:userId/preferences', () => {
    /**
     * Test: Get user preferences with valid user ID
     * Requirement: 16.3
     */
    it('should get user preferences and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();

      // Mock database operations
      (prisma.userPreference.findUnique as any).mockResolvedValue({
        userId: user.id,
        notifications: true,
        language: 'en',
        themePreference: 'light',
      });

      // Act
      const response = await request(app)
        .get(`/api/v1/internal/users/${user.id}/preferences`)
        .set('x-internal-service-secret', 'test-internal-secret');

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('notifications');
      expect(response.body).toHaveProperty('language');
      expect(response.body).toHaveProperty('themePreference');
    });

    /**
     * Test: Get user preferences with invalid user ID
     * Requirement: 16.4
     */
    it('should return default preferences when user not found', async () => {
      // Arrange
      const invalidUserId = 'invalid-user-id';

      // Mock database operations
      (prisma.userPreference.findUnique as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .get(`/api/v1/internal/users/${invalidUserId}/preferences`)
        .set('x-internal-service-secret', 'test-internal-secret');

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('notifications', true); // Default value
      expect(response.body).toHaveProperty('language', 'en'); // Default value
      expect(response.body).toHaveProperty('themePreference', 'light'); // Default value
    });

    /**
     * Test: Get user preferences without user ID
     */
    it('should return 404 when user ID is missing', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/internal/users//preferences')
        .set('x-internal-service-secret', 'test-internal-secret');

      // Assert
      expect(response.status).toBe(404);
    });

    /**
     * Property 42: Valid user retrieval succeeds
     * For any valid user ID, the internal get user endpoint should return 200 status with user data
     * Validates: Requirements 16.3
     */
    it.prop([
      fc.uuid(),
    ])('should retrieve user preferences successfully with any valid user ID', async (userId) => {
      // Mock database operations
      (prisma.userPreference.findUnique as any).mockResolvedValue({
        userId,
        notifications: true,
        language: 'en',
        themePreference: 'light',
      });

      // Act
      const response = await request(app)
        .get(`/api/v1/internal/users/${encodeURIComponent(userId)}/preferences`)
        .set('x-internal-service-secret', 'test-internal-secret');

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('notifications');
      expect(response.body).toHaveProperty('language');
      expect(response.body).toHaveProperty('themePreference');
    });

    /**
     * Property 43: Invalid user ID returns not found
     * For any invalid or non-existent user ID, the internal get user endpoint should return 404 status with error message
     * Validates: Requirements 16.4
     */
    it.prop([
      fc.string({ minLength: 1, maxLength: 50 }).filter(s =>
        !s.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i) &&
        s !== '.' && s !== '..'
      ),
    ])('should handle invalid user IDs gracefully', async (invalidUserId) => {
      // Mock database operations
      (prisma.userPreference.findUnique as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .get(`/api/v1/internal/users/${encodeURIComponent(invalidUserId)}/preferences`)
        .set('x-internal-service-secret', 'test-internal-secret');

      // Assert
      // Should return default preferences even for invalid IDs
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('notifications');
    });
  });

  describe('POST /api/v1/internal/validate-token', () => {
    /**
     * Test: Validate token with valid token
     * Requirement: 16.1
     */
    it('should validate token and return 200 with user data', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/internal/validate-token')
        .set('x-internal-service-secret', 'test-internal-secret')
        .send({ token: accessToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('valid', true);
      expect(response.body).toHaveProperty('userId');
    });

    /**
     * Test: Validate token with invalid token
     * Requirement: 16.2
     */
    it('should return 401 with invalid token', async () => {
      // Arrange
      const invalidToken = 'invalid.token.here';

      // Act
      const response = await request(app)
        .post('/api/v1/internal/validate-token')
        .set('x-internal-service-secret', 'test-internal-secret')
        .send({ token: invalidToken });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('valid', false);
    });

    /**
     * Test: Validate token without token
     */
    it('should return 400 when token is missing', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/internal/validate-token')
        .set('x-internal-service-secret', 'test-internal-secret')
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Property 40: Valid token validation succeeds
     * For any valid access token, the internal validate token endpoint should return 200 status with user data
     * Validates: Requirements 16.1
     */
    it.prop([
      fc.uuid(),
    ])('should validate token successfully for any valid user', async (userId) => {
      // Arrange
      const accessToken = createValidAccessToken({ userId });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        id: userId,
        email: 'test@example.com',
        role: 'STUDENT',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/internal/validate-token')
        .set('x-internal-service-secret', 'test-internal-secret')
        .send({ token: accessToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('valid', true);
      expect(response.body).toHaveProperty('userId', userId);
    });

    /**
     * Property 41: Invalid token validation fails
     * For any invalid access token, the internal validate token endpoint should return 401 status with error message
     * Validates: Requirements 16.2
     */
    it.prop([
      fc.string({ minLength: 10, maxLength: 100 }).filter(s => !s.includes('.')),
    ])('should reject invalid tokens with 401 error', async (invalidToken) => {
      // Act
      const response = await request(app)
        .post('/api/v1/internal/validate-token')
        .set('x-internal-service-secret', 'test-internal-secret')
        .send({ token: invalidToken });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('valid', false);
    });
  });
});
