import { User, Session, UserRole, Gender, Platform } from '@prisma/client';
import bcrypt from 'bcrypt';
import jwt from 'jsonwebtoken';
import crypto from 'crypto';

/**
 * Test fixtures for auth-service controller tests
 * 
 * This module provides factory functions for creating test data including:
 * - User fixtures (verified, unverified, 2FA-enabled)
 * - Token fixtures (valid, expired, invalid)
 * - Session fixtures
 * 
 * All fixtures return realistic test data that matches the production schema.
 */

// Environment variables for token generation
const ACCESS_TOKEN_SECRET = process.env.JWT_ACCESS_SECRET || 'dev-access-secret';
const REFRESH_TOKEN_SECRET = process.env.REFRESH_TOKEN_SECRET || 'dev-refresh-secret';
const ACCESS_TOKEN_TTL_SEC = parseInt(process.env.ACCESS_TOKEN_TTL_SEC || '900', 10); // 15 minutes
const REFRESH_TOKEN_TTL_SEC = parseInt(process.env.REFRESH_TOKEN_TTL_SEC || '2592000', 10); // 30 days

/**
 * Default test password (plain text)
 * Use this for login tests
 */
export const TEST_PASSWORD = 'Test123!@#';

/**
 * Default test password hash
 * Pre-computed hash of TEST_PASSWORD for performance
 */
export const TEST_PASSWORD_HASH = bcrypt.hashSync(TEST_PASSWORD, 10);

/**
 * Generate a unique test email
 * 
 * @param prefix - Optional prefix for the email (default: 'test')
 * @returns Unique email address
 */
export function generateTestEmail(prefix: string = 'test'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `${prefix}-${timestamp}-${random}@example.com`;
}

/**
 * Generate a unique username
 * 
 * @param prefix - Optional prefix for the username (default: 'user')
 * @returns Unique username
 */
export function generateTestUsername(prefix: string = 'user'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `${prefix}_${timestamp}_${random}`;
}

/**
 * Create a basic user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns User object
 */
export function createUserFixture(overrides: Partial<User> = {}): User {
  const now = new Date();

  return {
    id: crypto.randomUUID(),
    name: 'Test User',
    username: generateTestUsername(),
    email: generateTestEmail(),
    password: TEST_PASSWORD_HASH,
    dateOfBirth: new Date('2000-01-01'),
    gender: Gender.PREFER_NOT_TO_SAY,
    role: UserRole.STUDENT,
    profileImg: null,
    bio: null,
    country: null,
    onboardingCompleted: false,
    goals: [],
    newsletterEnabled: false,
    verified: false,
    twoFactorEnabled: false,
    twoFactorSecret: null,
    twoFactorBackupCodes: [],
    deviceBlocked: false,
    pendingDeviceFingerprint: null,
    isActive: true,
    deletedAt: null,
    lastLoginAt: null,
    lastUsernameChange: null,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  };
}

/**
 * Create a verified user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns Verified user object
 */
export function createVerifiedUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    onboardingCompleted: true,
    ...overrides,
  });
}

/**
 * Create an unverified user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns Unverified user object
 */
export function createUnverifiedUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: false,
    onboardingCompleted: false,
    ...overrides,
  });
}

/**
 * Create a user with 2FA enabled
 * 
 * @param overrides - Optional fields to override defaults
 * @returns User with 2FA enabled
 */
export function createUser2FAFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    onboardingCompleted: true,
    twoFactorEnabled: true,
    twoFactorSecret: 'JBSWY3DPEHPK3PXP', // Base32 encoded secret for testing
    twoFactorBackupCodes: [
      'BACKUP-CODE-1',
      'BACKUP-CODE-2',
      'BACKUP-CODE-3',
    ],
    ...overrides,
  });
}

/**
 * Create a deactivated user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns Deactivated user object
 */
export function createDeactivatedUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    isActive: false,
    ...overrides,
  });
}

/**
 * Create a deleted user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns Deleted user object
 */
export function createDeletedUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    isActive: false,
    deletedAt: new Date(),
    ...overrides,
  });
}

/**
 * Create an OAuth user fixture (no password)
 * 
 * @param overrides - Optional fields to override defaults
 * @returns OAuth user object
 */
export function createOAuthUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    onboardingCompleted: true,
    password: null, // OAuth users don't have passwords
    ...overrides,
  });
}

/**
 * Create a parent user fixture
 * 
 * @param overrides - Optional fields to override defaults
 * @returns Parent user object
 */
export function createParentUserFixture(overrides: Partial<User> = {}): User {
  return createUserFixture({
    verified: true,
    onboardingCompleted: true,
    role: UserRole.PARENT,
    ...overrides,
  });
}

/**
 * Token fixture options
 */
export interface TokenFixtureOptions {
  userId?: string;
  role?: string;
  expiresIn?: number | string;
  jti?: string;
}

/**
 * Create a valid access token
 * 
 * @param options - Token options
 * @returns Valid JWT access token
 */
export function createValidAccessToken(options: TokenFixtureOptions = {}): string {
  const {
    userId = crypto.randomUUID(),
    role = 'STUDENT',
    expiresIn = ACCESS_TOKEN_TTL_SEC,
    jti = crypto.randomUUID(),
  } = options;

  const payload = {
    sub: userId,
    jti,
    role,
    type: 'access',
  };

  return jwt.sign(payload, ACCESS_TOKEN_SECRET, {
    expiresIn: expiresIn as jwt.SignOptions['expiresIn'],
    algorithm: 'HS256',
  });
}

/**
 * Create an expired access token
 * 
 * @param options - Token options
 * @returns Expired JWT access token
 */
export function createExpiredAccessToken(options: TokenFixtureOptions = {}): string {
  const {
    userId = crypto.randomUUID(),
    role = 'STUDENT',
    jti = crypto.randomUUID(),
  } = options;

  const payload = {
    sub: userId,
    jti,
    role,
    type: 'access',
  };

  // Create token that expired 1 hour ago
  return jwt.sign(payload, ACCESS_TOKEN_SECRET, {
    expiresIn: -3600 as jwt.SignOptions['expiresIn'],
    algorithm: 'HS256',
  });
}

/**
 * Create an invalid access token
 * 
 * @returns Invalid JWT token (signed with wrong secret)
 */
export function createInvalidAccessToken(): string {
  const payload = {
    sub: crypto.randomUUID(),
    jti: crypto.randomUUID(),
    role: 'STUDENT',
    type: 'access',
  };

  // Sign with wrong secret
  return jwt.sign(payload, 'wrong-secret', {
    expiresIn: ACCESS_TOKEN_TTL_SEC,
    algorithm: 'HS256',
  });
}

/**
 * Create a malformed access token
 * 
 * @returns Malformed token string
 */
export function createMalformedAccessToken(): string {
  return 'not.a.valid.jwt.token';
}

/**
 * Create a valid refresh token
 * 
 * @param options - Token options
 * @returns Valid JWT refresh token
 */
export function createValidRefreshToken(options: TokenFixtureOptions = {}): string {
  const {
    userId = crypto.randomUUID(),
    expiresIn = REFRESH_TOKEN_TTL_SEC,
    jti = crypto.randomUUID(),
  } = options;

  const payload = {
    sub: userId,
    jti,
    type: 'refresh',
  };

  return jwt.sign(payload, REFRESH_TOKEN_SECRET, {
    expiresIn: expiresIn as jwt.SignOptions['expiresIn'],
    algorithm: 'HS256',
  });
}

/**
 * Create an expired refresh token
 * 
 * @param options - Token options
 * @returns Expired JWT refresh token
 */
export function createExpiredRefreshToken(options: TokenFixtureOptions = {}): string {
  const {
    userId = crypto.randomUUID(),
    jti = crypto.randomUUID(),
  } = options;

  const payload = {
    sub: userId,
    jti,
    type: 'refresh',
  };

  // Create token that expired 1 day ago
  return jwt.sign(payload, REFRESH_TOKEN_SECRET, {
    expiresIn: -86400 as jwt.SignOptions['expiresIn'],
    algorithm: 'HS256',
  });
}

/**
 * Create an invalid refresh token
 * 
 * @returns Invalid JWT refresh token (signed with wrong secret)
 */
export function createInvalidRefreshToken(): string {
  const payload = {
    sub: crypto.randomUUID(),
    jti: crypto.randomUUID(),
    type: 'refresh',
  };

  // Sign with wrong secret
  return jwt.sign(payload, 'wrong-secret', {
    expiresIn: REFRESH_TOKEN_TTL_SEC,
    algorithm: 'HS256',
  });
}

/**
 * Session fixture options
 */
export interface SessionFixtureOptions extends Partial<Session> {
  userId?: string;
  expiresInSeconds?: number;
}

/**
 * Create a session fixture
 * 
 * @param options - Session options
 * @returns Session object
 */
export function createSessionFixture(options: SessionFixtureOptions = {}): Session {
  const now = new Date();
  const userId = options.userId || crypto.randomUUID();
  const expiresInSeconds = options.expiresInSeconds ?? ACCESS_TOKEN_TTL_SEC;

  const expiresAt = new Date(now.getTime() + expiresInSeconds * 1000);
  const refreshExpiresAt = new Date(now.getTime() + REFRESH_TOKEN_TTL_SEC * 1000);

  return {
    id: crypto.randomUUID(),
    userId,
    deviceId: null,
    sessionToken: crypto.randomUUID(),
    refreshToken: crypto.randomUUID(),
    ipAddress: '127.0.0.1',
    userAgent: 'Mozilla/5.0 (Test Browser)',
    location: 'Test City, Test Country',
    lastLatitude: null,
    lastLongitude: null,
    lastLocationAccuracy: null,
    lastLocationAddress: null,
    lastLocationTimestamp: null,
    isActive: true,
    isRevoked: false,
    expiresAt,
    refreshExpiresAt,
    lastActivityAt: now,
    createdAt: now,
    revokedAt: null,
    ...options,
  };
}

/**
 * Create an active session fixture
 * 
 * @param options - Session options
 * @returns Active session object
 */
export function createActiveSessionFixture(options: SessionFixtureOptions = {}): Session {
  return createSessionFixture({
    isActive: true,
    isRevoked: false,
    expiresInSeconds: ACCESS_TOKEN_TTL_SEC,
    ...options,
  });
}

/**
 * Create an expired session fixture
 * 
 * @param options - Session options
 * @returns Expired session object
 */
export function createExpiredSessionFixture(options: SessionFixtureOptions = {}): Session {
  const now = new Date();
  const expiredAt = new Date(now.getTime() - 3600 * 1000); // Expired 1 hour ago

  return createSessionFixture({
    isActive: true,
    isRevoked: false,
    expiresAt: expiredAt,
    ...options,
  });
}

/**
 * Create a revoked session fixture
 * 
 * @param options - Session options
 * @returns Revoked session object
 */
export function createRevokedSessionFixture(options: SessionFixtureOptions = {}): Session {
  const now = new Date();

  return createSessionFixture({
    isActive: false,
    isRevoked: true,
    revokedAt: now,
    ...options,
  });
}

/**
 * Create a session with GPS location
 * 
 * @param options - Session options
 * @returns Session with GPS location data
 */
export function createSessionWithLocationFixture(options: SessionFixtureOptions = {}): Session {
  const now = new Date();

  return createSessionFixture({
    lastLatitude: 40.7128,
    lastLongitude: -74.0060,
    lastLocationAccuracy: 10.5,
    lastLocationAddress: '123 Test St, New York, NY 10001',
    lastLocationTimestamp: now,
    ...options,
  });
}

/**
 * Create multiple session fixtures for a user
 * 
 * @param userId - User ID
 * @param count - Number of sessions to create
 * @returns Array of session objects
 */
export function createMultipleSessionFixtures(userId: string, count: number = 3): Session[] {
  return Array.from({ length: count }, (_, index) => {
    const now = new Date();
    const createdAt = new Date(now.getTime() - (count - index) * 86400 * 1000); // Stagger by days

    return createSessionFixture({
      userId,
      createdAt,
      ipAddress: `192.168.1.${index + 1}`,
      userAgent: `Test Browser ${index + 1}`,
      location: `Test City ${index + 1}`,
    });
  });
}

/**
 * Decode a JWT token without verification (for testing)
 * 
 * @param token - JWT token
 * @returns Decoded payload
 */
export function decodeToken(token: string): any {
  return jwt.decode(token);
}

/**
 * Extract user ID from a JWT token
 * 
 * @param token - JWT token
 * @returns User ID from token payload
 */
export function extractUserIdFromToken(token: string): string {
  const decoded = jwt.decode(token) as any;
  return decoded?.sub || '';
}

/**
 * Extract JTI from a JWT token
 * 
 * @param token - JWT token
 * @returns JTI from token payload
 */
export function extractJtiFromToken(token: string): string {
  const decoded = jwt.decode(token) as any;
  return decoded?.jti || '';
}
