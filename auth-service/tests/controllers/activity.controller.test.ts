import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
  createSessionFixture,
  createMultipleSessionFixtures,
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
 * Activity Tracking Controller Tests
 * 
 * Tests for activity tracking endpoints including:
 * - Get activity with valid token
 * - Get activity without token
 * - Get activity with pagination
 * 
 * Requirements: 11.1, 11.2, 11.3, 22.1, 22.4
 */
describe('Activity Tracking Controller', () => {
  let app: Express;

  /**
   * Set up test environment before each test
   * Creates fresh test app instance and resets mocks
   */
  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();

    // Ensure userDevice mock methods exist after clearing
    if (!prisma.userDevice || !prisma.userDevice.count) {
      (prisma as any).userDevice = {
        count: jest.fn().mockResolvedValue(0),
        findMany: jest.fn().mockResolvedValue([]),
        findUnique: jest.fn().mockResolvedValue(null),
        findFirst: jest.fn().mockResolvedValue(null),
        create: jest.fn().mockResolvedValue({}),
        update: jest.fn().mockResolvedValue({}),
        upsert: jest.fn().mockResolvedValue({}),
      };
    }

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

  describe('GET /api/v1/auth/activity', () => {
    /**
     * Test: Get activity with valid token
     * Requirement: 11.1
     */
    it('should get activity with valid token and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const sessions = createMultipleSessionFixtures(user.id, 3);

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        id: user.id,
        lastLoginAt: new Date(),
        createdAt: user.createdAt,
      });

      (prisma.session.findMany as any).mockImplementation(async ({ where, take }: any) => {
        if (take === 5) {
          // Recent sessions query
          return sessions.slice(0, 5).map(s => ({
            ...s,
            device: {
              deviceName: 'Test Device',
              platform: 'WEB',
            },
          }));
        }
        // Active sessions query
        return sessions.map(s => ({
          ...s,
          device: {
            deviceName: 'Test Device',
            platform: 'WEB',
            isTrusted: true,
          },
        }));
      });

      (prisma.userDevice.count as any).mockResolvedValue(2);
      (prisma.userDevice.findMany as any).mockResolvedValue([
        {
          id: 'device-1',
          deviceName: 'Test Device 1',
          platform: 'WEB',
          isTrusted: true,
          lastLoginAt: new Date(),
          createdAt: new Date(),
        },
        {
          id: 'device-2',
          deviceName: 'Test Device 2',
          platform: 'MOBILE',
          isTrusted: false,
          lastLoginAt: new Date(),
          createdAt: new Date(),
        },
      ]);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/activity')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('account');
      expect(response.body).toHaveProperty('currentDevice');
      expect(response.body).toHaveProperty('sessions');
      expect(response.body).toHaveProperty('devices');
      expect(response.body).toHaveProperty('recentActivity');
      expect(response.body.account).toHaveProperty('lastLoginAt');
      expect(response.body.account).toHaveProperty('accountCreatedAt');
      expect(response.body.sessions).toHaveProperty('totalActive');
      expect(response.body.sessions).toHaveProperty('byPlatform');
      expect(response.body.devices).toHaveProperty('total');
      expect(response.body.devices).toHaveProperty('trusted');
      expect(Array.isArray(response.body.recentActivity)).toBe(true);
    });

    /**
     * Test: Get activity without token
     * Requirement: 11.2
     */
    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/auth/activity');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Get activity with pagination (implicit in recent activity)
     * Requirement: 11.3
     */
    it('should return paginated recent activity (last 5 sessions)', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const sessions = createMultipleSessionFixtures(user.id, 10); // Create 10 sessions

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        id: user.id,
        lastLoginAt: new Date(),
        createdAt: user.createdAt,
      });

      (prisma.session.findMany as any).mockImplementation(async ({ where, take }: any) => {
        if (take === 5) {
          // Recent sessions query - should return only 5
          return sessions.slice(0, 5).map(s => ({
            ...s,
            device: {
              deviceName: 'Test Device',
              platform: 'WEB',
            },
          }));
        }
        // Active sessions query
        return sessions.map(s => ({
          ...s,
          device: {
            deviceName: 'Test Device',
            platform: 'WEB',
            isTrusted: true,
          },
        }));
      });

      (prisma.userDevice.count as any).mockResolvedValue(2);
      (prisma.userDevice.findMany as any).mockResolvedValue([]);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/activity')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('recentActivity');
      expect(Array.isArray(response.body.recentActivity)).toBe(true);
      expect(response.body.recentActivity.length).toBeLessThanOrEqual(5);
    });

    /**
     * Test: Activity response includes device information
     */
    it('should include device information in activity response', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        id: user.id,
        lastLoginAt: new Date(),
        createdAt: user.createdAt,
      });

      (prisma.session.findMany as any).mockResolvedValue([]);
      (prisma.userDevice.count as any).mockResolvedValue(1);
      (prisma.userDevice.findMany as any).mockResolvedValue([
        {
          id: 'device-1',
          deviceName: 'Test Device',
          platform: 'WEB',
          isTrusted: true,
          lastLoginAt: new Date(),
          createdAt: new Date(),
        },
      ]);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/activity')
        .set('Authorization', `Bearer ${accessToken}`)
        .set('User-Agent', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124');

      // Assert
      expect(response.status).toBe(200);
      expect(response.body.currentDevice).toHaveProperty('deviceName');
      expect(response.body.currentDevice).toHaveProperty('platform');
      expect(response.body.currentDevice).toHaveProperty('ipAddress');
      expect(response.body.devices.list).toHaveLength(1);
      expect(response.body.devices.list[0]).toHaveProperty('name');
      expect(response.body.devices.list[0]).toHaveProperty('platform');
      expect(response.body.devices.list[0]).toHaveProperty('isTrusted');
    });

    /**
     * Test: Activity response includes session summary
     */
    it('should include session summary by platform', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        id: user.id,
        lastLoginAt: new Date(),
        createdAt: user.createdAt,
      });

      (prisma.session.findMany as any).mockImplementation(async ({ where, take }: any) => {
        if (take === 5) {
          return [];
        }
        // Return sessions from different platforms
        return [
          {
            id: 'session-1',
            userId: user.id,
            isActive: true,
            isRevoked: false,
            expiresAt: new Date(Date.now() + 3600000),
            lastActivityAt: new Date(),
            device: {
              deviceName: 'Web Browser',
              platform: 'WEB',
              isTrusted: true,
            },
          },
          {
            id: 'session-2',
            userId: user.id,
            isActive: true,
            isRevoked: false,
            expiresAt: new Date(Date.now() + 3600000),
            lastActivityAt: new Date(),
            device: {
              deviceName: 'Mobile App',
              platform: 'MOBILE',
              isTrusted: true,
            },
          },
        ];
      });

      (prisma.userDevice.count as any).mockResolvedValue(0);
      (prisma.userDevice.findMany as any).mockResolvedValue([]);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/activity')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body.sessions).toHaveProperty('totalActive', 2);
      expect(response.body.sessions).toHaveProperty('byPlatform');
      expect(response.body.sessions.byPlatform).toHaveProperty('WEB', 1);
      expect(response.body.sessions.byPlatform).toHaveProperty('MOBILE', 1);
    });
  });
});
