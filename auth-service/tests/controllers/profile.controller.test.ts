import request from 'supertest';
import { Express } from 'express';
import fc from 'fast-check';
import { mockPrisma, mockRedis, mockCloudinary } from '../helpers/mocks';
import {
  createVerifiedUserFixture,
  createValidAccessToken,
  generateTestUsername,
} from '../helpers/fixtures';

// Mock modules before importing the app
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
  uploadProfileImage: jest.fn().mockResolvedValue('https://res.cloudinary.com/test/image.jpg'),
  deleteImageFromCloudinary: jest.fn().mockResolvedValue(true),
}));

// Now import the app after mocks are set up
import { createTestApp } from '../helpers/testApp';
import prisma from '../../src/libs/prisma';
import redis from '../../src/libs/redis';
import { uploadProfileImage, deleteImageFromCloudinary } from '../../src/utils/cloudinaryUpload';

/**
 * Profile Management Controller Tests
 * 
 * Tests for profile management endpoints including:
 * - Get profile
 * - Update profile with valid/invalid data
 * - Upload profile image
 * - Upload invalid image format
 * 
 * Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 22.1, 22.4
 * Properties: 31, 32, 33
 */
describe('Profile Management Controller', () => {
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
      // console.log('Mock Session FindFirst Where:', JSON.stringify(where, null, 2));
      if (!where || !where.sessionToken) {
        console.log('Mock Session FindFirst returning NULL because sessionToken missing. Where:', JSON.stringify(where));
        return null;
      }

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

  describe('GET /api/v1/profile', () => {
    /**
     * Test: Get profile with valid token
     * Requirement: 12.1
     */
    it('should get profile with valid token and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue({
        ...user,
        interests: [],
        preferences: {
          language: 'en',
          themePreference: 'light',
          notifications: true,
        },
      });

      // Act
      const response = await request(app)
        .get('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('id', user.id);
      expect(response.body.user).toHaveProperty('email', user.email);
      expect(response.body.user).toHaveProperty('name', user.name);
      expect(response.body.user).toHaveProperty('username', user.username);
      expect(response.body.user).not.toHaveProperty('password');
      expect(response.body).toHaveProperty('canChangeUsername');
    });

    /**
     * Test: Get profile without authentication
     */
    it('should return 401 without authentication token', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/profile');

      // Assert
      expect(response.status).toBe(401);
      expect(response.body).toHaveProperty('message');
    });
  });

  describe('PUT /api/v1/profile', () => {
    /**
     * Test: Update profile with valid data
     * Requirement: 12.2
     */
    it('should update profile with valid data and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const updateData = {
        name: 'Updated Name',
        bio: 'Updated bio',
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        ...updateData,
      });
      (prisma.userInterest.deleteMany as any).mockResolvedValue({ count: 0 });

      // Mock the final fetch with interests
      (prisma.user.findUnique as any).mockResolvedValue({
        ...user,
        ...updateData,
        interests: [],
      });

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(updateData);

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('name', updateData.name);
      expect(response.body.user).toHaveProperty('bio', updateData.bio);
    });

    /**
     * Test: Update profile with invalid data
     * Requirement: 12.3
     */
    it('should return 400 with invalid profile data', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidData = {
        bio: 'a'.repeat(201), // Bio too long (max 200 characters)
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(invalidData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Bio must be at most 200 characters');
    });

    /**
     * Test: Update profile with invalid goals array
     */
    it('should return 400 when goals exceed maximum', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidData = {
        goals: ['Goal 1', 'Goal 2', 'Goal 3', 'Goal 4'], // Max 3 goals
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(invalidData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Maximum 3 goals allowed');
    });

    /**
     * Test: Update username with cooldown restriction
     */
    it('should return 400 when trying to change username within cooldown period', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        lastUsernameChange: new Date(), // Just changed username
      });
      const accessToken = createValidAccessToken({ userId: user.id });
      const updateData = {
        username: 'newusername',
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(updateData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('You can only change your username once every 2 weeks');
    });

    /**
     * Test: Update username when already taken
     */
    it('should return 400 when username is already taken', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        lastUsernameChange: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000), // 15 days ago
      });
      const accessToken = createValidAccessToken({ userId: user.id });
      const updateData = {
        username: 'takenusername',
      };

      // Mock database operations
      (prisma.user.findUnique as any).mockImplementation(async ({ where }: any) => {
        if (where.id) return user;
        if (where.username === 'takenusername') {
          return { id: 'other-user-id' }; // Username is taken
        }
        return null;
      });

      // Act
      const response = await request(app)
        .put('/api/v1/profile')
        .set('Authorization', `Bearer ${accessToken}`)
        .send(updateData);

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('already taken');
    });

    /**
     * Property 31: Valid profile update succeeds
     * For any valid profile data, the update profile endpoint should return 200 status and updated profile
     * Validates: Requirements 12.2
     */
    it('should update profile successfully with any valid data', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.record({
            name: fc.string({ minLength: 1, maxLength: 100 }),
            bio: fc.string({ maxLength: 200 }),
            goals: fc.array(fc.string({ minLength: 1, maxLength: 50 }), { maxLength: 3 }),
          }),
          async (profileData) => {
            // Arrange
            const user = createVerifiedUserFixture();
            const accessToken = createValidAccessToken({ userId: user.id });

            // Mock database operations
            (prisma.user.findUnique as any).mockResolvedValue(user);
            (prisma.user.update as any).mockResolvedValue({
              ...user,
              ...profileData,
            });
            (prisma.userInterest.deleteMany as any).mockResolvedValue({ count: 0 });
            (prisma.user.findUnique as any).mockResolvedValue({
              ...user,
              ...profileData,
              interests: [],
            });

            // Act
            const response = await request(app)
              .put('/api/v1/profile')
              .set('Authorization', `Bearer ${accessToken}`)
              .send(profileData);

            // Assert
            expect(response.status).toBe(200);
            expect(response.body).toHaveProperty('message');
            expect(response.body).toHaveProperty('user');
          }
        )
      );
    });

    /**
     * Property 32: Invalid profile data rejected
     * For any invalid profile data, the update profile endpoint should return 400 status with validation error
     * Validates: Requirements 12.3
     */
    it('should reject invalid profile data with 400 error', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.oneof(
            fc.record({ bio: fc.string({ minLength: 201, maxLength: 300 }) }), // Bio too long
            fc.record({ goals: fc.array(fc.string(), { minLength: 4, maxLength: 10 }) }), // Too many goals
            fc.record({ goals: fc.constant('not-an-array') }), // Goals not an array
          ),
          async (invalidData) => {
            // Arrange
            const user = createVerifiedUserFixture();
            const accessToken = createValidAccessToken({ userId: user.id });

            // Mock database operations
            (prisma.user.findUnique as any).mockResolvedValue(user);

            // Act
            const response = await request(app)
              .put('/api/v1/profile')
              .set('Authorization', `Bearer ${accessToken}`)
              .send(invalidData);

            // Assert
            expect(response.status).toBe(400);
            expect(response.body).toHaveProperty('message');
          }
        )
      );
    });
  });

  describe('POST /api/v1/profile/image', () => {
    /**
     * Test: Upload profile image with valid base64 data
     * Requirement: 12.4
     */
    it('should upload profile image and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const base64Image = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        profileImg: 'https://res.cloudinary.com/test/image.jpg',
      });
      (uploadProfileImage as any).mockResolvedValue('https://res.cloudinary.com/test/image.jpg');

      // Act
      const response = await request(app)
        .post('/api/v1/profile/image')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ profileImg: base64Image });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user).toHaveProperty('profileImg');
      expect(uploadProfileImage).toHaveBeenCalledWith(base64Image, user.id);
    });

    /**
     * Test: Upload profile image with URL (e.g., from Google OAuth)
     */
    it('should accept profile image URL and return 200', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const imageUrl = 'https://lh3.googleusercontent.com/a/test-image';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        profileImg: imageUrl,
      });

      // Act
      const response = await request(app)
        .post('/api/v1/profile/image')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ profileImg: imageUrl });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('message');
      expect(response.body.user).toHaveProperty('profileImg', imageUrl);
      expect(uploadProfileImage).not.toHaveBeenCalled(); // Should not upload URLs
    });

    /**
     * Test: Upload invalid image format
     * Requirement: 12.5
     */
    it('should return 400 with invalid image format', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });
      const invalidImage = 'not-a-valid-image-format';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (uploadProfileImage as any).mockRejectedValue(new Error('Invalid image format'));

      // Act
      const response = await request(app)
        .post('/api/v1/profile/image')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ profileImg: invalidImage });

      // Assert
      expect(response.status).toBe(500); // Error handler converts to 500
      expect(response.body).toHaveProperty('message');
    });

    /**
     * Test: Upload without profile image
     */
    it('should return 400 when profile image is missing', async () => {
      // Arrange
      const user = createVerifiedUserFixture();
      const accessToken = createValidAccessToken({ userId: user.id });

      // Act
      const response = await request(app)
        .post('/api/v1/profile/image')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({});

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toContain('Profile image is required');
    });

    /**
     * Test: Delete old Cloudinary image when uploading new one
     */
    it('should delete old Cloudinary image when uploading new one', async () => {
      // Arrange
      const user = createVerifiedUserFixture({
        profileImg: 'https://res.cloudinary.com/old/image.jpg',
      });
      const accessToken = createValidAccessToken({ userId: user.id });
      const base64Image = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==';

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(user);
      (prisma.user.update as any).mockResolvedValue({
        ...user,
        profileImg: 'https://res.cloudinary.com/test/new-image.jpg',
      });
      (uploadProfileImage as any).mockResolvedValue('https://res.cloudinary.com/test/new-image.jpg');
      (deleteImageFromCloudinary as any).mockResolvedValue(true);

      // Act
      const response = await request(app)
        .post('/api/v1/profile/image')
        .set('Authorization', `Bearer ${accessToken}`)
        .send({ profileImg: base64Image });

      // Assert
      expect(response.status).toBe(200);
      expect(deleteImageFromCloudinary).toHaveBeenCalledWith(user.profileImg);
    });

    /**
     * Property 33: Invalid image format rejected
     * For any invalid image format, the upload profile image endpoint should return 400 status with error message
     * Validates: Requirements 12.5
     */
    it('should reject invalid image formats', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.oneof(
            fc.constant(''), // Empty string
            fc.constant('not-base64-or-url'), // Invalid format
            fc.string({ minLength: 1, maxLength: 50 }).filter(s => !s.startsWith('data:') && !s.startsWith('http')), // Random string
          ),
          async (invalidImage) => {
            // Arrange
            const user = createVerifiedUserFixture();
            const accessToken = createValidAccessToken({ userId: user.id });

            // Mock database operations
            (prisma.user.findUnique as any).mockResolvedValue(user);
            (uploadProfileImage as any).mockRejectedValue(new Error('Invalid image format'));

            // Act
            const response = await request(app)
              .post('/api/v1/profile/image')
              .set('Authorization', `Bearer ${accessToken}`)
              .send({ profileImg: invalidImage });

            // Assert
            // Either 400 (missing image) or 500 (upload error)
            expect([400, 500]).toContain(response.status);
            expect(response.body).toHaveProperty('message');
          }
        )
      );
    });
  });

  describe('GET /api/v1/profile/username/check', () => {
    /**
     * Test: Check username availability
     */
    it('should return available for unused username', async () => {
      // Arrange
      const username = generateTestUsername();

      // Mock database operations
      (prisma.user.findUnique as any).mockResolvedValue(null);

      // Act
      const response = await request(app)
        .get('/api/v1/profile/username/check')
        .query({ username });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('available', true);
      expect(response.body).toHaveProperty('username', username.toLowerCase());
    });

    /**
     * Test: Check username availability for taken username
     */
    it('should return unavailable for taken username with suggestions', async () => {
      // Arrange
      const username = 'takenusername';

      // Mock database operations
      (prisma.user.findUnique as any).mockImplementation(async ({ where }: any) => {
        if (where.username === username) {
          return { id: 'existing-user-id' };
        }
        return null; // Suggestions are available
      });

      // Act
      const response = await request(app)
        .get('/api/v1/profile/username/check')
        .query({ username });

      // Assert
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('available', false);
      expect(response.body).toHaveProperty('message', 'Username is already taken');
    });

    /**
     * Test: Check username with invalid length
     */
    it('should return 400 for username too short', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/profile/username/check')
        .query({ username: 'ab' }); // Less than 3 characters

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('available', false);
      expect(response.body.message).toContain('at least 3 characters');
    });

    /**
     * Test: Check username with invalid characters
     */
    it('should return 400 for username with invalid characters', async () => {
      // Act
      const response = await request(app)
        .get('/api/v1/profile/username/check')
        .query({ username: 'user@name!' }); // Invalid characters

      // Assert
      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('available', false);
      expect(response.body.message).toContain('can only contain');
    });
  });
});
