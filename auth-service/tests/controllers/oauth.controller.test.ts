
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createOAuthUserFixture,
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

// Mock username generation
jest.mock('../../src/utils/username', () => ({
  __esModule: true,
  generateUniqueUsername: jest.fn().mockImplementation((base: string) => {
    return Promise.resolve(`${base}_${Date.now()}`);
  }),
}));

// Mock OTP utilities
jest.mock('../../src/utils/otp', () => ({
  __esModule: true,
  createAndStoreOtp: jest.fn().mockResolvedValue('123456'),
  verifyOtp: jest.fn().mockResolvedValue(true),
}));

// Mock Google OAuth2Client
const mockVerifyIdToken = jest.fn();

jest.mock('google-auth-library', () => {
  return {
    __esModule: true,
    OAuth2Client: class MockOAuth2Client {
      verifyIdToken = mockVerifyIdToken;
    },
  };
});

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import * as usernameUtils from '../../src/utils/username';
import prisma from '../../src/libs/prisma';

/**
 * OAuth Controller Tests
 * 
 * Tests for OAuth authentication endpoints including:
 * - Google mobile authentication
 * - New user creation via OAuth
 * - Existing user login via OAuth
 * 
 * Requirements: 22.1, 22.4
 */
describe('OAuth Controller', () => {
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
  });

  /**
   * Clean up after each test
   */
  afterEach(() => {
    // Additional cleanup can be added here if needed
  });

  describe('POST /api/v1/auth/google/mobile', () => {
    beforeEach(() => {
      // Reset the mock before each test
      mockVerifyIdToken.mockReset();

      // Default mock implementation
      mockVerifyIdToken.mockResolvedValue({
        getPayload: () => ({
          sub: 'google-user-id-123',
          email: 'test@example.com',
          name: 'Test User',
          picture: 'https://example.com/picture.jpg',
          email_verified: true,
        }),
      });
    });

    it('should authenticate with valid Google ID token and return 200', async () => {
      // Arrange
      const email = 'test@example.com';
      const googleId = 'google-user-id-123';
      const idToken = 'valid-google-id-token';

      const user = createOAuthUserFixture({
        email,
        name: 'Test User',
        profileImg: 'https://example.com/picture.jpg',
      });

      const authProvider = {
        id: 'provider-id',
        provider: 'GOOGLE',
        providerId: googleId,
        userId: user.id,
        accessToken: null,
        refreshToken: null,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      const device = {
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
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        ...user,
        providers: [authProvider],
      });
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        ...user,
        providers: [authProvider],
      });
      (prisma.user.findUnique as any).mockResolvedValueOnce({ password: null });
      (prisma.authProvider.update as any).mockResolvedValue(authProvider);
      (prisma.user.update as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);
      (prisma.userDevice.update as any).mockResolvedValue(device);
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('email', email);
    });

    it('should return 500 with invalid Google ID token', async () => {
      // Arrange
      const idToken = 'invalid-google-id-token';

      // Mock OAuth2Client to throw error
      mockVerifyIdToken.mockRejectedValue(new Error('Invalid token'));

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      // OAuth verification errors are caught by global error handler and return 500
      expect(response.status).toBe(500);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 500 with expired Google ID token', async () => {
      // Arrange
      const idToken = 'expired-google-id-token';

      // Mock OAuth2Client to throw error for expired token
      mockVerifyIdToken.mockRejectedValue(new Error('Token expired'));

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      // OAuth verification errors are caught by global error handler and return 500
      expect(response.status).toBe(500);
      expect(response.body).toHaveProperty('message');
    });

    it('should create new user with valid Google token', async () => {
      // Arrange
      const email = 'newuser@example.com';
      const googleId = 'google-user-id-456';
      const idToken = 'valid-google-id-token';

      // Mock OAuth2Client to return new user data
      mockVerifyIdToken.mockResolvedValue({
        getPayload: () => ({
          sub: googleId,
          email,
          name: 'New User',
          picture: 'https://example.com/new-picture.jpg',
          email_verified: true,
        }),
      });

      const newUser = createOAuthUserFixture({
        email,
        name: 'New User',
        username: 'newuser_123',
        profileImg: 'https://example.com/new-picture.jpg',
      });

      const authProvider = {
        id: 'provider-id',
        provider: 'GOOGLE',
        providerId: googleId,
        userId: newUser.id,
        accessToken: null,
        refreshToken: null,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      const device = {
        id: 'device-id',
        userId: newUser.id,
        deviceFingerprint: 'test-fingerprint',
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: true,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce(null); // User doesn't exist
      (prisma.user.create as any).mockResolvedValue({
        ...newUser,
        providers: [authProvider],
      });
      (prisma.user.findUnique as any).mockResolvedValueOnce(null); // deviceBlocked check
      (prisma.user.findUnique as any).mockResolvedValueOnce({ password: null });
      (prisma.userDevice.findUnique as any).mockResolvedValue(null); // New device
      (prisma.userDevice.findMany as any).mockResolvedValue([]); // No existing devices
      (prisma.userDevice.create as any).mockResolvedValue(device);
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: newUser.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });
      (prisma.user.update as any).mockResolvedValue(newUser);

      // Mock username generation
      (usernameUtils.generateUniqueUsername as jest.Mock).mockResolvedValue('newuser_123');

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('email', email);
      expect(prisma.user.create).toHaveBeenCalled();
    });

    it('should login existing user with valid Google token', async () => {
      // Arrange
      const email = 'existing@example.com';
      const googleId = 'google-user-id-789';
      const idToken = 'valid-google-id-token';

      // Mock OAuth2Client to return existing user data
      mockVerifyIdToken.mockResolvedValue({
        getPayload: () => ({
          sub: googleId,
          email,
          name: 'Existing User',
          picture: 'https://example.com/existing-picture.jpg',
          email_verified: true,
        }),
      });

      const existingUser = createOAuthUserFixture({
        email,
        name: 'Existing User',
        profileImg: 'https://example.com/existing-picture.jpg',
      });

      const authProvider = {
        id: 'provider-id',
        provider: 'GOOGLE',
        providerId: googleId,
        userId: existingUser.id,
        accessToken: null,
        refreshToken: null,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      const device = {
        id: 'device-id',
        userId: existingUser.id,
        deviceFingerprint: 'test-fingerprint',
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: true,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        ...existingUser,
        providers: [authProvider],
      });
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        ...existingUser,
        providers: [authProvider],
      });
      (prisma.user.findUnique as any).mockResolvedValueOnce({ password: null });
      (prisma.authProvider.update as any).mockResolvedValue(authProvider);
      (prisma.user.update as any).mockResolvedValue(existingUser);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);
      (prisma.userDevice.update as any).mockResolvedValue(device);
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: existingUser.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('email', email);
      expect(prisma.user.create).not.toHaveBeenCalled();
    });

    it('should return 400 with missing ID token', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('ID token is required');
    });

    it('should return 400 when Google email is not verified', async () => {
      // Arrange
      const idToken = 'valid-google-id-token';

      // Mock OAuth2Client to return unverified email
      mockVerifyIdToken.mockResolvedValue({
        getPayload: () => ({
          sub: 'google-user-id-123',
          email: 'unverified@example.com',
          name: 'Unverified User',
          picture: 'https://example.com/picture.jpg',
          email_verified: false, // Email not verified
        }),
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/google/mobile')
        .send({ idToken });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('not verified');
    });
  });

  /**
   * Property-Based Tests
   * 
   * These tests use fast-check to verify universal properties that should hold
   * for all valid inputs, providing stronger correctness guarantees than
   * example-based tests alone.
   */
  describe('Property-Based Tests', () => {
    describe('POST /api/v1/auth/google/mobile - Property Tests', () => {
      /**
       * Property 24: Valid Google token authentication succeeds
       * **Validates: Requirements 7.1**
       * 
       * For any valid Google ID token, the Google auth endpoint should
       * return 200 status with accessToken and refreshToken
       */
      it('should succeed for any valid Google ID token', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            // Generate valid Google ID
            fc.stringMatching(/^google-[a-zA-Z0-9]{10,20}$/),
            // Generate valid name
            fc.stringMatching(/^[a-zA-Z ]{2,50}$/),
            async (email, googleId, name) => {
              // Arrange
              const idToken = 'valid-google-id-token';

              // Mock OAuth2Client to return valid payload
              mockVerifyIdToken.mockResolvedValue({
                getPayload: () => ({
                  sub: googleId,
                  email,
                  name,
                  picture: 'https://example.com/picture.jpg',
                  email_verified: true,
                }),
              });

              const user = createOAuthUserFixture({
                email,
                name,
                profileImg: 'https://example.com/picture.jpg',
              });

              const authProvider = {
                id: 'provider-id',
                provider: 'GOOGLE',
                providerId: googleId,
                userId: user.id,
                accessToken: null,
                refreshToken: null,
                createdAt: new Date(),
                updatedAt: new Date(),
              };

              const device = {
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
              };

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValueOnce({
                ...user,
                providers: [authProvider],
              });
              (prisma.user.findUnique as any).mockResolvedValueOnce({
                ...user,
                providers: [authProvider],
              });
              (prisma.user.findUnique as any).mockResolvedValueOnce({ password: null });
              (prisma.authProvider.update as any).mockResolvedValue(authProvider);
              (prisma.user.update as any).mockResolvedValue(user);
              (prisma.userDevice.findUnique as any).mockResolvedValue(device);
              (prisma.userDevice.update as any).mockResolvedValue(device);
              (prisma.session.create as any).mockResolvedValue({
                id: 'session-id',
                userId: user.id,
                sessionToken: 'test-token',
                refreshToken: 'test-refresh',
              });

              // Act
              const response = await request(app)
                .post('/api/v1/auth/google/mobile')
                .send({ idToken });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed Google auth:', { email, googleId, name });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return tokens
              expect(response.body).toHaveProperty('accessToken');
              expect(response.body).toHaveProperty('refreshToken');

              // Should return user object
              expect(response.body).toHaveProperty('user');
              expect(response.body.user).toHaveProperty('email', email);

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 25: Invalid Google token rejected
       * **Validates: Requirements 7.2**
       * 
       * For any invalid Google ID token, the Google auth endpoint should
       * return 500 status with error message (OAuth verification errors
       * are caught by the global error handler)
       */
      it('should reject any invalid Google ID token', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate random invalid token
            fc.string({ minLength: 10, maxLength: 100 }),
            async (invalidToken) => {
              // Arrange
              // Mock OAuth2Client to throw error
              mockVerifyIdToken.mockRejectedValue(new Error('Invalid token'));

              // Act
              const response = await request(app)
                .post('/api/v1/auth/google/mobile')
                .send({ idToken: invalidToken });

              // Debug: log response if it fails
              if (response.status !== 500) {
                console.log('Failed rejection:', { invalidToken });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // OAuth verification errors are caught by global error handler and return 500
              expect(response.status).toBe(500);

              // Should return error message
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
});
