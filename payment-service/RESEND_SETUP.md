# Resend Email Setup Guide

The payment service uses [Resend](https://resend.com) for sending transactional emails, matching the configuration used in the auth service.

## Why Resend?

- ✅ Simple API (no SMTP configuration needed)
- ✅ High deliverability rates
- ✅ Built-in email tracking
- ✅ Generous free tier (100 emails/day)
- ✅ Easy to test and debug
- ✅ Consistent with auth service

## Setup Steps

### 1. Create Resend Account

1. Go to [resend.com](https://resend.com)
2. Sign up for a free account
3. Verify your email address

### 2. Get API Key

1. Go to [API Keys](https://resend.com/api-keys)
2. Click "Create API Key"
3. Give it a name (e.g., "Payment Service")
4. Select permissions: "Sending access"
5. Copy the API key (starts with `re_`)

### 3. Configure Domain (Optional but Recommended)

**For Production:**
1. Go to [Domains](https://resend.com/domains)
2. Click "Add Domain"
3. Enter your domain (e.g., `yourdomain.com`)
4. Add the DNS records to your domain provider
5. Wait for verification (usually a few minutes)

**For Development:**
- Use the default `onboarding@resend.dev` email
- This works for testing but has limitations

### 4. Update Environment Variables

Edit your `.env` file:

```env
# Email Configuration (Resend)
RESEND_API_KEY=re_your_actual_api_key_here
EMAIL_FROM=onboarding@resend.dev
EMAIL_FROM_NAME=Payment Service
```

**For Production with Custom Domain:**
```env
RESEND_API_KEY=re_your_actual_api_key_here
EMAIL_FROM=noreply@yourdomain.com
EMAIL_FROM_NAME=Your Company Name
```

## Email Types Sent

### 1. Subscription Renewal Notification
**Trigger:** Billing job runs daily at 2 AM
**Recipient:** Student with active subscription
**Content:**
- Course name
- Amount due
- Next billing date
- Payment link
- Subscription ID

**Example:**
```
Subject: Your Subscription Renewal is Due

Hi Ahmed,

Your monthly subscription for "Advanced React Development" is due for renewal.

Amount: 500.00 EGP
Next Billing Date: May 2, 2026

[Pay Now Button]

If you wish to cancel your subscription, you can do so from your account settings.
```

### 2. Payment Receipt
**Trigger:** Successful payment webhook
**Recipient:** Student who completed payment
**Content:**
- Order ID
- Payment date
- Amount paid
- List of courses purchased

**Example:**
```
Subject: Payment Receipt - Order Confirmation

Hi Ahmed,

Your payment has been processed successfully!

Order ID: order-uuid-123
Payment Date: April 2, 2026
Amount Paid: 1500.00 EGP

Courses:
- Advanced React Development
- Complete JavaScript Bootcamp

You now have full access to your enrolled courses. Happy learning!
```

### 3. Subscription Cancellation Confirmation
**Trigger:** Student cancels subscription
**Recipient:** Student who cancelled
**Content:**
- Course name
- Subscription ID
- Access until period ends

**Example:**
```
Subject: Subscription Cancelled

Hi Ahmed,

Your subscription for "Advanced React Development" has been cancelled successfully.

You will continue to have access until the end of your current billing period.

Subscription ID: sub-uuid-111

We're sorry to see you go! If you change your mind, you can always resubscribe.
```

## Testing

### Test Email Sending

1. **Start the service:**
   ```bash
   cd payment-service
   go run cmd/server/main.go
   ```

2. **Trigger a test email:**
   - Create a subscription
   - Manually trigger billing job
   - Check your email inbox

3. **Check Resend Dashboard:**
   - Go to [Logs](https://resend.com/logs)
   - See all sent emails
   - View delivery status
   - Check email content

### Test Without API Key

If `RESEND_API_KEY` is not set, emails will be logged to console instead:

```
EMAIL (not sent - Resend API key not configured):
To: student@example.com
Subject: Your Subscription Renewal is Due
```

This is useful for local development without setting up Resend.

## API Usage

The email service uses Resend's REST API:

```go
POST https://api.resend.com/emails
Authorization: Bearer re_your_api_key
Content-Type: application/json

{
  "from": "Payment Service <onboarding@resend.dev>",
  "to": ["student@example.com"],
  "subject": "Your Subscription Renewal is Due",
  "html": "<html>...</html>"
}
```

## Rate Limits

### Free Tier:
- 100 emails per day
- 3,000 emails per month
- Perfect for development and small projects

### Paid Plans:
- Starting at $20/month for 50,000 emails
- See [pricing](https://resend.com/pricing)

## Troubleshooting

### Issue: "Resend API key missing"
**Solution:** Set `RESEND_API_KEY` in your `.env` file

### Issue: "Invalid API key"
**Solution:** 
- Check the API key is correct
- Make sure it starts with `re_`
- Regenerate key if needed

### Issue: "Email not delivered"
**Solution:**
1. Check Resend logs for delivery status
2. Check spam folder
3. Verify recipient email is valid
4. For custom domains, verify DNS records

### Issue: "Rate limit exceeded"
**Solution:**
- Upgrade to paid plan
- Reduce email frequency
- Use email batching

## Best Practices

### 1. Use Custom Domain in Production
- Better deliverability
- Professional appearance
- Brand consistency

### 2. Monitor Email Logs
- Check Resend dashboard regularly
- Set up webhooks for delivery events
- Track bounce rates

### 3. Test Email Templates
- Send test emails before deploying
- Check rendering on different email clients
- Verify all links work

### 4. Handle Failures Gracefully
- Log email failures
- Don't block critical operations
- Implement retry logic for important emails

### 5. Respect User Preferences
- Allow users to unsubscribe
- Honor email preferences
- Don't spam users

## Comparison with Auth Service

Both services use the same Resend configuration:

| Feature | Auth Service | Payment Service |
|---------|-------------|-----------------|
| Provider | Resend | Resend |
| API Key | `RESEND_API_KEY` | `RESEND_API_KEY` |
| From Email | `EMAIL_FROM` | `EMAIL_FROM` |
| From Name | Configurable | Configurable |
| Templates | OTP, Security Alerts | Receipts, Renewals |

## Migration from SMTP

If you were using SMTP before:

1. Remove SMTP environment variables:
   ```env
   # Remove these:
   SMTP_HOST=
   SMTP_PORT=
   SMTP_USERNAME=
   SMTP_PASSWORD=
   ```

2. Add Resend variables:
   ```env
   # Add these:
   RESEND_API_KEY=re_your_key
   EMAIL_FROM=onboarding@resend.dev
   EMAIL_FROM_NAME=Payment Service
   ```

3. Restart the service

## Support

- **Resend Documentation:** https://resend.com/docs
- **Resend Support:** support@resend.com
- **Status Page:** https://status.resend.com

## Security Notes

- ✅ API keys are transmitted over HTTPS
- ✅ Store API keys in environment variables (never in code)
- ✅ Use different API keys for dev/staging/production
- ✅ Rotate API keys periodically
- ✅ Revoke compromised keys immediately

---

That's it! Your payment service is now configured to send emails using Resend, matching your auth service setup. 🎉
