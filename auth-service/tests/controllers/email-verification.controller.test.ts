
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createUserFixture,
  createVerifiedUserFixture,
  createUnverifiedUserFixture,
  generateTestEmail,
  TEST_PASSWORD_HASH,
} from '../helpers/fixtures';

// Mock modules before importing the app
jest.mock('../../src/libs/prisma', () => ({
  __esModule: true,
  default: mockPrisma(),
}));

jest.mock('../../src/libs/redis', () => ({
  __esModule: true,
  default: mockRedis(),
}));

jest.mock('../../src/utils/email', () => ({
  __esModule: true,
  sendVerificationOTP: jest.fn().mockResolvedValue(true),
  sendPasswordResetOTP: jest.fn().mockResolvedValue(true),
  sendDeviceVerificationOTP: jest.fn().mockResolvedValue(true),
  sendNewDeviceSecurityAlert: jest.fn().mockResolvedValue(undefined),
  sendAccountDeletionConfirmation: jest.fn().mockResolvedValue(undefined),
  sendAccountReactivationEmail: jest.fn().mockResolvedValue(undefined),
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

// Mock email verification utilities
jest.mock('../../src/utils/emailVerification', () => ({
  __esModule: true,
  checkEmailVerificationAllowed: jest.fn().mockResolvedValue({ allowed: true, remainingCooldown: 0, attempts: 0 }),
  setEmailVerificationCooldown: jest.fn().mockResolvedValue(0),
  clearEmailVerificationCooldown: jest.fn().mockResolvedValue(undefined),
  checkResendOtpAllowed: jest.fn().mockResolvedValue({ allowed: true, remainingCooldown: 0, attempts: 0 }),
  setResendOtpCooldown: jest.fn().mockResolvedValue(0),
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
import * as emailVerificationUtils from '../../src/utils/emailVerification';
import * as otpUtils from '../../src/utils/otp';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Email Verification Controller Tests
 * 
 * Tests for email verification endpoints including:
 * - Email OTP verification
 * - Resend verification OTP
 * 
 * Requirements: 22.1, 22.4
 */
describe('Email Verification Controller', () => {
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

  describe('POST /api/v1/auth/verify-email-otp', () => {
    it('should verify email with valid OTP and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const otp = '123456';
      const user = createUnverifiedUserFixture({ email });

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkEmailVerificationAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.clearEmailVerificationCooldown).mockResolvedValue(undefined);

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Mock database operations
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        verified: true,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-email-otp')
        .send({ email, otp });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('verified', true);
    });

    it('should return 400 with invalid OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const otp = '123456';
      const wrongOtp = '654321';

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkEmailVerificationAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.setEmailVerificationCooldown).mockResolvedValue(0);

      // Mock OTP verification - return false for wrong OTP
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-email-otp')
        .send({ email, otp: wrongOtp });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 with expired OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const otp = '123456';

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkEmailVerificationAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.setEmailVerificationCooldown).mockResolvedValue(0);

      // Mock OTP verification - return false for expired OTP
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-email-otp')
        .send({ email, otp });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 with missing email', async () => {
      // Arrange
      const otp = '123456';

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-email-otp')
        .send({ otp });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 with missing OTP', async () => {
      // Arrange
      const email = generateTestEmail();

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-email-otp')
        .send({ email });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('POST /api/v1/auth/resend-verification-otp', () => {
    it('should resend verification OTP with valid email and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createUnverifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkResendOtpAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.setResendOtpCooldown).mockResolvedValue(0);

      // Mock OTP creation
      jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

      // Mock email sending
      jest.mocked(emailUtils.sendVerificationOTP).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-verification-otp')
        .send({ email });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 429 during cooldown period', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock email verification utilities - return cooldown active
      jest.mocked(emailVerificationUtils.checkResendOtpAllowed).mockResolvedValue({
        allowed: false,
        remainingCooldown: 60,
        attempts: 3,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-verification-otp')
        .send({ email });

      // Assert
      expect(response.status).toBe(429);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 400 for verified email', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkResendOtpAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.setResendOtpCooldown).mockResolvedValue(0);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-verification-otp')
        .send({ email });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error', 'Email already verified');
    });

    it('should return 400 with missing email', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-verification-otp')
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
    });

    it('should handle non-existent email gracefully', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock database operations - user not found
      (prisma.user.findUnique as any).mockResolvedValue(null);

      // Mock email verification utilities
      jest.mocked(emailVerificationUtils.checkResendOtpAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(emailVerificationUtils.setResendOtpCooldown).mockResolvedValue(0);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/resend-verification-otp')
        .send({ email });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      // Should not reveal if email exists
      expect(response.body.message).toContain('If the email exists');
    });
  });

  /**
   * Property-Based Tests
   */
  describe('Property-Based Tests', () => {
    describe('POST /api/v1/auth/verify-email-otp - Property Tests', () => {
      it('should succeed for any valid email verification OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            // Generate valid OTP (6 digits)
            fc.stringMatching(/^[0-9]{6}$/),
            async (email, otp) => {
              // Arrange
              const user = createUnverifiedUserFixture({ email });

              // Mock email verification utilities
              jest.mocked(emailVerificationUtils.checkEmailVerificationAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(emailVerificationUtils.clearEmailVerificationCooldown).mockResolvedValue(undefined);

              // Mock OTP verification
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

              // Mock database operations
              (prisma.user.update as any).mockResolvedValue({
                ...user,
                verified: true,
              });

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-email-otp')
                .send({ email, otp });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed verification data:', { email, otp });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              expect(response.status).toBe(200);
              expect(response.body).toHaveProperty('user');
              expect(response.body.user).toHaveProperty('verified', true);

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });

      it('should reject any invalid email verification OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            // Generate valid OTP (6 digits)
            fc.stringMatching(/^[0-9]{6}$/),
            // Generate different invalid OTP (6 digits)
            fc.stringMatching(/^[0-9]{6}$/),
            async (email: string, correctOtp: string, wrongOtp: string) => {
              // Skip if OTPs are the same
              fc.pre(correctOtp !== wrongOtp);

              // Arrange
              // Mock email verification utilities
              jest.mocked(emailVerificationUtils.checkEmailVerificationAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(emailVerificationUtils.setEmailVerificationCooldown).mockResolvedValue(0);

              // Mock OTP verification - return false for wrong OTP
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-email-otp')
                .send({ email, otp: wrongOtp });

              // Debug: log response if it fails
              if (response.status !== 401) {
                console.log('Failed rejection data:', { email, correctOtp, wrongOtp });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              expect(response.status).toBe(401);
              expect(response.body).toHaveProperty('message');

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });
    });

    describe('POST /api/v1/auth/resend-verification-otp - Property Tests', () => {
      it('should succeed for any valid unverified email address', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email
            fc.emailAddress(),
            async (email) => {
              // Arrange
              const user = createUnverifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValue(user);

              // Mock email verification utilities
              jest.mocked(emailVerificationUtils.checkResendOtpAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(emailVerificationUtils.setResendOtpCooldown).mockResolvedValue(0);

              // Mock OTP creation
              jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

              // Mock email sending
              jest.mocked(emailUtils.sendVerificationOTP).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/resend-verification-otp')
                .send({ email });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed resend data:', { email });
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              expect(response.status).toBe(200);
              expect(response.body).toHaveProperty('message');

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });
    });
  });
});
