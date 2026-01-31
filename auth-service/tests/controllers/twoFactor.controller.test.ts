
import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createUser2FAFixture,
  createVerifiedUserFixture,
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

// Mock 2FA utilities
jest.mock('../../src/utils/twoFactor', () => ({
  __esModule: true,
  generateSecret: jest.fn(() => ({
    secret: 'JBSWY3DPEHPK3PXP',
    otpauthUrl: 'otpauth://totp/Test?secret=JBSWY3DPEHPK3PXP',
  })),
  generateQRCode: jest.fn(() => Promise.resolve('data:image/png;base64,test-qr-code')),
  verifyToken: jest.fn((_secret: string, token: string) => {
    // Accept any 6-digit token except '000000'
    return token.length === 6 && token !== '000000' && /^\d+$/.test(token);
  }),
  encryptSecret: jest.fn((secret: string) => `encrypted-${secret}`),
  decryptSecret: jest.fn((encrypted: string) => {
    return encrypted.replace('encrypted-', '');
  }),
  generateBackupCodes: jest.fn(() => [
    'BACKUP-CODE-1',
    'BACKUP-CODE-2',
    'BACKUP-CODE-3',
    'BACKUP-CODE-4',
    'BACKUP-CODE-5',
    'BACKUP-CODE-6',
    'BACKUP-CODE-7',
    'BACKUP-CODE-8',
    'BACKUP-CODE-9',
    'BACKUP-CODE-10',
  ]),
  encryptBackupCodes: jest.fn((codes: string[]) => codes),
  verifyBackupCode: jest.fn((codes: string[], code: string) => {
    const index = codes.indexOf(code);
    if (index !== -1) {
      const remaining = [...codes];
      remaining.splice(index, 1);
      return { valid: true, remainingCodes: remaining };
    }
    return { valid: false, remainingCodes: codes };
  }),
}));

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import prisma from '../../src/libs/prisma';

/**
 * Two-Factor Authentication Controller Tests
 * 
 * Tests for 2FA endpoints including:
 * - Enable 2FA (generate QR code)
 * - Verify 2FA setup
 * - Disable 2FA
 * - Verify 2FA login
 * - Get 2FA status
 * - Regenerate backup codes
 * 
 * Requirements: 22.1, 22.4
 */
describe('Two-Factor Authentication Controller', () => {
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

    // Mock session lookup for authentication middleware
    // This allows tests to pass authentication without complex database mocking
    (prisma.session.findFirst as any).mockImplementation(async ({ where }: any) => {
      if (!where || !where.sessionToken) return null;

      // Return a mock session that matches the token's JTI
      return {
        id: 'test-session-id',
        user: {
          id: where.userId || 'test-user-id',
          isActive: true,
          deletedAt: null,
          role: 'STUDENT',
        },
      };
    });

    // Also mock user.findUnique for cases where the controller looks up the user directly
    (prisma.user.findUnique as any).mockImplementation(async ({ where }: any) => {
      if (!where) return null;

      // Return a basic user for any lookup
      return createVerifiedUserFixture({
        id: where.id || 'test-user-id',
      });
    });
  });

  /**
   * Clean up after each test
   */
  afterEach(() => {
    // Additional cleanup can be added here if needed
  });

  describe('POST /api/v1/auth/2fa/enable', () => {
    it('should enable 2FA and return QR code with valid authentication', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        twoFactorSecret: null,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        twoFactorSecret: 'encrypted-secret',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/enable')
        .set('Authorization', `Bearer ${accessToken}`)
        .send();

      // Debug logging
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('qrCode');
      expect(response.body).toHaveProperty('secret');
      expect(response.body).toHaveProperty('manualEntryKey');
    });

    it('should return 400 if 2FA is already enabled', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/enable')
        .set('Authorization', `Bearer ${accessToken}`)
        .send();

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('already enabled');
    });

    it('should return 401 without authentication', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/enable')
        .send();

      // Assert
      expect(response.status).toBe(401);
    });
  });

  describe('POST /api/v1/auth/2fa/verify-setup', () => {
    it('should verify 2FA setup with valid TOTP and return backup codes', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        twoFactorSecret: 'encrypted-secret',
      });
      const accessToken = createValidAccessToken({ userId: user.id });
      const validToken = '123456';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        twoFactorEnabled: true,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-setup')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ token: validToken });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('backupCodes');
      expect(response.body).toHaveProperty('warning');
      expect(Array.isArray(response.body.backupCodes)).toBe(true);
    });

    it('should return 400 with invalid TOTP', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        twoFactorSecret: 'encrypted-secret',
      });
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidToken = '000000';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-setup')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ token: invalidToken });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid token');
    });

    it('should return 400 if no 2FA secret found', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        twoFactorSecret: null,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-setup')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ token: '123456' });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('No 2FA secret found');
    });

    it('should return 400 if token is missing', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-setup')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Token is required');
    });
  });

  describe('POST /api/v1/auth/2fa/disable', () => {
    it('should disable 2FA with valid password', async () => {
      // Arrange
      const user = createUser2FAFixture({
        password: TEST_PASSWORD_HASH,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce(user);
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        twoFactorBackupCodes: user.twoFactorBackupCodes,
      });
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: [],
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/disable')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD, token: '123456' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('disabled successfully');
    });

    it('should return 400 if 2FA is not enabled', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        password: TEST_PASSWORD_HASH,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/disable')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('not enabled');
    });

    it('should return 400 if password is missing', async () => {
      // Arrange
      const user = createUser2FAFixture({
        password: TEST_PASSWORD_HASH,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/disable')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Password is required');
    });
  });

  describe('POST /api/v1/auth/2fa/verify-login', () => {
    it('should verify 2FA login with valid TOTP and return full tokens', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const validToken = '123456';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce(user);
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        id: user.id,
        name: user.name,
        username: user.username,
        email: user.email,
        verified: user.verified,
        onboardingCompleted: user.onboardingCompleted,
        role: user.role,
        profileImg: user.profileImg,
      });
      (prisma.user.update as any).mockResolvedValue(user);
      // Don't override session.findFirst - let the beforeEach mock handle authentication
      (prisma.userDevice.findFirst as any).mockResolvedValue({
        id: 'device-id',
        userId: user.id,
      });
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-login')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ token: validToken });

      // Debug logging
      if (response.status !== 200) {
        console.log('Response status:', response.status);
        console.log('Response body:', response.body);
      }

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('twoFactorEnabled', true);
    });

    it('should return 400 with invalid TOTP', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidToken = '000000';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-login')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ token: invalidToken });

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Invalid token');
    });

    it('should verify 2FA login with backup code', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const backupCode = 'BACKUP-CODE-1';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValueOnce(user);
      (prisma.user.update as any).mockResolvedValueOnce(user);
      (prisma.user.findUnique as any).mockResolvedValueOnce({
        id: user.id,
        name: user.name,
        username: user.username,
        email: user.email,
        verified: user.verified,
        onboardingCompleted: user.onboardingCompleted,
        role: user.role,
        profileImg: user.profileImg,
      });
      (prisma.user.update as any).mockResolvedValue(user);
      // Don't override session.findFirst - let the beforeEach mock handle authentication
      (prisma.userDevice.findFirst as any).mockResolvedValue({
        id: 'device-id',
        userId: user.id,
      });
      (prisma.session.create as any).mockResolvedValue({
        id: 'session-id',
        userId: user.id,
        sessionToken: 'test-token',
        refreshToken: 'test-refresh',
      });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-login')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ backupCode });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('accessToken');
      expect(response.body).toHaveProperty('refreshToken');
    });

    it('should return 400 if neither token nor backup code provided', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-login')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Token or backup code is required');
    });

    it('should return 401 without authentication', async () => {
      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/verify-login')
        .send({ token: '123456' });

      // Assert
      expect(response.status).toBe(401);
    });
  });

  describe('GET /api/v1/auth/2fa/status', () => {
    it('should return 2FA status for authenticated user', async () => {
      // Arrange
      const user = createUser2FAFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .get('/api/v1/auth/2fa/status')
        .set('Authorization', `Bearer ${accessToken}`)
        .send();

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('twoFactorEnabled', true);
      expect(response.body).toHaveProperty('backupCodesCount');
      expect(typeof response.body.backupCodesCount).toBe('number');
    });

    it('should return 401 without authentication', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/auth/2fa/status')
        .send();

      // Assert
      expect(response.status).toBe(401);
    });
  });

  describe('POST /api/v1/auth/2fa/regenerate-backup-codes', () => {
    it('should regenerate backup codes with valid password', async () => {
      // Arrange
      const user = createUser2FAFixture({
        password: TEST_PASSWORD_HASH,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/regenerate-backup-codes')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('backupCodes');
      expect(response.body).toHaveProperty('warning');
      expect(Array.isArray(response.body.backupCodes)).toBe(true);
    });

    it('should return 400 if 2FA is not enabled', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        twoFactorEnabled: false,
        password: TEST_PASSWORD_HASH,
      });
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .post('/api/v1/auth/2fa/regenerate-backup-codes')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ password: TEST_PASSWORD });

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('not enabled');
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
    describe('POST /api/v1/auth/2fa/verify-setup - Property Tests', () => {
      /**
       * Property 26: Valid TOTP setup succeeds
       * **Validates: Requirements 8.2**
       * 
       * For any valid TOTP token during 2FA setup, the verify setup endpoint
       * should return 200 status with backup codes
       */
      it('should succeed for any valid TOTP token', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid 6-digit TOTP token
            fc.integer({ min: 100000, max: 999999 }).map(n => n.toString()),
            async (token) => {
              // Arrange
              const user = createVerifiedUserFixture({
                twoFactorEnabled: false,
                twoFactorSecret: 'encrypted-secret',
              });
              const accessToken = createValidAccessToken({ userId: user.id });

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValue(user);
              (prisma.user.update as any).mockResolvedValue({
                ...user,
                twoFactorEnabled: true,
              });

              // Act
              const response = await request(app)
                .post('/api/v1/auth/2fa/verify-setup')
                .set('Authorization', `Bearer ${accessToken}`)
                .send({ token });

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return backup codes
              expect(response.body).toHaveProperty('backupCodes');
              expect(Array.isArray(response.body.backupCodes)).toBe(true);

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });

      /**
       * Property 27: Invalid TOTP setup rejected
       * **Validates: Requirements 8.3**
       * 
       * For any invalid TOTP token during 2FA setup, the verify setup endpoint
       * should return 400 status with error message
       */
      it('should reject any invalid TOTP token', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate invalid tokens (not 6 digits or all zeros)
            fc.oneof(
              fc.string({ minLength: 1, maxLength: 5 }), // Too short
              fc.string({ minLength: 7, maxLength: 10 }), // Too long
              fc.constant('000000'), // All zeros
            ),
            async (invalidToken) => {
              // Arrange
              const user = createVerifiedUserFixture({
                twoFactorEnabled: false,
                twoFactorSecret: 'encrypted-secret',
              });
              const accessToken = createValidAccessToken({ userId: user.id });

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValue(user);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/2fa/verify-setup')
                .set('Authorization', `Bearer ${accessToken}`)
                .send({ token: invalidToken });

              // Assert
              // Should return 400 status
              expect(response.status).toBe(400);

              // Should return error message
              expect(response.body).toHaveProperty('message');
              expect(typeof response.body.message).toBe('string');

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });
    });

    describe('POST /api/v1/auth/2fa/verify-login - Property Tests', () => {
      /**
       * Property 28: Valid 2FA login succeeds
       * **Validates: Requirements 8.5**
       * 
       * For any valid TOTP token during 2FA login, the verify login endpoint
       * should return 200 status with full access tokens
       */
      it('should succeed for any valid TOTP token during login', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate valid 6-digit TOTP token
            fc.integer({ min: 100000, max: 999999 }).map(n => n.toString()),
            async (token) => {
              // Arrange
              const user = createUser2FAFixture();
              const accessToken = createValidAccessToken({ userId: user.id });

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValueOnce(user);
              (prisma.user.findUnique as any).mockResolvedValueOnce({
                id: user.id,
                name: user.name,
                username: user.username,
                email: user.email,
                verified: user.verified,
                onboardingCompleted: user.onboardingCompleted,
                role: user.role,
                profileImg: user.profileImg,
              });
              (prisma.user.update as any).mockResolvedValue(user);
              // Don't override session.findFirst - let the beforeEach mock handle authentication
              (prisma.userDevice.findFirst as any).mockResolvedValue({
                id: 'device-id',
                userId: user.id,
              });
              (prisma.session.create as any).mockResolvedValue({
                id: 'session-id',
                userId: user.id,
                sessionToken: 'test-token',
                refreshToken: 'test-refresh',
              });

              // Act
              const response = await request(app)
                .post('/api/v1/auth/2fa/verify-login')
                .set('Authorization', `Bearer ${accessToken}`)
                .send({ token });

              // Assert
              // Should return 200 status
              expect(response.status).toBe(200);

              // Should return tokens
              expect(response.body).toHaveProperty('accessToken');
              expect(response.body).toHaveProperty('refreshToken');

              // Should return user object
              expect(response.body).toHaveProperty('user');
              expect(response.body.user).toHaveProperty('twoFactorEnabled', true);

              // Clear mocks for next iteration
              jest.clearAllMocks();
            }
          ),
          { numRuns: 10 }
        );
      });

      /**
       * Property 29: Invalid 2FA login rejected
       * **Validates: Requirements 8.6**
       * 
       * For any invalid TOTP token during 2FA login, the verify login endpoint
       * should return 401 status with error message
       */
      it('should reject any invalid TOTP token during login', async () => {
        const fc = await import('fast-check');

        await fc.assert(
          fc.asyncProperty(
            // Generate invalid tokens
            fc.oneof(
              fc.string({ minLength: 1, maxLength: 5 }), // Too short
              fc.string({ minLength: 7, maxLength: 10 }), // Too long
              fc.constant('000000'), // All zeros
            ),
            async (invalidToken) => {
              // Arrange
              const user = createUser2FAFixture();
              const accessToken = createValidAccessToken({ userId: user.id });

              // Mock database operations
              (prisma.user.findUnique as any).mockResolvedValue(user);

              // Act
              const response = await request(app)
                .post('/api/v1/auth/2fa/verify-login')
                .set('Authorization', `Bearer ${accessToken}`)
                .send({ token: invalidToken });

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
          { numRuns: 10 }
        );
      });
    });
  });
});
