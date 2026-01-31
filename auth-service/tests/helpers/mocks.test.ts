import { mockPrisma, mockRedis, mockResend, mockCloudinary, resetAllMocks } from './mocks';

describe('Mock Factories', () => {
  beforeEach(() => {
    resetAllMocks();
  });

  describe('mockPrisma', () => {
    it('should create a mock Prisma client with all methods', () => {
      const prisma = mockPrisma();
      
      expect(prisma.user.findUnique).toBeDefined();
      expect(prisma.user.findFirst).toBeDefined();
      expect(prisma.user.findMany).toBeDefined();
      expect(prisma.user.create).toBeDefined();
      expect(prisma.user.update).toBeDefined();
      expect(prisma.user.delete).toBeDefined();
      expect(prisma.session.findUnique).toBeDefined();
      expect(prisma.$transaction).toBeDefined();
      expect(prisma.$disconnect).toBeDefined();
    });

    it('should return null by default for findUnique', async () => {
      const prisma = mockPrisma();
      const result = await prisma.user.findUnique({ where: { id: '123' } });
      
      expect(result).toBeNull();
    });

    it('should return empty array by default for findMany', async () => {
      const prisma = mockPrisma();
      const result = await prisma.user.findMany();
      
      expect(result).toEqual([]);
    });

    it('should accept default return values in options', async () => {
      const mockUser = { id: '123', email: 'test@example.com', name: 'Test User' };
      const prisma = mockPrisma({
        user: {
          findUnique: mockUser,
        },
      });
      
      const result = await prisma.user.findUnique({ where: { id: '123' } });
      expect(result).toEqual(mockUser);
    });

    it('should allow overriding mock behavior per test', async () => {
      const prisma = mockPrisma();
      const customUser = { id: '456', email: 'custom@example.com' };
      
      prisma.user.findUnique.mockResolvedValue(customUser);
      
      const result = await prisma.user.findUnique({ where: { id: '456' } });
      expect(result).toEqual(customUser);
    });

    it('should mock $transaction with callback', async () => {
      const prisma = mockPrisma();
      const mockUser = { id: '123', email: 'test@example.com' };
      
      prisma.user.create.mockResolvedValue(mockUser);
      
      const result = await prisma.$transaction(async (tx: any) => {
        return await tx.user.create({ data: {} });
      });
      
      expect(result).toEqual(mockUser);
    });

    it('should mock $transaction with array', async () => {
      const prisma = mockPrisma();
      const operations = [
        Promise.resolve({ id: '1' }),
        Promise.resolve({ id: '2' }),
      ];
      
      const result = await prisma.$transaction(operations);
      expect(result).toEqual(operations);
    });

    it('should track method calls', async () => {
      const prisma = mockPrisma();
      
      await prisma.user.findUnique({ where: { id: '123' } });
      
      expect(prisma.user.findUnique).toHaveBeenCalledWith({ where: { id: '123' } });
      expect(prisma.user.findUnique).toHaveBeenCalledTimes(1);
    });
  });

  describe('mockRedis', () => {
    it('should create a mock Redis client with all methods', () => {
      const redis = mockRedis();
      
      expect(redis.get).toBeDefined();
      expect(redis.set).toBeDefined();
      expect(redis.del).toBeDefined();
      expect(redis.exists).toBeDefined();
      expect(redis.expire).toBeDefined();
      expect(redis.ttl).toBeDefined();
      expect(redis.incr).toBeDefined();
      expect(redis.keys).toBeDefined();
    });

    it('should return null for non-existent keys', async () => {
      const redis = mockRedis();
      const result = await redis.get('non-existent');
      
      expect(result).toBeNull();
    });

    it('should accept initial data in options', async () => {
      const redis = mockRedis({
        data: { 'test-key': 'test-value' },
      });
      
      const result = await redis.get('test-key');
      expect(result).toBe('test-value');
    });

    it('should store and retrieve values', async () => {
      const redis = mockRedis();
      
      await redis.set('key', 'value');
      const result = await redis.get('key');
      
      expect(result).toBe('value');
    });

    it('should handle set with expiration', async () => {
      const redis = mockRedis();
      
      await redis.set('key', 'value', 'EX', 60);
      const result = await redis.get('key');
      
      expect(result).toBe('value');
      expect(redis.set).toHaveBeenCalledWith('key', 'value', 'EX', 60);
    });

    it('should delete keys', async () => {
      const redis = mockRedis({
        data: { 'key1': 'value1', 'key2': 'value2' },
      });
      
      const count = await redis.del('key1');
      const result = await redis.get('key1');
      
      expect(count).toBe(1);
      expect(result).toBeNull();
    });

    it('should delete multiple keys', async () => {
      const redis = mockRedis({
        data: { 'key1': 'value1', 'key2': 'value2', 'key3': 'value3' },
      });
      
      const count = await redis.del(['key1', 'key2']);
      
      expect(count).toBe(2);
      expect(await redis.get('key1')).toBeNull();
      expect(await redis.get('key2')).toBeNull();
      expect(await redis.get('key3')).toBe('value3');
    });

    it('should check key existence', async () => {
      const redis = mockRedis({
        data: { 'existing-key': 'value' },
      });
      
      expect(await redis.exists('existing-key')).toBe(1);
      expect(await redis.exists('non-existent')).toBe(0);
    });

    it('should set expiration on existing key', async () => {
      const redis = mockRedis({
        data: { 'key': 'value' },
      });
      
      const result = await redis.expire('key', 60);
      expect(result).toBe(1);
    });

    it('should return 0 when setting expiration on non-existent key', async () => {
      const redis = mockRedis();
      
      const result = await redis.expire('non-existent', 60);
      expect(result).toBe(0);
    });

    it('should return TTL for key with expiration', async () => {
      const redis = mockRedis();
      
      await redis.set('key', 'value', 'EX', 60);
      const ttl = await redis.ttl('key');
      
      expect(ttl).toBe(60);
    });

    it('should return -2 for non-existent key TTL', async () => {
      const redis = mockRedis();
      
      const ttl = await redis.ttl('non-existent');
      expect(ttl).toBe(-2);
    });

    it('should increment counter', async () => {
      const redis = mockRedis();
      
      const count1 = await redis.incr('counter');
      const count2 = await redis.incr('counter');
      
      expect(count1).toBe(1);
      expect(count2).toBe(2);
    });

    it('should decrement counter', async () => {
      const redis = mockRedis({
        data: { 'counter': '5' },
      });
      
      const count = await redis.decr('counter');
      expect(count).toBe(4);
    });

    it('should find keys by pattern', async () => {
      const redis = mockRedis({
        data: {
          'otp:user1': '123456',
          'otp:user2': '654321',
          'session:abc': 'data',
        },
      });
      
      const otpKeys = await redis.keys('otp:*');
      expect(otpKeys).toHaveLength(2);
      expect(otpKeys).toContain('otp:user1');
      expect(otpKeys).toContain('otp:user2');
    });

    it('should flush all keys', async () => {
      const redis = mockRedis({
        data: { 'key1': 'value1', 'key2': 'value2' },
      });
      
      await redis.flushall();
      
      expect(await redis.get('key1')).toBeNull();
      expect(await redis.get('key2')).toBeNull();
    });
  });

  describe('mockResend', () => {
    it('should create a mock Resend client', () => {
      const resend = mockResend();
      
      expect(resend.emails.send).toBeDefined();
    });

    it('should return success response by default', async () => {
      const resend = mockResend();
      
      const result = await resend.emails.send({
        from: 'test@example.com',
        to: ['recipient@example.com'],
        subject: 'Test',
        html: '<p>Test</p>',
      });
      
      expect(result.data).toBeDefined();
      expect(result.data.id).toMatch(/^mock-email-id-/);
      expect(result.error).toBeNull();
    });

    it('should allow overriding send behavior', async () => {
      const resend = mockResend();
      
      resend.emails.send.mockResolvedValue({
        data: null,
        error: { message: 'Failed to send email' },
      });
      
      const result = await resend.emails.send({
        from: 'test@example.com',
        to: ['recipient@example.com'],
        subject: 'Test',
        html: '<p>Test</p>',
      });
      
      expect(result.data).toBeNull();
      expect(result.error).toBeDefined();
    });

    it('should track email send calls', async () => {
      const resend = mockResend();
      
      await resend.emails.send({
        from: 'test@example.com',
        to: ['recipient@example.com'],
        subject: 'Test',
        html: '<p>Test</p>',
      });
      
      expect(resend.emails.send).toHaveBeenCalledWith(
        expect.objectContaining({
          from: 'test@example.com',
          to: ['recipient@example.com'],
        })
      );
    });
  });

  describe('mockCloudinary', () => {
    it('should create a mock Cloudinary client', () => {
      const cloudinary = mockCloudinary();
      
      expect(cloudinary.uploader.upload).toBeDefined();
      expect(cloudinary.uploader.destroy).toBeDefined();
      expect(cloudinary.config).toBeDefined();
    });

    it('should return success response for upload by default', async () => {
      const cloudinary = mockCloudinary();
      
      const result = await cloudinary.uploader.upload('data:image/png;base64,abc123');
      
      expect(result.secure_url).toBeDefined();
      expect(result.public_id).toBeDefined();
      expect(result.format).toBe('jpg');
    });

    it('should accept custom upload URL in options', async () => {
      const customUrl = 'https://res.cloudinary.com/custom/image.jpg';
      const cloudinary = mockCloudinary({
        defaultUploadUrl: customUrl,
      });
      
      const result = await cloudinary.uploader.upload('data:image/png;base64,abc123');
      
      expect(result.secure_url).toBe(customUrl);
    });

    it('should use public_id from upload options', async () => {
      const cloudinary = mockCloudinary();
      
      const result = await cloudinary.uploader.upload('data:image/png;base64,abc123', {
        public_id: 'custom-public-id',
      });
      
      expect(result.public_id).toBe('custom-public-id');
    });

    it('should simulate upload failure when configured', async () => {
      const cloudinary = mockCloudinary({
        uploadSuccess: false,
      });
      
      await expect(
        cloudinary.uploader.upload('data:image/png;base64,abc123')
      ).rejects.toThrow('Upload failed');
    });

    it('should return success for destroy by default', async () => {
      const cloudinary = mockCloudinary();
      
      const result = await cloudinary.uploader.destroy('test-public-id');
      
      expect(result.result).toBe('ok');
    });

    it('should simulate delete failure when configured', async () => {
      const cloudinary = mockCloudinary({
        deleteSuccess: false,
      });
      
      const result = await cloudinary.uploader.destroy('test-public-id');
      
      expect(result.result).toBe('not found');
    });

    it('should track upload calls', async () => {
      const cloudinary = mockCloudinary();
      
      await cloudinary.uploader.upload('data:image/png;base64,abc123', {
        folder: 'test-folder',
      });
      
      expect(cloudinary.uploader.upload).toHaveBeenCalledWith(
        'data:image/png;base64,abc123',
        expect.objectContaining({
          folder: 'test-folder',
        })
      );
    });

    it('should track destroy calls', async () => {
      const cloudinary = mockCloudinary();
      
      await cloudinary.uploader.destroy('test-public-id');
      
      expect(cloudinary.uploader.destroy).toHaveBeenCalledWith('test-public-id');
    });

    it('should return config', () => {
      const cloudinary = mockCloudinary();
      
      const config = cloudinary.config();
      
      expect(config.cloud_name).toBe('test-cloud');
      expect(config.api_key).toBe('test-api-key');
      expect(config.api_secret).toBe('test-api-secret');
    });
  });

  describe('resetAllMocks', () => {
    it('should clear all mock call history', async () => {
      const prisma = mockPrisma();
      const redis = mockRedis();
      
      await prisma.user.findUnique({ where: { id: '123' } });
      await redis.get('key');
      
      expect(prisma.user.findUnique).toHaveBeenCalledTimes(1);
      expect(redis.get).toHaveBeenCalledTimes(1);
      
      resetAllMocks();
      
      expect(prisma.user.findUnique).toHaveBeenCalledTimes(0);
      expect(redis.get).toHaveBeenCalledTimes(0);
    });
  });
});
