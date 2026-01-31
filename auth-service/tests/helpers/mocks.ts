
import { PrismaClient } from '@prisma/client';
import type Redis from 'ioredis';
import type { Resend } from 'resend';

/**
 * Options for configuring the Prisma mock
 */
export interface MockPrismaOptions {
  /**
   * Default return values for user queries
   */
  user?: {
    findUnique?: any;
    findFirst?: any;
    findMany?: any;
    create?: any;
    update?: any;
    delete?: any;
    count?: any;
  };

  /**
   * Default return values for session queries
   */
  session?: {
    findUnique?: any;
    findFirst?: any;
    findMany?: any;
    create?: any;
    update?: any;
    delete?: any;
    deleteMany?: any;
    count?: any;
  };

  /**
   * Default return values for userDevice queries
   */
  userDevice?: {
    findUnique?: any;
    findFirst?: any;
    findMany?: any;
    create?: any;
    update?: any;
    upsert?: any;
  };

  /**
   * Default return values for authProvider queries
   */
  authProvider?: {
    findUnique?: any;
    findFirst?: any;
    create?: any;
    update?: any;
  };

  /**
   * Default return values for parentLinkRequest queries
   */
  parentLinkRequest?: {
    findUnique?: any;
    findFirst?: any;
    findMany?: any;
    create?: any;
    update?: any;
    delete?: any;
  };

  /**
   * Default return values for parentChildLink queries
   */
  parentChildLink?: {
    findUnique?: any;
    findFirst?: any;
    findMany?: any;
    create?: any;
    delete?: any;
  };

  /**
   * Default return values for locationHistory queries
   */
  locationHistory?: {
    findMany?: any;
    create?: any;
  };

  /**
   * Default return values for userPreference queries
   */
  userPreference?: {
    findUnique?: any;
    create?: any;
    update?: any;
    upsert?: any;
  };
}

/**
 * Creates a mock Prisma client for testing
 * 
 * This factory creates a mock PrismaClient with all common database operations
 * mocked using Vitest. Each method returns a mock function that can be configured
 * per test.
 * 
 * @param options - Optional default return values for queries
 * @returns Mocked PrismaClient instance
 * 
 * @example
 * ```typescript
 * const prisma = mockPrisma({
 *   user: {
 *     findUnique: { id: '123', email: 'test@example.com' }
 *   }
 * });
 * 
 * // In test, override specific behavior
 * prisma.user.findUnique.mockResolvedValue(null);
 * ```
 */
export function mockPrisma(options: MockPrismaOptions = {}): any {
  const mock = {
    user: {
      findUnique: jest.fn().mockResolvedValue(options.user?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.user?.findFirst ?? null),
      findMany: jest.fn().mockResolvedValue(options.user?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.user?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.user?.update ?? {}),
      delete: jest.fn().mockResolvedValue(options.user?.delete ?? {}),
      count: jest.fn().mockResolvedValue(options.user?.count ?? 0),
      upsert: jest.fn().mockResolvedValue({}),
    },
    session: {
      findUnique: jest.fn().mockResolvedValue(options.session?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.session?.findFirst ?? null),
      findMany: jest.fn().mockResolvedValue(options.session?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.session?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.session?.update ?? {}),
      delete: jest.fn().mockResolvedValue(options.session?.delete ?? {}),
      deleteMany: jest.fn().mockResolvedValue(options.session?.deleteMany ?? { count: 0 }),
      count: jest.fn().mockResolvedValue(options.session?.count ?? 0),
      updateMany: jest.fn().mockResolvedValue({ count: 0 }),
    },
    userDevice: {
      findUnique: jest.fn().mockResolvedValue(options.userDevice?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.userDevice?.findFirst ?? null),
      findMany: jest.fn().mockResolvedValue(options.userDevice?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.userDevice?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.userDevice?.update ?? {}),
      upsert: jest.fn().mockResolvedValue(options.userDevice?.upsert ?? {}),
      delete: jest.fn().mockResolvedValue({}),
      deleteMany: jest.fn().mockResolvedValue({ count: 0 }),
      count: jest.fn().mockResolvedValue(0),
    },
    authProvider: {
      findUnique: jest.fn().mockResolvedValue(options.authProvider?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.authProvider?.findFirst ?? null),
      create: jest.fn().mockResolvedValue(options.authProvider?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.authProvider?.update ?? {}),
      upsert: jest.fn().mockResolvedValue({}),
    },
    parentLinkRequest: {
      findUnique: jest.fn().mockResolvedValue(options.parentLinkRequest?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.parentLinkRequest?.findFirst ?? null),
      findMany: jest.fn().mockResolvedValue(options.parentLinkRequest?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.parentLinkRequest?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.parentLinkRequest?.update ?? {}),
      delete: jest.fn().mockResolvedValue(options.parentLinkRequest?.delete ?? {}),
      upsert: jest.fn().mockResolvedValue({}),
    },
    parentChildLink: {
      findUnique: jest.fn().mockResolvedValue(options.parentChildLink?.findUnique ?? null),
      findFirst: jest.fn().mockResolvedValue(options.parentChildLink?.findFirst ?? null),
      findMany: jest.fn().mockResolvedValue(options.parentChildLink?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.parentChildLink?.create ?? {}),
      delete: jest.fn().mockResolvedValue(options.parentChildLink?.delete ?? {}),
      count: jest.fn().mockResolvedValue(0),
    },
    locationHistory: {
      findMany: jest.fn().mockResolvedValue(options.locationHistory?.findMany ?? []),
      create: jest.fn().mockResolvedValue(options.locationHistory?.create ?? {}),
      deleteMany: jest.fn().mockResolvedValue({ count: 0 }),
      count: jest.fn().mockResolvedValue(0),
    },
    userPreference: {
      findUnique: jest.fn().mockResolvedValue(options.userPreference?.findUnique ?? null),
      create: jest.fn().mockResolvedValue(options.userPreference?.create ?? {}),
      update: jest.fn().mockResolvedValue(options.userPreference?.update ?? {}),
      upsert: jest.fn().mockResolvedValue(options.userPreference?.upsert ?? {}),
    },
    interest: {
      findUnique: jest.fn().mockResolvedValue(null),
      findMany: jest.fn().mockResolvedValue([]),
      create: jest.fn().mockResolvedValue({}),
    },
    userInterest: {
      findMany: jest.fn().mockResolvedValue([]),
      create: jest.fn().mockResolvedValue({}),
      deleteMany: jest.fn().mockResolvedValue({ count: 0 }),
    },
    courseEnrollment: {
      findMany: jest.fn().mockResolvedValue([]),
      create: jest.fn().mockResolvedValue({}),
    },
    unlinkRequest: {
      findUnique: jest.fn().mockResolvedValue(null),
      findFirst: jest.fn().mockResolvedValue(null),
      findMany: jest.fn().mockResolvedValue([]),
      create: jest.fn().mockResolvedValue({}),
      update: jest.fn().mockResolvedValue({}),
      delete: jest.fn().mockResolvedValue({}),
    },

    $transaction: jest.fn().mockImplementation((callback) => {
      if (typeof callback === 'function') {
        return callback(mock);
      }
      return Promise.resolve(callback);
    }),
    $disconnect: jest.fn().mockResolvedValue(undefined),
    $connect: jest.fn().mockResolvedValue(undefined),
  };

  return mock as unknown as PrismaClient;
}

/**
 * Options for configuring the Redis mock
 */
export interface MockRedisOptions {
  /**
   * Initial data to populate the mock Redis store
   * Key-value pairs that will be returned by get() calls
   */
  data?: Record<string, string>;
}

/**
 * Creates a mock Redis client for testing
 * 
 * This factory creates a mock Redis client with common operations mocked.
 * It maintains an in-memory store for get/set operations to simulate Redis behavior.
 * 
 * @param options - Optional initial data for the mock store
 * @returns Mocked Redis client instance
 * 
 * @example
 * ```typescript
 * const redis = mockRedis({
 *   data: { 'otp:test@example.com': '123456' }
 * });
 * 
 * // In test
 * await redis.get('otp:test@example.com'); // Returns '123456'
 * await redis.set('key', 'value');
 * ```
 */
export function mockRedis(options: MockRedisOptions = {}): any {
  // In-memory store to simulate Redis behavior
  const store: Record<string, string> = { ...(options.data || {}) };
  const ttlStore: Record<string, number> = {};

  const mock: any = {
    get: jest.fn().mockImplementation((key: string) => {
      return Promise.resolve(store[key] || null);
    }),

    set: jest.fn().mockImplementation((key: string, value: string, ...args: any[]) => {
      store[key] = value;

      // Handle EX (expiration in seconds) option
      if (args[0] === 'EX' && typeof args[1] === 'number') {
        ttlStore[key] = args[1];
      }

      return Promise.resolve('OK');
    }),

    del: jest.fn().mockImplementation((key: string | string[]) => {
      const keys = Array.isArray(key) ? key : [key];
      let count = 0;

      keys.forEach((k) => {
        if (store[k]) {
          delete store[k];
          delete ttlStore[k];
          count++;
        }
      });

      return Promise.resolve(count);
    }),

    exists: jest.fn().mockImplementation((key: string) => {
      return Promise.resolve(store[key] ? 1 : 0);
    }),

    expire: jest.fn().mockImplementation((key: string, seconds: number) => {
      if (store[key]) {
        ttlStore[key] = seconds;
        return Promise.resolve(1);
      }
      return Promise.resolve(0);
    }),

    ttl: jest.fn().mockImplementation((key: string) => {
      return Promise.resolve(ttlStore[key] || -2);
    }),

    incr: jest.fn().mockImplementation((key: string) => {
      const current = parseInt(store[key] || '0', 10);
      const newValue = current + 1;
      store[key] = String(newValue);
      return Promise.resolve(newValue);
    }),

    decr: jest.fn().mockImplementation((key: string) => {
      const current = parseInt(store[key] || '0', 10);
      const newValue = current - 1;
      store[key] = String(newValue);
      return Promise.resolve(newValue);
    }),

    keys: jest.fn().mockImplementation((pattern: string) => {
      // Simple pattern matching for wildcards
      const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$');
      return Promise.resolve(Object.keys(store).filter(key => regex.test(key)));
    }),

    sadd: jest.fn().mockImplementation((key: string, ...members: string[]) => {
      // Simple set implementation - just track that members were added
      if (!store[key]) {
        store[key] = JSON.stringify(members);
      } else {
        const existing = JSON.parse(store[key] || '[]');
        const combined = [...new Set([...existing, ...members])];
        store[key] = JSON.stringify(combined);
      }
      return Promise.resolve(members.length);
    }),

    smembers: jest.fn().mockImplementation((key: string) => {
      const value = store[key];
      if (!value) return Promise.resolve([]);
      try {
        return Promise.resolve(JSON.parse(value));
      } catch {
        return Promise.resolve([]);
      }
    }),

    srem: jest.fn().mockImplementation((key: string, ...members: string[]) => {
      if (!store[key]) return Promise.resolve(0);
      try {
        const existing = JSON.parse(store[key] || '[]');
        const filtered = existing.filter((m: string) => !members.includes(m));
        store[key] = JSON.stringify(filtered);
        return Promise.resolve(existing.length - filtered.length);
      } catch {
        return Promise.resolve(0);
      }
    }),

    sismember: jest.fn().mockImplementation((key: string, member: string) => {
      if (!store[key]) return Promise.resolve(0);
      try {
        const members = JSON.parse(store[key] || '[]');
        return Promise.resolve(members.includes(member) ? 1 : 0);
      } catch {
        return Promise.resolve(0);
      }
    }),
  };

  // Add pipeline after mock is defined so it can reference mock methods
  mock.pipeline = jest.fn().mockImplementation(() => {
    // Return a mock pipeline object
    const commands: any[] = [];
    const pipelineObj: any = {
      set: jest.fn().mockImplementation((...args: any[]) => {
        commands.push({ cmd: 'set', args });
        return pipelineObj;
      }),
      sadd: jest.fn().mockImplementation((...args: any[]) => {
        commands.push({ cmd: 'sadd', args });
        return pipelineObj;
      }),
      del: jest.fn().mockImplementation((...args: any[]) => {
        commands.push({ cmd: 'del', args });
        return pipelineObj;
      }),
      expire: jest.fn().mockImplementation((...args: any[]) => {
        commands.push({ cmd: 'expire', args });
        return pipelineObj;
      }),
      exec: jest.fn().mockImplementation(async () => {
        // Execute all commands in the pipeline
        const results = [];
        for (const { cmd, args } of commands) {
          try {
            let result;
            if (cmd === 'set') {
              result = await mock.set(...args);
            } else if (cmd === 'sadd') {
              result = await mock.sadd(...args);
            } else if (cmd === 'del') {
              result = await mock.del(...args);
            } else if (cmd === 'expire') {
              result = await mock.expire(...args);
            } else {
              result = 'OK';
            }
            results.push([null, result]);
          } catch (error) {
            results.push([error, null]);
          }
        }
        return results;
      }),
    };
    return pipelineObj;
  });

  mock.flushall = jest.fn().mockImplementation(() => {
    Object.keys(store).forEach(key => delete store[key]);
    Object.keys(ttlStore).forEach(key => delete ttlStore[key]);
    return Promise.resolve('OK');
  });

  mock.quit = jest.fn().mockResolvedValue('OK');
  mock.disconnect = jest.fn().mockResolvedValue(undefined);

  // Event emitter methods (for compatibility)
  mock.on = jest.fn();
  mock.once = jest.fn();
  mock.emit = jest.fn();

  return mock as unknown as Redis;
}

/**
 * Creates a mock Resend email client for testing
 * 
 * This factory creates a mock Resend client with the emails.send method mocked.
 * By default, it simulates successful email sending.
 * 
 * @returns Mocked Resend client instance
 * 
 * @example
 * ```typescript
 * const resend = mockResend();
 * 
 * // In test, verify email was sent
 * await resend.emails.send({ to: 'test@example.com', subject: 'Test' });
 * expect(resend.emails.send).toHaveBeenCalledWith(
 *   expect.objectContaining({ to: ['test@example.com'] })
 * );
 * 
 * // Simulate email failure
 * resend.emails.send.mockResolvedValue({ 
 *   data: null, 
 *   error: { message: 'Failed to send' } 
 * });
 * ```
 */
export function mockResend(): any {
  const mock = {
    emails: {
      send: jest.fn().mockResolvedValue({
        data: { id: 'mock-email-id-' + Date.now() },
        error: null,
      }),
    },
  };

  return mock as unknown as Resend;
}

/**
 * Options for configuring the Cloudinary mock
 */
export interface MockCloudinaryOptions {
  /**
   * Default secure URL to return from upload operations
   */
  defaultUploadUrl?: string;

  /**
   * Whether upload operations should succeed by default
   */
  uploadSuccess?: boolean;

  /**
   * Whether delete operations should succeed by default
   */
  deleteSuccess?: boolean;
}

/**
 * Creates a mock Cloudinary client for testing
 * 
 * This factory creates a mock Cloudinary v2 client with uploader methods mocked.
 * By default, it simulates successful upload and delete operations.
 * 
 * @param options - Optional configuration for mock behavior
 * @returns Mocked Cloudinary client instance
 * 
 * @example
 * ```typescript
 * const cloudinary = mockCloudinary({
 *   defaultUploadUrl: 'https://res.cloudinary.com/test/image.jpg'
 * });
 * 
 * // In test
 * const result = await cloudinary.uploader.upload('data:image/png;base64,...');
 * expect(result.secure_url).toBe('https://res.cloudinary.com/test/image.jpg');
 * 
 * // Simulate upload failure
 * cloudinary.uploader.upload.mockRejectedValue(new Error('Upload failed'));
 * ```
 */
export function mockCloudinary(options: MockCloudinaryOptions = {}): any {
  const {
    defaultUploadUrl = 'https://res.cloudinary.com/test-cloud/image/upload/v1234567890/test-image.jpg',
    uploadSuccess = true,
    deleteSuccess = true,
  } = options;

  const mock = {
    uploader: {
      upload: jest.fn().mockImplementation((file: string, uploadOptions?: any) => {
        if (uploadSuccess) {
          return Promise.resolve({
            public_id: uploadOptions?.public_id || 'test-public-id',
            secure_url: defaultUploadUrl,
            url: defaultUploadUrl.replace('https://', 'http://'),
            format: 'jpg',
            width: 800,
            height: 600,
            bytes: 102400,
            created_at: new Date().toISOString(),
          });
        } else {
          return Promise.reject(new Error('Upload failed'));
        }
      }),

      destroy: jest.fn().mockImplementation((publicId: string) => {
        if (deleteSuccess) {
          return Promise.resolve({
            result: 'ok',
          });
        } else {
          return Promise.resolve({
            result: 'not found',
          });
        }
      }),
    },

    config: jest.fn().mockReturnValue({
      cloud_name: 'test-cloud',
      api_key: 'test-api-key',
      api_secret: 'test-api-secret',
    }),
  };

  return mock;
}

/**
 * Helper function to reset all mocks
 * 
 * This should be called in afterEach() to ensure test isolation
 * 
 * @example
 * ```typescript
 * afterEach(() => {
 *   jest.clearAllMocks();
 * });
 * ```
 */
export function resetAllMocks(): void {
  jest.clearAllMocks();
}
