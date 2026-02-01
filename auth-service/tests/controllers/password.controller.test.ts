
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createUserFixture,
  createVerifiedUserFixture,
  generateTestEmail,
  generateTestUsername,
  TEST_PASSWORD,
} from '../helpers/fixtures';

// Mock modules before importing the app
// Note: jest.mock is hoisted, so we use factory functions
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
  publishNotification: jest.fn(() => Promise.resolve(undefined)),
}));

// Mock user language utility
jest.mock('../../src/utils/userLanguage', () => ({
  __esModule: true,
  getUserLanguageByEmail: jest.fn().mockResolvedValue('en'),
}));

// Mock password reset utilities
jest.mock('../../src/utils/passwordReset', () => ({
  __esModule: true,
  checkForgotPasswordAllowed: jest.fn().mockResolvedValue({ allowed: true, remainingCooldown: 0, attempts: 0 }),
  setForgotPasswordCooldown: jest.fn().mockResolvedValue(0),
  checkResetPasswordAllowed: jest.fn().mockResolvedValue({ allowed: true, remainingCooldown: 0, attempts: 0 }),
  setResetPasswordCooldown: jest.fn().mockResolvedValue(0),
  clearAllPasswordResetCooldowns: jest.fn().mockResolvedValue(undefined),
}));

// Mock OTP utilities
jest.mock('../../src/utils/otp', () => ({
  __esModule: true,
  createAndStoreOtp: jest.fn().mockResolvedValue('123456'),
  verifyOtp: jest.fn().mockResolvedValue(true),
  verifyOtpWithoutConsuming: jest.fn().mockResolvedValue(true),
}));

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import * as emailUtils from '../../src/utils/email';
import * as passwordResetUtils from '../../src/utils/passwordReset';
import * as otpUtils from '../../src/utils/otp';
import * as notificationsClient from '../../src/utils/notifications-client';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Password Controller Tests
 * 
 * Tests for password management endpoints including:
 * - Forgot password (request OTP)
 * - Verify reset OTP
 * - Reset password
 * 
 * Requirements: 22.1, 22.4
 */
describe('Password Controller', () => {
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
   * Additional cleanup if needed
   */
  afterEach(() => {
    // Additional cleanup can be added here if needed
  });

  describe('POST /api/v1/auth/forgot-password', () => {
    it('should send OTP with valid email and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkForgotPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setForgotPasswordCooldown).mockResolvedValue(0);

      // Mock OTP creation
      jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

      // Mock email sending
      jest.mocked(emailUtils.sendPasswordResetOTP).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/forgot-password')
        .send({ emailOrUsername: email });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('OTP has been sent');
    });

    it('should send OTP with valid username and return 200', async () => {
      // Arrange
      const username = generateTestUsername();
      const user = createVerifiedUserFixture({ username });

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkForgotPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setForgotPasswordCooldown).mockResolvedValue(0);

      // Mock OTP creation
      jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

      // Mock email sending
      jest.mocked(emailUtils.sendPasswordResetOTP).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/forgot-password')
        .send({ emailOrUsername: username });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('OTP has been sent');
    });

    it('should return 400 with missing emailOrUsername', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/forgot-password')
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Email or username is required');
    });

    it('should return 400 with non-existent email', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock user not found
      (prisma.user.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/forgot-password')
        .send({ emailOrUsername: email });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('User not found');
    });

    it('should return 429 when rate limit is exceeded', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities to simulate cooldown
      jest.mocked(passwordResetUtils.checkForgotPasswordAllowed).mockResolvedValue({
        allowed: false,
        remainingCooldown: 300, // 5 minutes remaining
        attempts: 3,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/forgot-password')
        .send({ emailOrUsername: email });

      // Assert
      expect(response.status).toBe(429);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Too many password reset requests');
    });
  });

  describe('POST /api/v1/auth/verify-reset-otp', () => {
    it('should verify valid OTP and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const otp = '123456';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtpWithoutConsuming).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: email, otp });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('valid', true);
      expect(response.body.message).toContain('OTP verified');
    });

    it('should return 401 with invalid OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const invalidOtp = '999999';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });

      // Mock OTP verification to fail
      jest.mocked(otpUtils.verifyOtpWithoutConsuming).mockResolvedValue(false);
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: email, otp: invalidOtp });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid or expired OTP');
    });

    it('should return 401 with expired OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const otp = '123456';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });

      // Mock OTP verification to fail (expired)
      jest.mocked(otpUtils.verifyOtpWithoutConsuming).mockResolvedValue(false);
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: email, otp });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid or expired OTP');
    });

    it('should return 400 with missing fields', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: generateTestEmail() }); // Missing otp

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Missing required fields');
    });

    it('should return 400 with non-existent user', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock user not found
      (prisma.user.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: email, otp: '123456' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('User not found');
    });

    it('should return 429 when verification rate limit is exceeded', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities to simulate cooldown
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: false,
        remainingCooldown: 1800, // 30 minutes remaining
        attempts: 3,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/verify-reset-otp')
        .send({ emailOrUsername: email, otp: '123456' });

      // Assert
      expect(response.status).toBe(429);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Too many verification attempts');
    });
  });

  describe('POST /api/v1/auth/reset-password', () => {
    it('should reset password with valid OTP and return 200', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const otp = '123456';
      const newPassword = 'NewPassword123!';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, password: 'hashed-new-password' });

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);
      jest.mocked(passwordResetUtils.clearAllPasswordResetCooldowns).mockResolvedValue(undefined);

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp, newPassword });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Password reset successful');

      // Verify password was updated
      expect(prisma.user.update).toHaveBeenCalledWith(
        expect.objectContaining({
          where: { email },
          data: expect.objectContaining({
            password: expect.any(String),
          }),
        })
      );
    });

    it('should reset password with username and valid OTP', async () => {
      // Arrange
      const username = generateTestUsername();
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ username, email });
      const otp = '123456';
      const newPassword = 'NewPassword123!';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({ ...user, password: 'hashed-new-password' });

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);
      jest.mocked(passwordResetUtils.clearAllPasswordResetCooldowns).mockResolvedValue(undefined);

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: username, otp, newPassword });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Password reset successful');
    });

    it('should return 401 with invalid OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const invalidOtp = '999999';
      const newPassword = 'NewPassword123!';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);

      // Mock OTP verification to fail
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp: invalidOtp, newPassword });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid or expired OTP');
    });

    it('should return 401 with expired OTP', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const otp = '123456';
      const newPassword = 'NewPassword123!';

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });
      jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);

      // Mock OTP verification to fail (expired)
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(false);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp, newPassword });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid or expired OTP');
    });

    it('should return 400 with missing fields', async () => {
      // Act - missing newPassword
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: generateTestEmail(), otp: '123456' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Missing required fields');
    });

    it('should return 400 with non-existent user', async () => {
      // Arrange
      const email = generateTestEmail();

      // Mock user not found
      (prisma.user.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp: '123456', newPassword: 'NewPassword123!' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('User not found');
    });

    it('should return 429 when reset rate limit is exceeded', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities to simulate cooldown
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: false,
        remainingCooldown: 1800, // 30 minutes remaining
        attempts: 3,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp: '123456', newPassword: 'NewPassword123!' });

      // Assert
      expect(response.status).toBe(429);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Too many reset attempts');
    });

    it('should return 400 with weak password (less than 8 characters)', async () => {
      // Arrange
      const email = generateTestEmail();
      const user = createVerifiedUserFixture({ email });
      const otp = '123456';
      const weakPassword = 'weak'; // Less than 8 characters

      // Mock database operations
      (prisma.user.findFirst as any).mockResolvedValue(user);

      // Mock password reset utilities
      jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
        allowed: true,
        remainingCooldown: 0,
        attempts: 0,
      });

      // Mock OTP verification
      jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/reset-password')
        .send({ emailOrUsername: email, otp, newPassword: weakPassword });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Password must be at least 8 characters');
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
    describe('POST /api/v1/auth/forgot-password - Property Tests', () => {
      /**
       * Property 13: Valid password reset request succeeds
       * **Validates: Requirements 4.1**
       * 
       * For any valid email address of existing user, the forgot password endpoint
       * should return 200 status and send OTP
       */
      it('should succeed for any valid email of existing user', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            async (email) => {
              // Arrange
              const user = createVerifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkForgotPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(passwordResetUtils.setForgotPasswordCooldown).mockResolvedValue(0);

              // Mock OTP creation
              jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

              // Mock email sending
              jest.mocked(emailUtils.sendPasswordResetOTP).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/forgot-password')
                .send({ emailOrUsername: email });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed for email:', email);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return success message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('OTP has been sent');

              // Should have called OTP creation
              expect(otpUtils.createAndStoreOtp).toHaveBeenCalled();

              // Should have sent email
              expect(emailUtils.sendPasswordResetOTP).toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 13 (variant): Valid password reset request succeeds with username
       * **Validates: Requirements 4.1**
       * 
       * For any valid username of existing user, the forgot password endpoint
       * should return 200 status and send OTP
       */
      it('should succeed for any valid username of existing user', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid usernames (3-20 chars, alphanumeric and underscore)
            fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9_]{2,19}$/),
            async (username) => {
              // Arrange
              const email = generateTestEmail();
              const user = createVerifiedUserFixture({ username, email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkForgotPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(passwordResetUtils.setForgotPasswordCooldown).mockResolvedValue(0);

              // Mock OTP creation
              jest.mocked(otpUtils.createAndStoreOtp).mockResolvedValue('123456');

              // Mock email sending
              jest.mocked(emailUtils.sendPasswordResetOTP).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/forgot-password')
                .send({ emailOrUsername: username });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed for username:', username);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return success message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('OTP has been sent');

              // Should have called OTP creation
              expect(otpUtils.createAndStoreOtp).toHaveBeenCalled();

              // Should have sent email
              expect(emailUtils.sendPasswordResetOTP).toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 14: Non-existent email returns not found
       * **Validates: Requirements 4.2**
       * 
       * For any non-existent email address, the forgot password endpoint
       * should return 400 status with error message
       */
      it('should return 400 for any non-existent email', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            async (email) => {
              // Arrange
              // Mock user not found
              (prisma.user.findFirst as any).mockResolvedValue(null);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/forgot-password')
                .send({ emailOrUsername: email });

              // Debug: log response if it fails
              if (response.status !== 400) {
                console.log('Failed for email:', email);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 400 status
              expect(response.status).toBe(400);

              // Should return error message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('User not found');

              // Should NOT have called OTP creation
              expect(otpUtils.createAndStoreOtp).not.toHaveBeenCalled();

              // Should NOT have sent email
              expect(emailUtils.sendPasswordResetOTP).not.toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 14 (variant): Non-existent username returns not found
       * **Validates: Requirements 4.2**
       * 
       * For any non-existent username, the forgot password endpoint
       * should return 400 status with error message
       */
      it('should return 400 for any non-existent username', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid usernames (3-20 chars, alphanumeric and underscore)
            fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9_]{2,19}$/),
            async (username) => {
              // Arrange
              // Mock user not found
              (prisma.user.findFirst as any).mockResolvedValue(null);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/forgot-password')
                .send({ emailOrUsername: username });

              // Debug: log response if it fails
              if (response.status !== 400) {
                console.log('Failed for username:', username);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 400 status
              expect(response.status).toBe(400);

              // Should return error message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('User not found');

              // Should NOT have called OTP creation
              expect(otpUtils.createAndStoreOtp).not.toHaveBeenCalled();

              // Should NOT have sent email
              expect(emailUtils.sendPasswordResetOTP).not.toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });
    });

    describe('POST /api/v1/auth/verify-reset-otp - Property Tests', () => {
      /**
       * Property 15: Valid OTP verification succeeds
       * **Validates: Requirements 4.3**
       * 
       * For any valid OTP code, the verify reset OTP endpoint
       * should return 200 status with success message
       */
      it('should succeed for any valid OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            // Generate 6-digit OTP codes
            fc.stringMatching(/^[0-9]{6}$/),
            async (email, otp) => {
              // Arrange
              const user = createVerifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });

              // Mock OTP verification to succeed
              jest.mocked(otpUtils.verifyOtpWithoutConsuming).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-reset-otp')
                .send({ emailOrUsername: email, otp });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed for email:', email, 'otp:', otp);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return success message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('OTP verified');
              expect(response.body).toHaveProperty('valid', true);

              // Should have called OTP verification
              expect(otpUtils.verifyOtpWithoutConsuming).toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 16: Invalid OTP rejected
       * **Validates: Requirements 4.4**
       * 
       * For any invalid OTP code, the verify reset OTP endpoint
       * should return 400 status with error message
       */
      it('should reject any invalid OTP', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            // Generate 6-digit OTP codes
            fc.stringMatching(/^[0-9]{6}$/),
            async (email, otp) => {
              // Arrange
              const user = createVerifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });

              // Mock OTP verification to fail
              jest.mocked(otpUtils.verifyOtpWithoutConsuming).mockResolvedValue(false);
              jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/verify-reset-otp')
                .send({ emailOrUsername: email, otp });

              // Debug: log response if it fails
              if (response.status !== 401) {
                console.log('Failed for email:', email, 'otp:', otp);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 401 status
              expect(response.status).toBe(401);

              // Should return error message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('Invalid or expired OTP');

              // Should have called OTP verification
              expect(otpUtils.verifyOtpWithoutConsuming).toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });
    });

    describe('POST /api/v1/auth/reset-password - Property Tests', () => {
      /**
       * Property 17: Valid password reset succeeds
       * **Validates: Requirements 4.5**
       * 
       * For any valid reset token and new password, the reset password endpoint
       * should return 200 status and update password
       */
      it('should succeed for any valid OTP and strong password', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            // Generate 6-digit OTP codes
            fc.stringMatching(/^[0-9]{6}$/),
            // Generate strong passwords (8-20 chars with variety)
            fc.stringMatching(/^[A-Za-z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]{8,20}$/),
            async (email, otp, newPassword) => {
              // Arrange
              const user = createVerifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);
              (prisma.user.update as any).mockResolvedValue({ ...user, password: 'hashed-new-password' });

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });
              jest.mocked(passwordResetUtils.setResetPasswordCooldown).mockResolvedValue(60);
              jest.mocked(passwordResetUtils.clearAllPasswordResetCooldowns).mockResolvedValue(undefined);

              // Mock OTP verification
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/reset-password')
                .send({ emailOrUsername: email, otp, newPassword });

              // Debug: log response if it fails
              if (response.status !== 200) {
                console.log('Failed for email:', email, 'otp:', otp, 'password length:', newPassword.length);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return success message
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('Password reset successful');

              // Should have called OTP verification
              expect(otpUtils.verifyOtp).toHaveBeenCalled();

              // Should have updated password
              expect(prisma.user.update).toHaveBeenCalled();

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 } // Run 10 times with different random inputs
        );
      });

      /**
       * Property 18: Weak password rejected
       * **Validates: Requirements 4.7**
       * 
       * For any weak password (not meeting strength requirements), the reset password endpoint
       * should return 400 status with validation error
       */
      it('should reject any weak password (less than 8 characters)', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid email addresses
            fc.emailAddress(),
            // Generate 6-digit OTP codes
            fc.stringMatching(/^[0-9]{6}$/),
            // Generate weak passwords (1-7 chars)
            fc.stringMatching(/^[A-Za-z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]{1,7}$/),
            async (email, otp, weakPassword) => {
              // Arrange
              const user = createVerifiedUserFixture({ email });

              // Mock database operations
              (prisma.user.findFirst as any).mockResolvedValue(user);

              // Mock password reset utilities
              jest.mocked(passwordResetUtils.checkResetPasswordAllowed).mockResolvedValue({
                allowed: true,
                remainingCooldown: 0,
                attempts: 0,
              });

              // Mock OTP verification
              jest.mocked(otpUtils.verifyOtp).mockResolvedValue(true);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/reset-password')
                .send({ emailOrUsername: email, otp, newPassword: weakPassword });

              // Debug: log response if it fails
              if (response.status !== 400) {
                console.log('Failed for email:', email, 'password length:', weakPassword.length);
                console.log('Response status:', response.status);
                console.log('Response body:', response.body);
              }

              // Assert
              // Should return 400 status
              expect(response.status).toBe(400);

              // Should return validation error
              expect(response.body).toHaveProperty('message');
              expect(response.body.message).toContain('Password must be at least 8 characters');

              // Should NOT have called OTP verification (validation happens first)
              expect(otpUtils.verifyOtp).not.toHaveBeenCalled();

              // Should NOT have updated password
              expect(prisma.user.update).not.toHaveBeenCalled();

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
