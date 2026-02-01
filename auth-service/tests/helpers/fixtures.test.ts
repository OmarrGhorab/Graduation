
import {
  TEST_PASSWORD,
  TEST_PASSWORD_HASH,
  generateTestEmail,
  generateTestUsername,
  createUserFixture,
  createVerifiedUserFixture,
  createUnverifiedUserFixture,
  createUser2FAFixture,
  createDeactivatedUserFixture,
  createDeletedUserFixture,
  createOAuthUserFixture,
  createParentUserFixture,
  createValidAccessToken,
  createExpiredAccessToken,
  createInvalidAccessToken,
  createMalformedAccessToken,
  createValidRefreshToken,
  createExpiredRefreshToken,
  createInvalidRefreshToken,
  createSessionFixture,
  createActiveSessionFixture,
  createExpiredSessionFixture,
  createRevokedSessionFixture,
  createSessionWithLocationFixture,
  createMultipleSessionFixtures,
  decodeToken,
  extractUserIdFromToken,
  extractJtiFromToken,
} from './fixtures';
import bcrypt from 'bcrypt';
import jwt from 'jsonwebtoken';
import { UserRole, Gender } from '@prisma/client';

describe('Test Fixtures', () => {
  describe('Password Fixtures', () => {
    it('should provide a test password', () => {
      expect(TEST_PASSWORD).toBe('Test123!@#');
    });

    it('should provide a valid password hash', async () => {
      const isValid = await bcrypt.compare(TEST_PASSWORD, TEST_PASSWORD_HASH);
      expect(isValid).toBe(true);
    });
  });

  describe('Email and Username Generators', () => {
    it('should generate unique test emails', () => {
      const email1 = generateTestEmail();
      const email2 = generateTestEmail();

      expect(email1).toMatch(/@example\.com$/);
      expect(email2).toMatch(/@example\.com$/);
      expect(email1).not.toBe(email2);
    });

    it('should generate test emails with custom prefix', () => {
      const email = generateTestEmail('admin');
      expect(email).toMatch(/^admin-/);
      expect(email).toMatch(/@example\.com$/);
    });

    it('should generate unique usernames', () => {
      const username1 = generateTestUsername();
      const username2 = generateTestUsername();

      expect(username1).toMatch(/^user_/);
      expect(username2).toMatch(/^user_/);
      expect(username1).not.toBe(username2);
    });

    it('should generate usernames with custom prefix', () => {
      const username = generateTestUsername('admin');
      expect(username).toMatch(/^admin_/);
    });
  });

  describe('User Fixtures', () => {
    it('should create a basic user fixture', () => {
      const user = createUserFixture();

      expect(user).toHaveProperty('id');
      expect(user).toHaveProperty('email');
      expect(user).toHaveProperty('username');
      expect(user.password).toBe(TEST_PASSWORD_HASH);
      expect(user.verified).toBe(false);
      expect(user.onboardingCompleted).toBe(false);
      expect(user.twoFactorEnabled).toBe(false);
      expect(user.isActive).toBe(true);
      expect(user.role).toBe(UserRole.STUDENT);
    });

    it('should create user with overrides', () => {
      const user = createUserFixture({
        name: 'Custom Name',
        email: 'custom@example.com',
        role: UserRole.TEACHER,
      });

      expect(user.name).toBe('Custom Name');
      expect(user.email).toBe('custom@example.com');
      expect(user.role).toBe(UserRole.TEACHER);
    });

    it('should create a verified user fixture', () => {
      const user = createVerifiedUserFixture();

      expect(user.verified).toBe(true);
      expect(user.onboardingCompleted).toBe(true);
    });

    it('should create an unverified user fixture', () => {
      const user = createUnverifiedUserFixture();

      expect(user.verified).toBe(false);
      expect(user.onboardingCompleted).toBe(false);
    });

    it('should create a user with 2FA enabled', () => {
      const user = createUser2FAFixture();

      expect(user.verified).toBe(true);
      expect(user.twoFactorEnabled).toBe(true);
      expect(user.twoFactorSecret).toBeTruthy();
      expect(user.twoFactorBackupCodes).toHaveLength(3);
    });

    it('should create a deactivated user fixture', () => {
      const user = createDeactivatedUserFixture();

      expect(user.verified).toBe(true);
      expect(user.isActive).toBe(false);
      expect(user.deletedAt).toBeNull();
    });

    it('should create a deleted user fixture', () => {
      const user = createDeletedUserFixture();

      expect(user.isActive).toBe(false);
      expect(user.deletedAt).toBeInstanceOf(Date);
    });

    it('should create an OAuth user fixture', () => {
      const user = createOAuthUserFixture();

      expect(user.verified).toBe(true);
      expect(user.password).toBeNull();
    });

    it('should create a parent user fixture', () => {
      const user = createParentUserFixture();

      expect(user.role).toBe(UserRole.PARENT);
      expect(user.verified).toBe(true);
    });
  });

  describe('Token Fixtures', () => {
    const ACCESS_TOKEN_SECRET = process.env.JWT_ACCESS_SECRET || 'test-jwt-secret-key-for-testing-only';
    const REFRESH_TOKEN_SECRET = process.env.REFRESH_TOKEN_SECRET || 'test-jwt-refresh-secret-key-for-testing-only';

    describe('Access Tokens', () => {
      it('should create a valid access token', () => {
        const token = createValidAccessToken();

        expect(token).toBeTruthy();
        expect(typeof token).toBe('string');

        // Verify token can be decoded
        const decoded = jwt.verify(token, ACCESS_TOKEN_SECRET) as any;
        expect(decoded.type).toBe('access');
        expect(decoded.sub).toBeTruthy();
        expect(decoded.jti).toBeTruthy();
      });

      it('should create access token with custom user ID', () => {
        const userId = 'custom-user-id';
        const token = createValidAccessToken({ userId });

        const decoded = jwt.verify(token, ACCESS_TOKEN_SECRET) as any;
        expect(decoded.sub).toBe(userId);
      });

      it('should create access token with custom role', () => {
        const token = createValidAccessToken({ role: 'ADMIN' });

        const decoded = jwt.verify(token, ACCESS_TOKEN_SECRET) as any;
        expect(decoded.role).toBe('ADMIN');
      });

      it('should create an expired access token', () => {
        const token = createExpiredAccessToken();

        expect(token).toBeTruthy();

        // Verify token is expired
        expect(() => {
          jwt.verify(token, ACCESS_TOKEN_SECRET);
        }).toThrow('jwt expired');
      });

      it('should create an invalid access token', () => {
        const token = createInvalidAccessToken();

        expect(token).toBeTruthy();

        // Verify token cannot be verified with correct secret
        expect(() => {
          jwt.verify(token, ACCESS_TOKEN_SECRET);
        }).toThrow();
      });

      it('should create a malformed access token', () => {
        const token = createMalformedAccessToken();

        expect(token).toBe('not.a.valid.jwt.token');

        // Verify token cannot be verified
        expect(() => {
          jwt.verify(token, ACCESS_TOKEN_SECRET);
        }).toThrow();
      });
    });

    describe('Refresh Tokens', () => {
      it('should create a valid refresh token', () => {
        const token = createValidRefreshToken();

        expect(token).toBeTruthy();
        expect(typeof token).toBe('string');

        // Verify token can be decoded
        const decoded = jwt.verify(token, REFRESH_TOKEN_SECRET) as any;
        expect(decoded.type).toBe('refresh');
        expect(decoded.sub).toBeTruthy();
        expect(decoded.jti).toBeTruthy();
      });

      it('should create refresh token with custom user ID', () => {
        const userId = 'custom-user-id';
        const token = createValidRefreshToken({ userId });

        const decoded = jwt.verify(token, REFRESH_TOKEN_SECRET) as any;
        expect(decoded.sub).toBe(userId);
      });

      it('should create an expired refresh token', () => {
        const token = createExpiredRefreshToken();

        expect(token).toBeTruthy();

        // Verify token is expired
        expect(() => {
          jwt.verify(token, REFRESH_TOKEN_SECRET);
        }).toThrow('jwt expired');
      });

      it('should create an invalid refresh token', () => {
        const token = createInvalidRefreshToken();

        expect(token).toBeTruthy();

        // Verify token cannot be verified with correct secret
        expect(() => {
          jwt.verify(token, REFRESH_TOKEN_SECRET);
        }).toThrow();
      });
    });

    describe('Token Utilities', () => {
      it('should decode token without verification', () => {
        const userId = 'test-user-id';
        const token = createValidAccessToken({ userId });

        const decoded = decodeToken(token);
        expect(decoded.sub).toBe(userId);
        expect(decoded.type).toBe('access');
      });

      it('should extract user ID from token', () => {
        const userId = 'test-user-id';
        const token = createValidAccessToken({ userId });

        const extractedUserId = extractUserIdFromToken(token);
        expect(extractedUserId).toBe(userId);
      });

      it('should extract JTI from token', () => {
        const jti = 'test-jti';
        const token = createValidAccessToken({ jti });

        const extractedJti = extractJtiFromToken(token);
        expect(extractedJti).toBe(jti);
      });
    });
  });

  describe('Session Fixtures', () => {
    it('should create a basic session fixture', () => {
      const session = createSessionFixture();

      expect(session).toHaveProperty('id');
      expect(session).toHaveProperty('userId');
      expect(session).toHaveProperty('sessionToken');
      expect(session).toHaveProperty('refreshToken');
      expect(session.isActive).toBe(true);
      expect(session.isRevoked).toBe(false);
      expect(session.expiresAt).toBeInstanceOf(Date);
    });

    it('should create session with custom user ID', () => {
      const userId = 'custom-user-id';
      const session = createSessionFixture({ userId });

      expect(session.userId).toBe(userId);
    });

    it('should create session with custom expiration', () => {
      const expiresInSeconds = 3600; // 1 hour
      const session = createSessionFixture({ expiresInSeconds });

      const now = new Date();
      const expectedExpiry = new Date(now.getTime() + expiresInSeconds * 1000);

      // Allow 1 second tolerance for test execution time
      expect(Math.abs(session.expiresAt.getTime() - expectedExpiry.getTime())).toBeLessThan(1000);
    });

    it('should create an active session fixture', () => {
      const session = createActiveSessionFixture();

      expect(session.isActive).toBe(true);
      expect(session.isRevoked).toBe(false);
      expect(session.expiresAt.getTime()).toBeGreaterThan(Date.now());
    });

    it('should create an expired session fixture', () => {
      const session = createExpiredSessionFixture();

      expect(session.isActive).toBe(true);
      expect(session.isRevoked).toBe(false);
      expect(session.expiresAt.getTime()).toBeLessThan(Date.now());
    });

    it('should create a revoked session fixture', () => {
      const session = createRevokedSessionFixture();

      expect(session.isActive).toBe(false);
      expect(session.isRevoked).toBe(true);
      expect(session.revokedAt).toBeInstanceOf(Date);
    });

    it('should create a session with GPS location', () => {
      const session = createSessionWithLocationFixture();

      expect(session.lastLatitude).toBe(40.7128);
      expect(session.lastLongitude).toBe(-74.0060);
      expect(session.lastLocationAccuracy).toBe(10.5);
      expect(session.lastLocationAddress).toBeTruthy();
      expect(session.lastLocationTimestamp).toBeInstanceOf(Date);
    });

    it('should create multiple sessions for a user', () => {
      const userId = 'test-user-id';
      const count = 5;
      const sessions = createMultipleSessionFixtures(userId, count);

      expect(sessions).toHaveLength(count);

      // All sessions should have the same user ID
      sessions.forEach(session => {
        expect(session.userId).toBe(userId);
      });

      // Sessions should have different IPs and user agents
      const ips = sessions.map(s => s.ipAddress);
      const uniqueIps = new Set(ips);
      expect(uniqueIps.size).toBe(count);
    });

    it('should create sessions with staggered creation dates', () => {
      const userId = 'test-user-id';
      const sessions = createMultipleSessionFixtures(userId, 3);

      // Sessions should be ordered by creation date (oldest first)
      expect(sessions[0].createdAt.getTime()).toBeLessThan(sessions[1].createdAt.getTime());
      expect(sessions[1].createdAt.getTime()).toBeLessThan(sessions[2].createdAt.getTime());
    });
  });
});
