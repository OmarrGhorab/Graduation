# Environment Variables Examples

This document contains example `.env` files for each service. Copy the relevant section to create your `.env` file in each service directory.

---

## API Gateway (.env)

Create this file at: `api-gateway/.env`

```env
# API Gateway Environment Variables

# Server Configuration
PORT=3000
NODE_ENV=development

# CORS Configuration
# Comma-separated list of allowed origins
# Use "*" to allow all origins (not recommended for production)
# Mobile apps typically don't send origin header, so they'll be allowed
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Arcjet Protection (Optional)
# Get your key from https://arcjet.com
# Leave empty to disable Arcjet protection
ARCJET_KEY=
```

---

## Auth Service (.env)

Create this file at: `auth-service/.env`

```env
# Auth Service Environment Variables

# Server Configuration
PORT=6001
NODE_ENV=development
SERVICE_NAME=Auth Service

# Database Configuration
DATABASE_URL=postgresql://username:password@localhost:5432/graduation_db

# JWT Configuration
# IMPORTANT: JWT_ACCESS_SECRET must match notification-service
JWT_ACCESS_SECRET=your-super-secret-access-token-key-change-this-in-production
JWT_REFRESH_SECRET=your-super-secret-refresh-token-key-change-this-in-production

# Token Expiration (in seconds)
ACCESS_TOKEN_TTL_SEC=900          # 15 minutes
REFRESH_TOKEN_TTL_SEC=2592000    # 30 days

# Redis Configuration
REDIS_URL=redis://localhost:6379

# Notification Service Integration
NOTIFICATION_SERVICE_URL=http://localhost:6003
INTERNAL_SERVICE_SECRET=your-internal-service-secret-must-match-notification-service

# Email Configuration (Resend)
RESEND_API_KEY=re_your_resend_api_key_here
EMAIL_FROM=onboarding@resend.dev

# Cloudinary Configuration (for image uploads)
CLOUDINARY_CLOUD_NAME=your-cloudinary-cloud-name
CLOUDINARY_API_KEY=your-cloudinary-api-key
CLOUDINARY_API_SECRET=your-cloudinary-api-secret

# Google OAuth Configuration (Optional)
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com

# Two-Factor Authentication Encryption Key
# Generate a secure random key: openssl rand -base64 32
TWO_FACTOR_ENCRYPTION_KEY=your-two-factor-encryption-key-32-chars-minimum

# OTP Configuration
OTP_TTL_SEC=600                   # 10 minutes
OTP_ATTEMPT_LIMIT=5               # Max attempts before cooldown
OTP_COOLDOWN_SEC=900              # 15 minutes cooldown after max attempts

# Email Verification Configuration
EMAIL_VERIFICATION_COOLDOWN_SEC=900        # 15 minutes between verification attempts
EMAIL_VERIFICATION_LONG_COOLDOWN_SEC=3600  # 60 minutes after max attempts
EMAIL_VERIFICATION_MAX_ATTEMPTS=5          # Max verification attempts
RESEND_OTP_COOLDOWN_SEC=60                 # 1 minute between resend requests
RESEND_OTP_MAX_ATTEMPTS=5                  # Max resend attempts per window
RESEND_OTP_ATTEMPTS_WINDOW_SEC=3600        # 1 hour window for resend attempts

# Password Reset Configuration
FORGOT_PASSWORD_COOLDOWN_SEC=300           # 5 minutes between forgot password requests
FORGOT_PASSWORD_LONG_COOLDOWN_SEC=1800     # 30 minutes after max attempts
FORGOT_PASSWORD_MAX_ATTEMPTS=3              # Max forgot password attempts
RESET_PASSWORD_COOLDOWN_SEC=60              # 1 minute between reset attempts
RESET_PASSWORD_LONG_COOLDOWN_SEC=1800       # 30 minutes after max attempts
RESET_PASSWORD_MAX_ATTEMPTS=3               # Max reset password attempts

# Arcjet Configuration (Optional)
ARCJET_KEY=
```

---

## Notification Service (.env)

Create this file at: `notification-service/.env`

```env
# Notification Service Environment Variables

# Server Configuration
PORT=6003
NODE_ENV=development

# Database Configuration
DATABASE_URL=postgresql://username:password@localhost:5432/graduation_db

# JWT Configuration
# IMPORTANT: JWT_ACCESS_SECRET must match auth-service
JWT_ACCESS_SECRET=your-super-secret-access-token-key-change-this-in-production

# Internal Service Authentication
# IMPORTANT: INTERNAL_SERVICE_SECRET must match auth-service
INTERNAL_SERVICE_SECRET=your-internal-service-secret-must-match-auth-service

# CORS Configuration
# Comma-separated list of allowed origins
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Redis Configuration (Optional - for future use)
REDIS_URL=redis://localhost:6379

# Firebase Cloud Messaging (FCM) Configuration
# Get these from Firebase Console > Project Settings > Service Accounts
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CLIENT_EMAIL=firebase-adminsdk-xxxxx@your-project-id.iam.gserviceaccount.com

# Private key from Firebase service account JSON file
# Replace \n with actual newlines or use \\n in .env file
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\nYOUR_PRIVATE_KEY_HERE\n-----END PRIVATE KEY-----\n"
```

---

## Important Notes

### 🔐 Shared Secrets

These secrets **MUST** match between services:

1. **JWT_ACCESS_SECRET**
   - Must be identical in `auth-service/.env` and `notification-service/.env`
   - Used to verify JWT tokens issued by auth service

2. **INTERNAL_SERVICE_SECRET**
   - Must be identical in `auth-service/.env` and `notification-service/.env`
   - Used for inter-service authentication

### 🔥 Firebase Setup

To get Firebase credentials:

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Select your project
3. Go to **Project Settings** > **Service Accounts**
4. Click **Generate New Private Key**
5. Download the JSON file
6. Extract these values:
   - `project_id` → `FIREBASE_PROJECT_ID`
   - `client_email` → `FIREBASE_CLIENT_EMAIL`
   - `private_key` → `FIREBASE_PRIVATE_KEY`

**Important:** When copying `private_key` to `.env`:
- Keep the `-----BEGIN PRIVATE KEY-----` and `-----END PRIVATE KEY-----` lines
- Replace actual newlines with `\n` or `\\n` depending on your env parser
- Keep it in quotes: `"-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"`

### 🔑 Generating Secure Keys

Generate secure random keys using:

```bash
# For JWT secrets (32+ characters recommended)
openssl rand -base64 32

# For internal service secret
openssl rand -base64 32

# For two-factor encryption key
openssl rand -base64 32
```

### 📝 Quick Setup Commands

```bash
# Create .env files from examples
cp ENV_EXAMPLES.md api-gateway/.env.example
cp ENV_EXAMPLES.md auth-service/.env.example
cp ENV_EXAMPLES.md notification-service/.env.example

# Or manually create each .env file and copy the relevant section
```

### ✅ Validation Checklist

Before starting services, verify:

- [ ] All `.env` files are created in their respective directories
- [ ] `JWT_ACCESS_SECRET` matches in auth-service and notification-service
- [ ] `INTERNAL_SERVICE_SECRET` matches in auth-service and notification-service
- [ ] Database URLs are correct and databases exist
- [ ] Redis URL is correct (if using Redis)
- [ ] Firebase credentials are valid
- [ ] All required API keys are set (Resend, Cloudinary, etc.)

---

## Environment-Specific Configurations

### Development
```env
NODE_ENV=development
# Use localhost URLs
# Enable verbose logging
```

### Production
```env
NODE_ENV=production
# Use production database URLs
# Use production API keys
# Disable debug logging
# Use strong secrets (never use defaults)
```

---

## Troubleshooting

### "JWT_SECRET not configured" Error
- Make sure `JWT_ACCESS_SECRET` is set (not `JWT_SECRET`)
- Verify it matches between auth-service and notification-service

### CORS Errors
- Check `ALLOWED_ORIGINS` in API Gateway
- For mobile apps, ensure CORS allows requests with no origin

### Firebase Initialization Failed
- Verify `FIREBASE_PRIVATE_KEY` has proper newline characters
- Check that all Firebase env vars are set
- Ensure Firebase service account has proper permissions

### Internal Service Authentication Failed
- Verify `INTERNAL_SERVICE_SECRET` matches in both services
- Check that auth-service is sending the header: `x-internal-secret`

