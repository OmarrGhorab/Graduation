import request from 'supertest';
import { Express } from 'express';
import { mockPrisma, mockRedis } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createParentUserFixture,
  createValidAccessToken,
} from '../helpers/fixtures';
import crypto from 'crypto';

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
  updateNotifications: jest.fn().mockResolvedValue(undefined),
}));

// Mock user language utility
jest.mock('../../src/utils/userLanguage', () => ({
  __esModule: true,
  getUserLanguage: jest.fn().mockResolvedValue('en'),
}));

// Mock parent link helper
jest.mock('../../src/utils/parent-link', () => ({
  __esModule: true,
  sendParentLinkRequestHelper: jest.fn().mockResolvedValue({
    id: 'link-request-id',
    parentId: 'parent-id',
    childId: 'child-id',
    verificationCode: '123456',
    status: 'PENDING',
  }),
  sendUnlinkRequestHelper: jest.fn().mockResolvedValue({
    id: 'unlink-request-id',
  }),
  sendMultipleParentLinkRequests: jest.fn().mockResolvedValue([]),
}));

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';

/**
 * Parent Link Controller Tests
 * 
 * Tests for parent link endpoints including:
 * - Create parent link
 * - Verify parent link
 * - Accept parent link
 * - Reject parent link
 * - Verify expired parent link
 * 
 * Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 22.1, 22.4
 * Properties: 36, 37
 */
describe('Parent Link Controller', () => {
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

  describe('POST /api/v1/parent-link/request', () => {
    /**
     * Test: Create parent link with valid data
     * Requirement: 14.1
     */
    it('should create parent link and return 201', async () => {
      // Arrange
      const child = createVerifiedUserFixture();
      const parent = createParentUserFixture();
      const accessToken = createValidAccessToken({ userId: child.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockImplementation(async ({ where }: any) => {
        if (where.id === parent.id) return { ...parent, role: 'PARENT' };
        if (where.id === child.id) return child;
        return null;
      });

      (prisma.parentChildLink.findUnique as any).mockResolvedValue(null); // Not already linked
      (prisma.parentLinkRequest.findUnique as any).mockResolvedValue(null); // No existing request
      (prisma.parentLinkRequest.upsert as any).mockResolvedValue({
        id: 'link-request-id',
        parentId: parent.id,
        childId: child.id,
        status: 'PENDING',
        createdAt: new Date(),
      });
      (prisma.parentLinkRequest.findUnique as any).mockResolvedValue({
        id: 'link-request-id',
        parentId: parent.id,
        childId: child.id,
        status: 'PENDING',
        createdAt: new Date(),
        parent: {
          id: parent.id,
          username: parent.username,
          name: parent.name,
          profileImg: parent.profileImg,
        }
      });

      // Act
      const response = await request(app)
        .post('/api/v1/parent-link/request')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ parentId: parent.id });

      // Assert
      expect(response.status).toBe(201);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('request');
    });
  });

  describe('POST /api/v1/parent-link/respond', () => {
    /**
     * Test: Accept parent link
     * Requirement: 14.3
     */
    it('should accept parent link and return 200', async () => {
      // Arrange
      const child = createVerifiedUserFixture();
      const parent = createParentUserFixture();
      const accessToken = createValidAccessToken({ userId: parent.id });
      const requestId = 'link-request-id';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(parent);
      (prisma.parentLinkRequest.findUnique as any).mockResolvedValue({
        id: requestId,
        parentId: parent.id,
        childId: child.id,
        status: 'PENDING',
        createdAt: new Date(),
        child: { id: child.id, name: child.name, username: child.username }
      });

      (prisma.parentChildLink.count as any).mockResolvedValue(0);
      (prisma.$transaction as any).mockImplementation(async (callback: any) => {
        const tx = {
          parentChildLink: {
            create: jest.fn().mockResolvedValue({
              id: 'parent-child-link-id',
              parentId: parent.id,
              childId: child.id,
            }),
            count: jest.fn().mockResolvedValue(0),
          },
          parentLinkRequest: {
            update: jest.fn().mockResolvedValue({
              id: requestId,
              status: 'ACCEPTED',
              respondedAt: new Date(),
            }),
          },
        };
        return callback(tx);
      });

      // Act
      const response = await request(app)
        .post('/api/v1/parent-link/respond')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ requestId, action: 'accept' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.request.status).toBe('ACCEPTED');
    });

    /**
     * Test: Reject parent link
     * Requirement: 14.4
     */
    it('should reject parent link and return 200', async () => {
      // Arrange
      const child = createVerifiedUserFixture();
      const parent = createParentUserFixture();
      const accessToken = createValidAccessToken({ userId: parent.id });
      const requestId = 'link-request-id';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(parent);
      (prisma.parentLinkRequest.findUnique as any).mockResolvedValue({
        id: requestId,
        parentId: parent.id,
        childId: child.id,
        status: 'PENDING',
        createdAt: new Date(),
        child: { id: child.id, name: child.name, username: child.username }
      });

      (prisma.$transaction as any).mockImplementation(async (callback: any) => {
        const tx = {
          parentLinkRequest: {
            update: jest.fn().mockResolvedValue({
              id: requestId,
              status: 'DECLINED',
              respondedAt: new Date(),
            }),
          },
        };
        return callback(tx);
      });

      // Act
      const response = await request(app)
        .post('/api/v1/parent-link/respond')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ requestId, action: 'decline' });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.request.status).toBe('DECLINED');
    });
  });

  describe('GET /api/v1/parent-link/verify-link', () => {
    /**
     * Test: Verify parent-child link exists (internal endpoint)
     */
    it('should verify parent-child link exists', async () => {
      // Arrange
      const child = createVerifiedUserFixture();
      const parent = createParentUserFixture();

      // Mock database operations
      (prisma.parentChildLink.findFirst as any).mockResolvedValue({
        id: 'parent-child-link-id',
        parentId: parent.id,
        childId: child.id,
      });

      // Act
      const response = await request(app)
        .get('/api/v1/parent-link/verify-link')
        .set('x-internal-service-secret', 'test-internal-secret')
        .query({ parentId: parent.id, childId: child.id });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('linked', true);
    });

    /**
     * Test: Verify parent-child link does not exist
     */
    it('should return false when link does not exist', async () => {
      // Arrange
      const child = createVerifiedUserFixture();
      const parent = createParentUserFixture();

      // Mock database operations
      (prisma.parentChildLink.findFirst as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .get('/api/v1/parent-link/verify-link')
        .set('x-internal-service-secret', 'test-internal-secret')
        .query({ parentId: parent.id, childId: child.id });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('linked', false);
    });
  });
});
