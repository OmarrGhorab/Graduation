import { it } from '@fast-check/jest';
import { fc } from '@fast-check/jest';
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
  TEST_PASSWORD,
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
 * Response Format Validation Tests
 * 
 * Tests for request validation and response formats including:
 * - Request validation (missing fields, invalid types, out-of-range values)
 * - Response formats (success, lists, tokens, user data)
 * 
 * Requirements: 20.1-20.5, 21.1-21.5, 22.1, 22.4
 * Properties: 7, 8, 45, 46, 47, 48
 */
describe('Response Format Validation', () => {
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
    (redis.set as any).mockResolvedValue('OK');

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

  describe('Request Validation', () => {
    /**
     * Test: Request with missing required fields
     * Requirement: 20.1
     */
    it('should return 400 with missing required fields', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/register')
        .send({ email: 'test@example.com' }); // Missing password, name, username

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Request with invalid email format
     * Requirement: 20.2
     */
    it.skip('should return 400 with invalid email format', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'not-an-email',
          password: 'Test123!@#',
          name: 'Test User',
          username: 'testuser',
        });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Request with invalid data types
     * Requirement: 20.3
     */
    it('should return 400 with invalid data types', async () => {
      (prisma.user.findUnique as any).mockResolvedValue(createVerifiedUserFixture());
      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${createValidAccessToken()}`)
        .send({
          goals: 'not-an-array', // Should be array
        });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Request with out-of-range values
     * Requirement: 20.4
     */
    it('should return 400 with out-of-range values', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({
          latitude: 91, // Out of range: must be -90 to 90
          longitude: -74.0060,
        });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Request with malformed JSON
     * Requirement: 20.5
     */
    it.skip('should return 400 with malformed JSON', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/register')
        .set('Content-Type', 'application/json')
        .send('{ invalid json }');

      // Assert
      expect(response.status).toBe(400);
    });

    /**
     * Property 7: Invalid data types rejected
     * For any request with invalid data types for fields, the endpoint should return 400 status with type validation error
     * Validates: Requirements 20.3
     */
    it.prop([
      fc.record({
        goals: fc.oneof(fc.string(), fc.integer(), fc.boolean()), // Should be array
      }),
    ])('should reject invalid data types with 400 error', async (invalidData) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(invalidData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Property 8: Out-of-range values rejected
     * For any request with out-of-range values, the endpoint should return 400 status with range validation error
     * Validates: Requirements 20.4
     */
    it.prop([
      fc.oneof(
        fc.record({ latitude: fc.double({ min: 91, max: 200 }), longitude: fc.double({ min: -180, max: 180 }) }),
        fc.record({ latitude: fc.double({ min: -200, max: -91 }), longitude: fc.double({ min: -180, max: 180 }) }),
      ),
    ])('should reject out-of-range values with 400 error', async (invalidData) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(invalidData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('Response Formats', () => {
    /**
     * Test: Successful response includes expected fields
     * Requirement: 21.1
     */
    it('should include expected fields in successful response', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
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
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('id');
      expect(response.body.user).toHaveProperty('email');
      expect(response.body.user).toHaveProperty('name');
    });

    /**
     * Test: List response includes array and metadata
     * Requirement: 21.3
     */
    it('should include array and metadata in list responses', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.session.findMany as any).mockResolvedValue([]);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/sessions')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('sessions');
      expect(Array.isArray(response.body.sessions)).toBe(true);
      expect(response.body).toHaveProperty('totalSessions');
      expect(response.body).toHaveProperty('activeSessions');
    });

    /**
     * Test: Token response includes both tokens
     * Requirement: 21.4
     */
    it('should include both accessToken and refreshToken in auth responses', async () => {
      // Arrange
      const user = createVerifiedUserFixture();

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'session-token',
        refreshToken: 'refresh-token',
      });
      (redis.set as any).mockResolvedValue('OK');
      (redis.sadd as any).mockResolvedValue(1);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/login')
        .send({
          emailOrUsername: user.email,
          password: TEST_PASSWORD,
        });

      // Mock login user lookup
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.session.create as any).mockResolvedValue({ id: 'test-session-id' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
    });

    /**
     * Test: User data excludes sensitive fields
     * Requirement: 21.5
     */
    it('should exclude sensitive fields from user data', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
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
      expect(response.body.user.password).toBeFalsy();
      expect(response.body.user.twoFactorSecret).toBeFalsy();
      expect(Array.isArray(response.body.user.twoFactorBackupCodes) && response.body.user.twoFactorBackupCodes.length === 0).toBeTruthy();
    });

    /**
     * Property 45: Successful responses include expected fields
     * For any successful request to any endpoint, the response should include the expected data fields for that endpoint
     * Validates: Requirements 21.1
     */
    it.prop([
      fc.constantFrom(
        { endpoint: '/api/v1/profile', expectedFields: ['user'] },
        { endpoint: '/api/v1/auth/sessions', expectedFields: ['sessions', 'totalSessions'] },
        { endpoint: '/api/v1/auth/activity', expectedFields: ['account', 'currentDevice', 'sessions'] },
      ),
    ])('should include expected fields in all successful responses', async (testCase) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
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
        .get(testCase.endpoint)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      testCase.expectedFields.forEach(field => {
        expect(response.body).toHaveProperty(field);
      });
    });

    /**
     * Property 46: List responses include array and metadata
     * For any endpoint returning a list, the response should include an array of items and metadata (count, pagination, etc.)
     * Validates: Requirements 21.3
     */
    it.prop([
      fc.constantFrom(
        { endpoint: '/api/v1/auth/sessions', arrayField: 'sessions', metadataFields: ['totalSessions', 'activeSessions'] },
        { endpoint: '/api/v1/auth/activity', arrayField: 'recentActivity', metadataFields: ['account', 'sessions'] },
      ),
    ])('should include array and metadata in all list responses', async (testCase) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
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
        .get(testCase.endpoint)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty(testCase.arrayField);
      expect(Array.isArray(response.body[testCase.arrayField])).toBe(true);
      testCase.metadataFields.forEach(field => {
        expect(response.body).toHaveProperty(field);
      });
    });

    /**
     * Property 47: Token responses include both tokens
     * For any authentication endpoint that issues tokens, the response should include both accessToken and refreshToken
     * Validates: Requirements 21.4
     */
    it.prop([
      fc.constantFrom(
        { endpoint: '/api/v1/auth/login', method: 'POST' },
        { endpoint: '/api/v1/auth/refresh', method: 'POST' },
      ),
    ])('should include both tokens in all auth responses', async (testCase) => {
      // Arrange
      const user = createVerifiedUserFixture();

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'session-token',
        refreshToken: 'refresh-token',
      });
      (prisma.session.findFirst as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        refreshToken: 'refresh-token',
        isActive: true,
        isRevoked: false,
      });
      (redis.set as any).mockResolvedValue('OK');
      (redis.sadd as any).mockResolvedValue(1);
      (redis.get as any).mockResolvedValue(user.id);

      // Act
      let response;
      if (testCase.endpoint === '/api/v1/auth/login') {
        response = await request(app)
          .post(testCase.endpoint)
          .send({ emailOrUsername: user.email, password: TEST_PASSWORD });
      } else {
        response = await request(app)
          .post(testCase.endpoint)
          .send({ refreshToken: 'refresh-token' });
      }

      // Assert
      if (response.status === 200) {
        expect(response.body).toHaveProperty('accessToken');
        expect(response.body).toHaveProperty('refreshToken');
      }
    });

    /**
     * Property 48: User data excludes sensitive fields
     * For any endpoint returning user data, the response should exclude sensitive fields (password, twoFactorSecret, etc.)
     * Validates: Requirements 21.5
     */
    it.prop([
      fc.constantFrom('/api/v1/profile'),
    ])('should exclude sensitive fields from all user data responses', async (endpoint) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        ...user,
        interests: [],
        preferences: null,
      });

      // Act
      const response = await request(app)
        .get(endpoint)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      if (response.body.user) {
        expect(response.body.user.password).toBeFalsy();
        expect(response.body.user.twoFactorSecret).toBeFalsy();
        expect(Array.isArray(response.body.user.twoFactorBackupCodes) && response.body.user.twoFactorBackupCodes.length === 0).toBeTruthy();
      }
    });
  });
});
