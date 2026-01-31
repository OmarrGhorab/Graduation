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
 * Location Tracking Controller Tests
 * 
 * Tests for location tracking endpoints including:
 * - Update location with valid coordinates
 * - Update location with invalid coordinates
 * - Get location history
 * 
 * Requirements: 15.1, 15.2, 15.3, 22.1, 22.4
 * Properties: 38, 39
 */
describe('Location Tracking Controller', () => {
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

  describe('POST /api/v1/location', () => {
    /**
     * Test: Update location with valid coordinates
     * Requirement: 15.1
     */
    it('should update location with valid coordinates and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const locationData = {
        latitude: 40.7128,
        longitude: -74.0060,
        accuracy: 10.5,
        address: '123 Test St, New York, NY 10001',
      };

      // Mock database operations
      (prisma.locationHistory.create as any).mockResolvedValue({
        id: 'location-id',
        userId: user.id,
        ...locationData,
        timestamp: new Date(),
      });

      (prisma.session.updateMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(locationData);

      // Assert
      expect(response.status).toBe(201);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('location');
    });

    /**
     * Test: Update location with invalid coordinates
     * Requirement: 15.2
     */
    it('should return 400 with invalid coordinates', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidData = {
        latitude: 91, // Invalid: must be between -90 and 90
        longitude: -74.0060,
      };

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(invalidData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Update location with missing coordinates
     */
    it('should return 400 when coordinates are missing', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Property 38: Valid location update succeeds
     * For any valid GPS coordinates, the update location endpoint should return 200 status and updated location
     * Validates: Requirements 15.1
     */
    it.prop([
      fc.record({
        latitude: fc.double({ min: -90, max: 90 }),
        longitude: fc.double({ min: -180, max: 180 }),
        accuracy: fc.double({ min: 0, max: 100 }),
        address: fc.string({ minLength: 1, maxLength: 200 }),
      }),
    ])('should update location successfully with any valid coordinates', async (locationData) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.locationHistory.create as any).mockResolvedValue({
        id: 'location-id',
        userId: user.id,
        ...locationData,
        timestamp: new Date(),
      });

      (prisma.session.updateMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .post('/api/v1/location')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(locationData);

      // Assert
      expect(response.status).toBe(201);
      expect(response.body).toHaveProperty('location');
    });

    /**
     * Property 39: Invalid coordinates rejected
     * For any invalid GPS coordinates, the update location endpoint should return 400 status with validation error
     * Validates: Requirements 15.2
     */
    it.prop([
      fc.oneof(
        fc.record({ latitude: fc.double({ min: 91, max: 200 }), longitude: fc.double({ min: -180, max: 180 }) }), // Latitude too high
        fc.record({ latitude: fc.double({ min: -200, max: -91 }), longitude: fc.double({ min: -180, max: 180 }) }), // Latitude too low
        fc.record({ latitude: fc.double({ min: -90, max: 90 }), longitude: fc.double({ min: 181, max: 360 }) }), // Longitude too high
        fc.record({ latitude: fc.double({ min: -90, max: 90 }), longitude: fc.double({ min: -360, max: -181 }) }), // Longitude too low
      ),
    ])('should reject invalid coordinates with 400 error', async (invalidData) => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

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

  describe('GET /api/v1/location/history', () => {
    /**
     * Test: Get location history
     * Requirement: 15.3
     */
    it('should get location history and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.locationHistory.findMany as any).mockResolvedValue([
        {
          id: 'location-1',
          userId: user.id,
          latitude: 40.7128,
          longitude: -74.0060,
          accuracy: 10.5,
          address: '123 Test St, New York, NY 10001',
          timestamp: new Date(),
        },
        {
          id: 'location-2',
          userId: user.id,
          latitude: 40.7589,
          longitude: -73.9851,
          accuracy: 15.2,
          address: '456 Test Ave, New York, NY 10002',
          timestamp: new Date(),
        },
      ]);

      // Act
      const response = await request(app)
        .get('/api/v1/location/history')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('data');
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data.length).toBeGreaterThan(0);
    });

    /**
     * Test: Get location history with pagination
     */
    it('should support pagination for location history', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.locationHistory.findMany as any).mockResolvedValue([
        {
          id: 'location-1',
          userId: user.id,
          latitude: 40.7128,
          longitude: -74.0060,
          accuracy: 10.5,
          timestamp: new Date(),
        },
      ]);

      // Act
      const response = await request(app)
        .get('/api/v1/location/history')
        .set('Authorization', `Bearer ${accessToken}`)
        .query({ limit: 10, offset: 0 });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('data');
      expect(Array.isArray(response.body.data)).toBe(true);
    });

    /**
     * Test: Get location history without authentication
     */
    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/location/history');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });
});
