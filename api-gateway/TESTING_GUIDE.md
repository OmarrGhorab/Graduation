# API Gateway Testing Guide

This guide explains how to test the refactored API Gateway with existing services.

## Prerequisites

Before testing, ensure you have:

1. ✅ All services configured with proper `.env` files
2. ✅ Dependencies installed (`npm install` in each service)
3. ✅ Database and Redis connections available (for auth and notification services)

## Quick Start

### 1. Start All Services

Open three separate terminals:

**Terminal 1 - Auth Service:**
```bash
cd auth-service
npm start
```

**Terminal 2 - Notification Service:**
```bash
cd notification-service
npm start
```

**Terminal 3 - API Gateway:**
```bash
cd api-gateway
npm start
```

### 2. Run Integration Tests

In a fourth terminal:

```bash
cd api-gateway

# Basic integration test
node test-integration.js

# Comprehensive integration test
node test-comprehensive.js
```

## Test Scripts

### test-integration.js

Basic integration test that verifies:
- Health check endpoint
- Auth service proxy routes
- Notification service proxy routes
- Location request proxy routes

**Expected Output:** All tests should pass (5/5)

### test-comprehensive.js

Comprehensive test suite that verifies:
- Health check with upstream service status
- Multiple auth service endpoints
- Multiple notification service endpoints
- Error handling (404, 401, 400)
- CORS headers
- Backward compatibility

**Expected Output:** All tests should pass (10/10)

## Manual Testing

You can also test endpoints manually using curl or a REST client:

### Health Check
```bash
curl http://localhost:3000/health
```

Expected: JSON response with gateway and upstream service status

### Auth Service (via Gateway)
```bash
# Root endpoint
curl http://localhost:3000/

# Register (will fail with 400 - missing fields)
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'
```

### Notification Service (via Gateway)
```bash
# Get notifications (will fail with 401 - no auth)
curl http://localhost:3000/api/v1/notifications \
  -H "Authorization: Bearer fake-token"
```

## Expected Behaviors

### Success Cases (200 OK)
- `GET /health` - Gateway health check
- `GET /` - Auth service root

### Authentication Required (401)
- `GET /api/v1/notifications` - Without valid token
- `POST /api/v1/location/request/:childId` - Without valid token

### Bad Request (400)
- `POST /api/v1/auth/register` - Missing required fields
- `POST /api/v1/auth/login` - Missing required fields

### Not Found (404)
- Any non-existent route (proxied to auth service, which returns 404)

## Troubleshooting

### Services Won't Start

**Auth Service Error: "Cannot find module 'debug-logger'"**
- Solution: The file `auth-service/src/utils/debug-logger.ts` should exist
- If missing, it was created during testing

**Database Connection Error**
- Check your `.env` file has correct `DATABASE_URL`
- Ensure PostgreSQL is running and accessible

**Redis Connection Error**
- Check your `.env` file has correct `REDIS_URL`
- Ensure Redis is running and accessible

### Gateway Won't Start

**Missing Environment Variables**
- Ensure `api-gateway/.env` has all required variables:
  - `PORT`
  - `NODE_ENV`
  - `ALLOWED_ORIGINS`
  - `AUTH_SERVICE_URL`
  - `NOTIFICATION_SERVICE_URL`

### Tests Fail

**Connection Refused**
- Ensure all three services are running
- Check that services are running on expected ports:
  - API Gateway: 3000
  - Auth Service: 6001
  - Notification Service: 6003

**Unexpected Status Codes**
- This is normal for protected endpoints (401, 403)
- This is normal for invalid requests (400)
- Check the test output to see if the failure is expected

## Continuous Testing

For development, you can run services in watch mode:

```bash
# In each service directory
npm run dev
```

This will automatically restart the service when code changes.

## Test Coverage

To check unit test coverage:

```bash
cd api-gateway
npm run test:coverage
```

Expected: 80%+ coverage across all modules

## Integration with CI/CD

These integration tests can be automated in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Start Services
  run: |
    cd auth-service && npm start &
    cd notification-service && npm start &
    cd api-gateway && npm start &
    sleep 5

- name: Run Integration Tests
  run: |
    cd api-gateway
    node test-integration.js
    node test-comprehensive.js
```

## Next Steps

After successful testing:

1. ✅ Review test results in `INTEGRATION_TEST_REPORT.md`
2. ✅ Deploy to staging environment
3. ✅ Run smoke tests in staging
4. ✅ Deploy to production

## Support

For issues or questions:
- Check the `README.md` for architecture details
- Review the `INTEGRATION_TEST_REPORT.md` for test results
- Check service logs for detailed error messages
