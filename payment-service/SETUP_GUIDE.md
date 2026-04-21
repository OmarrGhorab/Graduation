# Payment Service Setup Guide

Complete guide for setting up the payment service with cart and subscription features.

## Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- Kafka (optional, for event streaming)

## Installation Steps

### 1. Clone and Install Dependencies

```bash
cd payment-service
go mod download
```

### 2. Database Setup

Run the migrations in order:

```bash
# Connect to PostgreSQL
psql -U postgres -d your_database

# Run migrations
\i migrations/001_create_payment_tables.sql
\i migrations/002_add_cart_and_subscriptions.sql
\i migrations/003_add_payment_methods.sql
```

Or use a migration tool like `golang-migrate`:

```bash
migrate -path migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" up
```

### 3. Environment Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` and configure:

#### Required Settings

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=your_database

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Internal Service Secret (shared across microservices)
INTERNAL_SERVICE_SECRET=your-secret-key

# Paymob Credentials
PAYMOB_API_KEY=your-api-key
PAYMOB_CARD_INTEGRATION_ID=your-card-integration-id
PAYMOB_WALLET_INTEGRATION_ID=your-wallet-integration-id
PAYMOB_IFRAME_ID=your-iframe-id
PAYMOB_HMAC_SECRET=your-hmac-secret

# Service URLs
AUTH_SERVICE_URL=http://localhost:6001
COURSES_SERVICE_URL=http://localhost:8085
```

#### Optional Settings

```env
# Email (for subscription notifications)
RESEND_API_KEY=re_your_api_key_here
EMAIL_FROM=onboarding@resend.dev
EMAIL_FROM_NAME=Payment Service

# Kafka (for event streaming)
KAFKA_BROKERS=localhost:9092
```

### 4. Email Setup (Optional but Recommended)

The payment service uses **Resend** for sending emails (same as auth service).

**Quick Setup:**
1. Create account at [resend.com](https://resend.com)
2. Get API key from [API Keys page](https://resend.com/api-keys)
3. Add to `.env`:
   ```env
   RESEND_API_KEY=re_your_actual_api_key_here
   EMAIL_FROM=onboarding@resend.dev
   EMAIL_FROM_NAME=Payment Service
   ```

**For Production:**
- Add custom domain in Resend dashboard
- Update `EMAIL_FROM` to use your domain
- See `RESEND_SETUP.md` for detailed instructions

If Resend is not configured, emails will be logged to console instead of being sent.

### 5. Run the Service

```bash
# Development
go run cmd/server/main.go

# Production build
go build -o payment-service cmd/server/main.go
./payment-service
```

The service will start on port `8090` by default (configurable via `SERVER_PORT`).

## Features Overview

### 1. Shopping Cart

Users can add multiple courses to their cart before checkout:

```bash
# Add course to cart
curl -X POST http://localhost:8090/api/v1/cart/add \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "courseId": "course-uuid",
    "billingType": "ONE_TIME"
  }'

# View cart
curl -X GET http://localhost:8090/api/v1/cart \
  -H "Authorization: Bearer <token>"

# Checkout cart
curl -X POST http://localhost:8090/api/v1/cart/checkout \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "paymentMethod": "CARD",
    "phoneNumber": "01234567890",
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@example.com"
  }'
```

### 2. Monthly Subscriptions

Courses can be billed monthly:

```bash
# Add monthly subscription course to cart
curl -X POST http://localhost:8090/api/v1/cart/add \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "courseId": "course-uuid",
    "billingType": "MONTHLY"
  }'

# View subscriptions
curl -X GET http://localhost:8090/api/v1/subscriptions \
  -H "Authorization: Bearer <token>"

# Cancel subscription
curl -X POST http://localhost:8090/api/v1/subscriptions/cancel \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriptionId": "sub-uuid"
  }'
```

### 3. Automatic Billing

The subscription billing job runs automatically every 24 hours (configurable in `bootstrap/container.go`).

For testing, you can change the interval:

```go
// In bootstrap/container.go, line ~180
go c.BillingJob.StartScheduler(c.jobCtx, 1*time.Hour) // Run every hour for testing
```

Or trigger manually:

```go
// Add a test endpoint in your code
func (h *SubscriptionHandler) TriggerBilling(c *fiber.Ctx) error {
    err := h.billingJob.Run(c.Context())
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.JSON(fiber.Map{"message": "Billing job completed"})
}
```

## API Endpoints

### Cart Endpoints

- `POST /api/v1/cart/add` - Add course to cart
- `POST /api/v1/cart/remove` - Remove course from cart
- `GET /api/v1/cart` - Get cart contents
- `DELETE /api/v1/cart/clear` - Clear cart
- `POST /api/v1/cart/checkout` - Checkout cart

### Subscription Endpoints

- `GET /api/v1/subscriptions` - Get user subscriptions
- `GET /api/v1/subscriptions/:id` - Get subscription details
- `POST /api/v1/subscriptions/cancel` - Cancel subscription

### Payment Endpoints (Legacy)

- `POST /api/v1/payments/create` - Create single course payment
- `GET /api/v1/payments/:id/status` - Get payment status

### Webhook

- `POST /api/v1/webhooks/paymob` - Paymob payment webhook

## Background Jobs

### Subscription Billing Job

**Purpose**: Process monthly subscription renewals

**Schedule**: Runs every 24 hours by default

**What it does**:
1. Finds subscriptions due for billing
2. Checks if user has stored payment method
3. Attempts automatic charge (if implemented) or creates manual payment
4. Sends email notification to user with payment link
5. Updates subscription billing dates

**Configuration**:
- Interval: `bootstrap/container.go` line ~180
- Email templates: `infrastructure/notification/email_service.go`

## Email Notifications

The service sends emails for:

1. **Subscription Renewal**: When monthly payment is due
2. **Payment Receipt**: After successful payment
3. **Subscription Cancellation**: Confirmation of cancellation

**Email Provider:** Resend (same as auth service)

**Configuration:**
```env
RESEND_API_KEY=re_your_api_key
EMAIL_FROM=onboarding@resend.dev
EMAIL_FROM_NAME=Payment Service
```

Email templates are HTML-based and customizable in `infrastructure/notification/email_service.go`.

See `RESEND_SETUP.md` for detailed setup instructions.

## Stored Payment Methods (Future Enhancement)

The database schema supports storing tokenized payment methods for automatic billing:

```sql
-- payment_methods table
- token: Tokenized payment method from Paymob
- last_four: Last 4 digits of card
- card_brand: Visa, Mastercard, etc.
- is_default: Default payment method for user
```

To implement automatic charging:
1. Integrate Paymob's card tokenization API
2. Store token after first successful payment
3. Use token for subsequent subscription renewals
4. Update `jobs/subscription_billing.go` to charge stored method

## Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run integration tests
go test -tags=integration ./tests/...
```

### Manual Testing

1. Add courses to cart with different billing types
2. Checkout cart
3. Complete payment via Paymob
4. Verify enrollments are activated
5. Check subscriptions are created for monthly items
6. Wait for billing job or trigger manually
7. Verify renewal emails are sent

## Monitoring

### Logs

The service logs important events:
- Payment processing
- Subscription renewals
- Email sending
- Errors and failures

### Metrics (TODO)

Consider adding:
- Prometheus metrics for payment success/failure rates
- Subscription churn rate
- Cart abandonment rate

## Troubleshooting

### Emails Not Sending

1. Check Resend API key in `.env`
2. Verify API key is valid (starts with `re_`)
3. Check Resend dashboard logs: https://resend.com/logs
4. Verify recipient email is valid
5. Check spam folder
6. If Resend not configured, emails are logged to console

See `RESEND_SETUP.md` for detailed troubleshooting.

### Subscriptions Not Renewing

1. Check billing job is running (look for log: "Starting subscription billing scheduler")
2. Verify subscriptions exist: `SELECT * FROM subscriptions WHERE status = 'ACTIVE'`
3. Check next_billing_date is in the past
4. Look for errors in logs
5. Manually trigger job for testing

### Cart Checkout Fails

1. Verify courses exist in courses service
2. Check user is not already enrolled
3. Verify Paymob credentials
4. Check Redis connection
5. Look for errors in logs

### Database Migration Issues

```bash
# Check current migration version
SELECT version FROM schema_migrations;

# Rollback if needed
migrate -path migrations -database "postgresql://..." down 1

# Re-run migrations
migrate -path migrations -database "postgresql://..." up
```

## Production Deployment

### Checklist

- [ ] Set strong `INTERNAL_SERVICE_SECRET`
- [ ] Configure production database with SSL
- [ ] Set up Redis with password
- [ ] Configure production SMTP
- [ ] Set up monitoring and alerting
- [ ] Configure log aggregation
- [ ] Set up database backups
- [ ] Test webhook endpoint is publicly accessible
- [ ] Configure rate limiting
- [ ] Set up health checks
- [ ] Review and adjust billing job interval

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o payment-service cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/payment-service .
COPY --from=builder /app/.env .
EXPOSE 8090
CMD ["./payment-service"]
```

```bash
docker build -t payment-service .
docker run -p 8090:8090 --env-file .env payment-service
```

## Support

For issues or questions:
1. Check logs for error messages
2. Review this guide and CART_AND_SUBSCRIPTIONS.md
3. Check Paymob documentation
4. Contact the development team

## Next Steps

1. Implement automatic charging with stored payment methods
2. Add user email/phone lookup from auth service
3. Add course name lookup from courses service
4. Implement payment retry logic for failed subscriptions
5. Add subscription pause/resume functionality
6. Implement proration for mid-cycle changes
7. Add discount/coupon system
8. Set up monitoring and alerting
