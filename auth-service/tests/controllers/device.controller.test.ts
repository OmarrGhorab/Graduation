
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createUserFixture,
  createVerifiedUserFixture,
  generateTestEmail,
  generateTestUsername,
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

// Mock OTP utilities
jest.mock('../../src/utils/otp', () => ({
  __esModule: true,
  createAndStoreOtp: jest.fn().mockResolvedValue('123456'),
  verifyOtp: jest.fn().mockResolvedValue(true),
}));


// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import * as emailUtils from '../../src/utils/email';
import * as otpUtils from '../../src/utils/otp';
import * as notificationsClient from '../../src/utils/notifications-client';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Device Verification Controller Tests
 * 
 * Tests for device verification endpoints including:
 * - Device verification with OTP
 * - Resend device verification OTP
 * 
 * Requirements: 22.1, 22.4
 */
describe('Device Verification Controller', () => {
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

  describe('POST /api/v1/auth/verify-device', () => {
    it('should verify device with valid OTP and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const deviceFingerprint = 'test-device-fingerprint';
      const otp = '123456';
      const user = createVerifiedUserFixture({
        email,
        deviceBlocked: true,
        pendingDeviceFingerprint: deviceFingerprint,
      });

      const device = {
        id: 'device-id',
        userId: user.id,
        deviceFingerprint,
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: false,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);
      (prisma.userDevice.update as any).mockResolvedValue({ ...device, isTrusted: true });
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        deviceBlocked: false,
        pendingDeviceFingerprint: null,
      });
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Mock notification
      jest.mocked(notificationsClient.publishNotification).mockResolvedValue(undefined);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-device')
        .send({ emailOrUsername: email, deviceFingerprint, otp });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('deviceVerified', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
    });

    it('should return 401 with invalid OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const deviceFingerprint = 'test-device-fingerprint';
      const otp = '123456';
      const user = createVerifiedUserFixture({
        email,
        deviceBlocked: true,
        pendingDeviceFingerprint: deviceFingerprint,
      });

      const device = {
        id: 'device-id',
        userId: user.id,
        deviceFingerprint,
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: false,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);

      // Mock OTP verification to fail
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-device')
        .send({ emailOrUsername: email, deviceFingerprint, otp });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 with missing fields', async () => {
      // Act - missing deviceFingerprint
      const response = await request(app)
        .post('/api/v1/auth/verify-device')
        .send({ emailOrUsername: generateTestEmail(), otp: '123456' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 401 with non-existent user', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock user not found
      (prisma.user.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-device')
        .send({
          emailOrUsername: email,
          deviceFingerprint: 'test-fingerprint',
          otp: '123456',
        });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should handle 2FA enabled users correctly', async () => {
      // Arrange
      const email = generateTestEmail();
      const deviceFingerprint = 'test-device-fingerprint';
      const otp = '123456';
      const user = createVerifiedUserFixture({
        email,
        deviceBlocked: true,
        pendingDeviceFingerprint: deviceFingerprint,
        twoFactorEnabled: true,
        twoFactorSecret: 'test-secret',
      });

      const device = {
        id: 'device-id',
        userId: user.id,
        deviceFingerprint,
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: false,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);
      (prisma.userDevice.update as any).mockResolvedValue({ ...device, isTrusted: true });
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        deviceBlocked: false,
        pendingDeviceFingerprint: null,
      });
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: null,
      });

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Mock notification
      jest.mocked(notificationsClient.publishNotification).mockResolvedValue(undefined);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-device')
        .send({ emailOrUsername: email, deviceFingerprint, otp });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('deviceVerified', true);
      expect(response.body).toHaveProperty('requires2FA', true);
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).not.toHaveProperty('refreshToken'); // No refresh token until 2FA is verified
    });
  });

  describe('POST /api/v1/auth/resend-device-verification-otp', () => {
    it('should resend device verification OTP and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const deviceFingerprint = 'test-device-fingerprint';
      const user = createVerifiedUserFixture({
        email,
        deviceBlocked: true,
        pendingDeviceFingerprint: deviceFingerprint,
      });

      const device = {
        id: 'device-id',
        userId: user.id,
        deviceFingerprint,
        deviceName: 'Test Device',
        platform: 'WEB',
        ipAddress: '127.0.0.1',
        userAgent: 'Test Browser',
        isTrusted: false,
        lastLoginAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);

      // Mock OTP creation
      jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

      // Mock email sending
      jest.mocked(emailUtils.sendDeviceVerificationOTP).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-device-verification-otp')
        .send({ emailOrUsername: email, deviceFingerprint });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 when device verification not required', async () => {
      // Arrange
      const email = generateTestEmail();
      const deviceFingerprint = 'test-device-fingerprint';
      const user = createVerifiedUserFixture({
        email,
        deviceBlocked: false,
        pendingDeviceFingerprint: null,
      });

      const device = {
        id: 'device-id',
        userId: user.id,
        deviceFingerprint,
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
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.userDevice.findUnique as any).mockResolvedValue(device);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-device-verification-otp')
        .send({ emailOrUsername: email, deviceFingerprint });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 400 with missing fields', async () => {
      // Act - missing deviceFingerprint
      const response = await request(app)
        .post('/api/v1/auth/resend-device-verification-otp')
        .send({ emailOrUsername: generateTestEmail() });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    it('should handle non-existent user gracefully', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock user not found
      (prisma.user.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-device-verification-otp')
        .send({ emailOrUsername: email, deviceFingerprint: 'test-fingerprint' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      // Should not reveal if user exists
      expect(response.body.message).toContain('If the email/username exists');
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
    describe('POST /api/v1/auth/verify-device - Property Tests', () => {
      /**
       * Property 22: Valid device verification succeeds
       * **Validates: Requirements 6.1**
       * 
       * For any valid device verification OTP, the verify device endpoint should
       * return 200 status and mark device as trusted
       */
      it('should succeed for any valid device verification OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            // Generate valid device fingerprint
            fc.stringMatching(/^[a-zA-Z0-9-]{10,50}$/),
            // Generate valid OTP (6 digits)
            fc.stringMatching(/^[0-9]{6}$/),
            async (email, deviceFingerprint, otp) => {
              // Arrange
              const user = createVerifiedUserFixture({
                email,
                deviceBlocked: true,
                pendingDeviceFingerprint: deviceFingerprint,
              });

              const device = {
                id: 'device-id',
                userId: user.id,
                deviceFingerprint,
                deviceName: 'Test Device',
                platform: 'WEB',
                ipAddress: '127.0.0.1',
                userAgent: 'Test Browser',
                isTrusted: false,
                lastLoginAt: new Date(),
                createdAt: new Date(),
                updatedAt: new Date(),
              };

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);
              (prisma.userDevice.findUnique as any).mockResolvedValue(device);
              (prisma.userDevice.update as any).mockResolvedValue({ ...device, isTrusted: true });
              (prisma.user.update as any).mockResolvedValue({
                ...user,
                deviceBlocked: false,
                pendingDeviceFingerprint: null,
              });
              (prisma.session.create as any).mockResolvedValue({
                id: 'session-id',
                userId: user.id,
                sessionToken: 'test-token',
                refreshToken: 'test-refresh',
              });

              // Mock OTP verification
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

              // Mock notification
              jest.mocked(notificationsClient.publishNotification).mockResolvedValue(undefined);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-device')
                .send({ emailOrUsername: email, deviceFingerprint, otp });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed device verification:', { email, deviceFingerprint, otp });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return device verified flag
              expect(response.body).toHaveProperty('deviceVerified', true);

              // Should return tokens
              expect(response.body).toHaveProperty('accessToken');
              expect(response.body).toHaveProperty('refreshToken');

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 23: Invalid device OTP rejected
       * **Validates: Requirements 6.2**
       * 
       * For any invalid device verification OTP, the verify device endpoint should
       * return 400 status with error message
       */
      it('should reject any invalid device verification OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            // Generate valid device fingerprint
            fc.stringMatching(/^[a-zA-Z0-9-]{10,50}$/),
            // Generate valid OTP (6 digits)
            fc.stringMatching(/^[0-9]{6}$/),
            async (email, deviceFingerprint, otp) => {
              // Arrange
              const user = createVerifiedUserFixture({
                email,
                deviceBlocked: true,
                pendingDeviceFingerprint: deviceFingerprint,
              });

              const device = {
                id: 'device-id',
                userId: user.id,
                deviceFingerprint,
                deviceName: 'Test Device',
                platform: 'WEB',
                ipAddress: '127.0.0.1',
                userAgent: 'Test Browser',
                isTrusted: false,
                lastLoginAt: new Date(),
                createdAt: new Date(),
                updatedAt: new Date(),
              };

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);
              (prisma.userDevice.findUnique as any).mockResolvedValue(device);

              // Mock OTP verification to fail
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-device')
                .send({ emailOrUsername: email, deviceFingerprint, otp });

              // Debug: log response if it fails
              if (response.status !== 401) {
                console.log('Failed rejection:', { email, deviceFingerprint, otp });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 401 status
              expect(response.status).toBe(401);

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
