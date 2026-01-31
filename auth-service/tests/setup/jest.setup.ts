/**
 * Global test setup for Jest
 * 
 * This file runs before all tests and sets up the global test environment.
 * It configures mocks, environment variables, and other global test utilities.
 */

// Set up test environment variables before any imports
// JWT configuration
process.env.JWT_ACCESS_SECRET = 'test-jwt-secret-key-for-testing-only';
process.env.REFRESH_TOKEN_SECRET = 'test-jwt-refresh-secret-key-for-testing-only';
process.env.ACCESS_TOKEN_TTL_SEC = '900'; // 15 minutes
process.env.REFRESH_TOKEN_TTL_SEC = '2592000'; // 30 days

// Encryption configuration
process.env.ENCRYPTION_KEY = 'test-encryption-key-32-bytes!!';

// Redis configuration
process.env.REDIS_HOST = 'localhost';
process.env.REDIS_PORT = '6379';

// Email configuration
process.env.RESEND_API_KEY = 'test-resend-api-key';
process.env.EMAIL_FROM = 'test@example.com';

// Cloudinary configuration
process.env.CLOUDINARY_CLOUD_NAME = 'test-cloud';
process.env.CLOUDINARY_API_KEY = 'test-api-key';
process.env.CLOUDINARY_API_SECRET = 'test-api-secret';

// Location API configuration
process.env.IPAPI_KEY = 'test-ipapi-key';

// Google OAuth configuration
process.env.GOOGLE_CLIENT_ID = 'test-google-client-id';

// OTP configuration
process.env.OTP_LENGTH = '6';
process.env.OTP_TTL_SEC = '300'; // 5 minutes
process.env.OTP_MAX_ATTEMPTS = '3';

// Email verification configuration
process.env.EMAIL_VERIFICATION_COOLDOWN_SEC = '60'; // 1 minute
process.env.EMAIL_VERIFICATION_MAX_ATTEMPTS = '5';
process.env.EMAIL_VERIFICATION_EXTENDED_COOLDOWN_SEC = '300'; // 5 minutes

// Password reset configuration
process.env.PASSWORD_RESET_TOKEN_TTL_SEC = '3600'; // 1 hour

// Session configuration
process.env.SESSION_CLEANUP_INTERVAL_MS = '3600000'; // 1 hour

// 2FA configuration
process.env.TWO_FACTOR_ISSUER = 'TestApp';
process.env.TWO_FACTOR_BACKUP_CODES_COUNT = '8';

// Internal service configuration
process.env.INTERNAL_SERVICE_SECRET = 'test-internal-secret';

// Clean up after each test
afterEach(() => {
  jest.clearAllMocks();
  jest.restoreAllMocks();
});
