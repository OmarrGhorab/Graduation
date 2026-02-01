import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createDeactivatedUserFixture,
  createDeletedUserFixture,
  createOAuthUserFixture,
  createValidAccessToken,
  TEST_PASSWORD,
  TEST_PASSWORD_HASH,
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

// Mock Cloudinary
jest.mock('../../src/utils/cloudinaryUpload', () => ({
  __esModule: true,
  uploadImageToCloudinary: jest.fn().mockResolvedValue('https://res.cloudinary.com/test/image.jpg'),
  deleteImageFromCloudinary: jest.fn().mockResolvedValue(true),
}));


// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Account Management Controller Tests
 * 
 * Tests for account management endpoints including:
 * - Deactivate account
 * - Delete account
 * - Reactivate account
 * - Delete profile image
 * 
 * Requirements: 22.1, 22.4
 */
describe('Account Management Controller', () => {
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

    // Mock Redis operations for token revocation
    (redis.smembers as any).mockResolvedValue([]);
    (redis.del as any).mockResolvedValue(1);

    // Mock session lookup for authentication middleware
    // This allows tests to pass authentication without complex database mocking
    (prisma.session.findFirst as any).mockImplementation(async ({ where }: any) => {
      if (!where || !where.sessionToken) return null;

      // Return a mock session that matches the token's JTI
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

  describe('POST /api/v1/auth/account/deactivate', () => {
    it('should deactivate account with valid token and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, isActive: false });
      (prisma.session.updateMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/deactivate')
        .set('Authorization', `Bearer ${accessToken}`);

      // Debug: log response if it fails
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('deactivated', true);
    });

    it('should return 400 when deactivating already deactivated account', async () => {
      // Arrange
      const user = createDeactivatedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/deactivate')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toContain('already deactivated');
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/deactivate');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('POST /api/v1/auth/account/delete', () => {
    it('should delete account with valid password and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture({ password: TEST_PASSWORD_HASH });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, deletedAt: new Date(), isActive: false });
      (prisma.session.updateMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/delete')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD });

      // Debug: log response if it fails
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('deleted', true);
    });

    it('should return 401 with invalid password', async () => {
      // Arrange
      const user = createVerifiedUserFixture({ password: TEST_PASSWORD_HASH });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/delete')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: 'WrongPassword123!' });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should delete OAuth account without password and return 200', async () => {
      // Arrange
      const user = createOAuthUserFixture(); // OAuth users have no password
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, deletedAt: new Date(), isActive: false });
      (prisma.session.updateMany as any).mockResolvedValue({ count: 1 });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/delete')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({}); // No password needed for OAuth accounts

      // Debug: log response if it fails
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('deleted', true);
    });

    it('should return 400 when deleting already deleted account', async () => {
      // Arrange
      const user = createDeletedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/delete')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toContain('already deleted');
    });
  });

  describe('POST /api/v1/auth/account/confirm-reactivation', () => {
    it('should reactivate account with valid token and return 200', async () => {
      // Arrange
      const user = createDeactivatedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, isActive: true });
      (prisma.userDevice.findUnique as any).mockResolvedValue(null);
      (prisma.userDevice.create as any).mockResolvedValue({
        id: 'device-id',
        userId: user.id,
        deviceFingerprint: 'test-fingerprint',
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: true,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      });
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/confirm-reactivation')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('accountReactivated', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
      expect(response.body).toHaveProperty('user');
    });

    it('should return 400 when reactivating already active account', async () => {
      // Arrange
      const user = createVerifiedUserFixture(); // Already active
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/confirm-reactivation')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toContain('already active');
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/account/confirm-reactivation');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('DELETE /api/v1/auth/account/profile-image', () => {
    it('should delete profile image and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        profileImg: 'https://res.cloudinary.com/test/image.jpg',
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, profileImg: null });

      // Act
      const response = await request(app)
        .delete('/api/v1/auth/account/profile-image')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 200 when no profile image exists', async () => {
      // Arrange
      const user = createVerifiedUserFixture({ profileImg: null });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .delete('/api/v1/auth/account/profile-image')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('No profile image');
    });

    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .delete('/api/v1/auth/account/profile-image');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  /**
   * Property-Based Tests
   */
  describe('Property-Based Tests', () => {
    /**
     * Property 30: Invalid password for deletion rejected
     * **Validates: Requirements 9.4**
     * 
     * For any invalid password during account deletion,
     * the delete account endpoint should return 401 status with error message
     */
    it('should reject account deletion with any invalid password', async () => {
      const fc = await import('fast-check');

      await fc.assert(
        fc.asyncProperty(
          // Generate random invalid passwords (anything except TEST_PASSWORD)
          fc.string({ minLength: 1, maxLength: 50 })
            .filter(pwd => pwd !== TEST_PASSWORD),
          async (invalidPassword) => {
            // Arrange
            const user = createVerifiedUserFixture({ password: TEST_PASSWORD_HASH });
            const accessToken = createValidAccessToken({ userId: user.id });

            // Mock database operations
            (prisma.user.findUnique as any).mockResolvedValue(user);

            // Act
            const response = await request(app)
              .post('/api/v1/auth/account/delete')
              .set('Authorization', `Bearer ${accessToken}`)
              .send({ password: invalidPassword });

            // Assert
            // Should return 401 status
            expect(response.status).toBe(401);

            // Should have error message
            expect(response.body).toHaveProperty('message');
            expect(typeof response.body.message).toBe('string');

            // Clear mocks for next iteration
            jest.clearAllMocks();
          }
        ),
        { numRuns: 10 } // Run 10 times with different random inputs
      );
    });
  });
});