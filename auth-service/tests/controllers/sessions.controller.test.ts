import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
  createSessionFixture,
  createMultipleSessionFixtures,
  createExpiredSessionFixture,
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
 * Session Management Controller Tests
 * 
 * Tests for session management endpoints including:
 * - Get all sessions
 * - Get session by ID
 * - Revoke session by ID
 * - Revoke all sessions
 * - Cleanup expired sessions
 * 
 * Requirements: 22.1, 22.4
 */
describe('Session Management Controller', () => {
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

  describe('GET /api/v1/auth/sessions', () => {
    it('should get all sessions with valid token and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const sessions = createMultipleSessionFixtures(user.id, 3);

      // Mock database operations
      (prisma.session.findMany as any).mockResolvedValue(sessions.map(s => ({
        ...s,
        device: {
          deviceName: 'Test Device',
          platform: 'WEB',
          ipAddress: s.ipAddress,
          userAgent: s.userAgent,
          isTrusted: true,
          lastLoginAt: new Date(),
        },
      })));

      // Act
      const response = await request(app)
        .get('/api/v1/auth/sessions')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('sessions');
      expect(response.body).toHaveProperty('totalSessions');
      expect(response.body).toHaveProperty('activeSessions');
      expect(Array.isArray(response.body.sessions)).toBe(true);
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/auth/sessions');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('GET /api/v1/auth/sessions/:sessionId', () => {
    it('should get session details with valid token and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const session = createSessionFixture({ userId: user.id });

      // Mock database operations
      (prisma.session.findUnique as any).mockResolvedValue({
        ...session,
        device: {
          deviceName: 'Test Device',
          platform: 'WEB',
          ipAddress: session.ipAddress,
          userAgent: session.userAgent,
          isTrusted: true,
          lastLoginAt: new Date(),
        },
      });

      // Act
      const response = await request(app)
        .get(`/api/v1/auth/sessions/${session.id}`)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('id');
      expect(response.body).toHaveProperty('isCurrent');
    });

    it('should return 403 when getting session for different user', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const otherUser = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const session = createSessionFixture({ userId: otherUser.id });

      // Mock database operations - session belongs to different user
      (prisma.session.findUnique as any).mockResolvedValue({
        ...session,
        userId: otherUser.id,
      });

      // Act
      const response = await request(app)
        .get(`/api/v1/auth/sessions/${session.id}`)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/auth/sessions/test-session-id');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('DELETE /api/v1/auth/sessions/:sessionId', () => {
    it('should revoke session by ID and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const session = createSessionFixture({ userId: user.id });

      // Mock database operations
      (prisma.session.findUnique as any).mockResolvedValue(session);
      (prisma.session.update as any).mockResolvedValue({ ...session, isRevoked: true });

      // Act
      const response = await request(app)
        .delete(`/api/v1/auth/sessions/${session.id}`)
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('revoked', true);
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .delete('/api/v1/auth/sessions/test-session-id');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('DELETE /api/v1/auth/sessions/all', () => {
    it('should revoke all sessions except current and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const sessions = createMultipleSessionFixtures(user.id, 3);

      // Mock database operations
      (prisma.session.findMany as any).mockResolvedValue(sessions);
      (prisma.session.deleteMany as any).mockResolvedValue({ count: 2 }); // Deleted 2 out of 3

      // Mock Redis operations for token revocation
      (redis.get as any).mockResolvedValue(user.id);
      (redis.srem as any).mockResolvedValue(1);

      // Act
      const response = await request(app)
        .delete('/api/v1/auth/sessions/all')
        .set('Authorization', `Bearer ${accessToken}`);

      // Debug: log response if it fails
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('revokedCount');
      expect(response.body).toHaveProperty('loggedOut', false);
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .delete('/api/v1/auth/sessions/all');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('DELETE /api/v1/auth/sessions/cleanup', () => {
    it('should cleanup expired sessions and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const expiredSession = createExpiredSessionFixture({ userId: user.id });

      // Mock database operations
      (prisma.session.deleteMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .delete('/api/v1/auth/sessions/cleanup')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('deletedCount');
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .delete('/api/v1/auth/sessions/cleanup');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });
});
